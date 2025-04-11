package main

import (
	"SecureSignIn/db"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Middleware to recover from panics and log errors
func logAndRecover(handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic: %v\nStack trace:\n%s", err, string(debug.Stack()))
				httpError := echo.NewHTTPError(http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
				c.Error(httpError)
			}
		}()

		log.Printf("Request: %s %s from %s", c.Request().Method, c.Request().URL.Path, c.RealIP())
		err := handler(c)
		if err != nil {
			log.Printf("Handler error for %s %s: %v", c.Request().Method, c.Request().URL.Path, err)
			return err
		}
		log.Printf("Response %d sent for: %s %s", c.Response().Status, c.Request().Method, c.Request().URL.Path)
		return nil
	}
}

// Template cache
var templates = make(map[string]*template.Template)

// Load templates on init
func init() {
	templatesDir := "templates"
	log.Printf("Loading templates from: %s", templatesDir)

	layouts, err := filepath.Glob(filepath.Join(templatesDir, "base.html"))
	if err != nil || len(layouts) == 0 {
		log.Fatalf("Error loading base template: %v (found: %d)", err, len(layouts))
	}
	log.Printf("Found base template: %v", layouts)

	includes, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		log.Fatalf("Error finding template includes: %v", err)
	}
	log.Printf("Found templates: %v", includes)

	funcMap := template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}

	for _, include := range includes {
		if filepath.Base(include) == "base.html" {
			continue
		}

		files := append([]string{include}, layouts...)
		fileName := filepath.Base(include)
		log.Printf("Loading template: %s with files: %v", fileName, files)
		templates[fileName] = template.Must(template.New(fileName).Funcs(funcMap).ParseFiles(files...))
	}

	log.Printf("Templates loaded successfully. Count: %d", len(templates))
	if _, ok := templates["login.html"]; !ok {
		log.Fatalf("FATAL: login.html template not loaded correctly.")
	}
}

// Render a template given a model
func renderTemplate(c echo.Context, tmpl string, data interface{}) error {
	log.Printf("Attempting to render template: %s", tmpl)
	log.Printf("Template data: %+v", data)

	t, ok := templates[tmpl]
	if !ok {
		log.Printf("Template %s does not exist in map. Available templates: %v", tmpl, templates)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Template %s not found.", tmpl))
	}

	var buf strings.Builder
	err := t.ExecuteTemplate(&buf, "base", data)
	if err != nil {
		log.Printf("Error executing template %s: %v", tmpl, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error rendering page.")
	}

	return c.HTML(http.StatusOK, buf.String())
}

// Page data struct
type PageData struct {
	Title            string
	Error            string
	Success          string
	ActivePage       string
	Users            []map[string]interface{}
	LoginLogs        []map[string]interface{}
	IsLoggedIn       bool
	Username         string
	ResetToken       string
	Email            string
	ShowCodeInput    bool
	SecurityQuestion string
	QuestionID       int64
	UserID           int64
	ResetMethod      string // "email" or "security"
	HasSecurityQ     bool   // Whether the user has a security question set up
	// Fields for registration form data persistence
	DOB string
	SSN string
}

// --- Password Reset Token Store (In-Memory - Demo Only) ---
type ResetTokenInfo struct {
	UserID int
	Expiry time.Time
}

var (
	resetTokens = make(map[string]ResetTokenInfo)
	tokenMutex  sync.RWMutex
)

const resetTokenValidity = 15 * time.Minute // Token valid for 15 minutes

// generateResetToken creates a secure random token.
func generateResetToken() (string, error) {
	b := make([]byte, 16) // 128 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// storeResetToken stores a token for a user.
func storeResetToken(userID int) (string, error) {
	token, err := generateResetToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	resetTokens[token] = ResetTokenInfo{
		UserID: userID,
		Expiry: time.Now().Add(resetTokenValidity),
	}
	log.Printf("Stored reset token for user ID %d (expires %v)", userID, resetTokens[token].Expiry)
	return token, nil
}

// validateResetToken checks if a token is valid and returns the user ID.
func validateResetToken(token string) (int, bool) {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()
	info, exists := resetTokens[token]
	valid := exists && !time.Now().After(info.Expiry)

	if exists && !valid { // Token expired, remove it
		tokenMutex.Lock()
		delete(resetTokens, token)
		tokenMutex.Unlock()
		log.Printf("Reset token %s expired and removed.", token)
	}

	if valid {
		log.Printf("Validated reset token %s for user ID %d.", token, info.UserID)
	} else {
		log.Printf("Reset token %s invalid or not found.", token)
	}
	return info.UserID, valid
}

// invalidateResetToken removes a token from the store.
func invalidateResetToken(token string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	delete(resetTokens, token)
	log.Printf("Invalidated reset token %s.", token)
}

// --- End Token Store ---

// RegistrationForm represents the registration form data
type RegistrationForm struct {
	Username         string `form:"username"`
	Email            string `form:"email"`
	Password         string `form:"password"`
	ConfirmPassword  string `form:"confirmPassword"`
	DOB              string `form:"dob"`
	SSN              string `form:"ssn"`
	SecurityQuestion string `form:"security_question"`
	SecurityAnswer   string `form:"security_answer"`
}

// --- Handlers ---

// Index handler - Redirects appropriately
func indexHandler(c echo.Context) error {
	data := PageData{
		Title:      "Home",
		ActivePage: "home",
	}
	return renderTemplate(c, "index.html", data)
}

// Helper to convert *sql.Rows to []map[string]interface{}
func rowsToMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			rowMap[colName] = *val
		}
		results = append(results, rowMap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	return results, nil
}

// Dashboard handler - For logged in users (Simplified, assumes auth middleware sets user)
func dashboardHandler(c echo.Context) error {
	// Make sure user is logged in first by checking for username cookie
	cookie, err := c.Cookie("username")
	if err != nil || cookie.Value == "" {
		log.Printf("Dashboard access attempted without valid session")
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
	}

	username := cookie.Value
	log.Printf("Dashboard accessed by user: %s", username)

	// Fetch all users
	allUsers, err := db.GetAllUsers()
	if err != nil {
		log.Printf("Error getting all users: %v", err)
		return renderTemplate(c, "dashboard.html", PageData{
			Title:      "Dashboard",
			Error:      "Error retrieving user data",
			ActivePage: "dashboard",
			IsLoggedIn: true,
			Username:   username,
		})
	}
	userMaps, err := rowsToMap(allUsers)
	if err != nil {
		log.Printf("Error converting user rows to map: %v", err)
	}

	// Fetch login history
	loginHistory, err := db.GetLoginHistory()
	if err != nil {
		log.Printf("Error getting login history: %v", err)
		return renderTemplate(c, "dashboard.html", PageData{
			Title:      "Dashboard",
			Error:      "Error retrieving login history",
			ActivePage: "dashboard",
			IsLoggedIn: true,
			Username:   username,
		})
	}
	historyMaps, err := rowsToMap(loginHistory)
	if err != nil {
		log.Printf("Error converting login history rows to map: %v", err)
	}

	// Get the user ID for security question check
	var userID int64
	for _, user := range userMaps {
		if user["username"] == username {
			userID, _ = user["id"].(int64)
			break
		}
	}

	// Check if user has security question
	hasSecurityQ := false
	if userID > 0 {
		hasSecurityQ, err = db.HasSecurityQuestion(userID)
		if err != nil {
			log.Printf("Error checking security question: %v", err)
		}
	}

	return renderTemplate(c, "dashboard.html", PageData{
		Title:        "Dashboard",
		ActivePage:   "dashboard",
		Users:        userMaps,
		LoginLogs:    historyMaps,
		IsLoggedIn:   true,
		Username:     username,
		HasSecurityQ: hasSecurityQ,
	})
}

// Login handler - Render login page
func loginHandler(c echo.Context) error {
	// Get error parameter
	errorMsg := c.QueryParam("error")

	// If error is about needing to be logged in, and we're already on the login page,
	// don't show this confusing error
	if errorMsg == "You must be logged in to access this page" {
		errorMsg = ""
	}

	data := PageData{
		Title:      "Login",
		ActivePage: "login",
		Success:    c.QueryParam("success"),
		Error:      errorMsg,
	}
	return renderTemplate(c, "login.html", data)
}

// Auth handler - Process login form
func basicAuthHandler(c echo.Context) error {
	usernameOrEmail := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	if usernameOrEmail == "" || password == "" {
		data := PageData{
			Title:      "Login",
			Error:      "Username/Email and password cannot be empty",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return renderTemplate(c, "login.html", data)
	}

	var userRow *sql.Row
	var err error

	// Check if input is an email (contains @)
	isEmail := strings.Contains(usernameOrEmail, "@")

	if isEmail {
		// Get user by email
		userRow, err = db.GetUserByEmail(usernameOrEmail)
	} else {
		// Get user by username
		userRow, err = db.GetUserByUsername(usernameOrEmail)
	}

	if err != nil {
		log.Printf("Error getting user: %v", err)
		data := PageData{
			Title:      "Login",
			Error:      "An error occurred while checking credentials. Please try again.",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return renderTemplate(c, "login.html", data)
	}

	var userID int64 // Use int64 for potential Postgres ID
	var storedUsername string
	var storedDOB string
	var storedSSN string
	var storedPassword string
	var storedEmail string

	// Scan user data (now both email and username lookups have the same structure)
	err = userRow.Scan(&userID, &storedUsername, &storedDOB, &storedSSN, &storedPassword, &storedEmail)

	if err != nil {
		if err == sql.ErrNoRows {
			// User not found
			data := PageData{
				Title:      "Login",
				Error:      "Invalid username/email or password",
				Username:   usernameOrEmail, // Preserve the input
				ActivePage: "login",
			}
			return renderTemplate(c, "login.html", data)
		}
		log.Printf("Error scanning user row: %v", err)
		data := PageData{
			Title:      "Login",
			Error:      "An error occurred while checking credentials. Please try again.",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return renderTemplate(c, "login.html", data)
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		// Log failed login attempt before returning error
		log.Printf("Login failed for user: %s (Invalid Password) from %s", usernameOrEmail, c.RealIP())
		data := PageData{
			Title:      "Login",
			Error:      "Invalid username/email or password",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return renderTemplate(c, "login.html", data)
	}

	log.Printf("User %s (ID: %d) logged in successfully from %s", storedUsername, userID, c.RealIP())
	// Log successful login attempt
	if err := db.LogLoginAttempt(userID, c.RealIP(), true); err != nil {
		log.Printf("Warning: Failed to log successful login attempt for user ID %d: %v", userID, err)
		// Continue with login even if logging fails
	}

	// Set cookie to maintain session
	cookie := new(http.Cookie)
	cookie.Name = "username"
	cookie.Value = storedUsername
	cookie.Expires = time.Now().Add(24 * time.Hour) // Cookie expires in 24 hours
	cookie.Path = "/"
	c.SetCookie(cookie)

	return c.Redirect(http.StatusSeeOther, "/dashboard?success=Successfully logged in&user="+storedUsername)
}

// Register handler - Render registration page
func registerHandler(c echo.Context) error {
	data := PageData{
		Title:      "Register",
		ActivePage: "register",
		Email:      "",
		Username:   "",
		DOB:        "",
		SSN:        "",
	}
	return renderTemplate(c, "register.html", data)
}

// Generate a random 6-digit code
func generateVerificationCode() string {
	code := make([]byte, 3) // 3 bytes = 6 hex digits
	rand.Read(code)
	return fmt.Sprintf("%06x", code)[:6]
}

// Forgot Password handler - Render/Process forgot password form
func forgotHandler(c echo.Context) error {
	if c.Request().Method == "POST" {
		email := strings.TrimSpace(c.FormValue("email"))
		resetCode := c.FormValue("resetCode")
		newPassword := c.FormValue("newPassword")
		confirmPassword := c.FormValue("confirmPassword")

		log.Printf("Debug - Forgot password POST: email=%s, resetCode=%s, newPassword length=%d",
			email, resetCode, len(newPassword))

		// Validate email
		if email == "" {
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "Email is required",
				ActivePage: "forgot",
			})
		}

		// Get user by email
		row, err := db.GetUserByEmail(email)
		if err != nil {
			log.Printf("Error getting user by email: %v", err)
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "An error occurred. Please try again.",
				ActivePage: "forgot",
				Email:      email,
			})
		}

		var userID int64
		var username string
		var storedEmail string
		err = row.Scan(&userID, &username, &storedEmail)
		if err != nil {
			if err == sql.ErrNoRows {
				return renderTemplate(c, "forgot.html", PageData{
					Title:      "Forgot Password",
					Error:      "No account found with this email address",
					ActivePage: "forgot",
					Email:      email,
				})
			}
			log.Printf("Error scanning user data: %v", err)
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "An error occurred. Please try again.",
				ActivePage: "forgot",
				Email:      email,
			})
		}

		// If reset code is not provided, generate and send one
		if resetCode == "" {
			code := generateVerificationCode()
			expiresAt := time.Now().Add(15 * time.Minute)

			log.Printf("Debug - Generating new reset code '%s' for user ID %d", code, userID)

			err = db.StoreResetCode(userID, code, expiresAt)
			if err != nil {
				log.Printf("Error storing reset code: %v", err)
				return renderTemplate(c, "forgot.html", PageData{
					Title:      "Forgot Password",
					Error:      "An error occurred generating reset code. Please try again.",
					ActivePage: "forgot",
					Email:      email,
				})
			}

			// TODO: Send email with reset code
			// For now, we'll just show it on screen for testing
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Success:    fmt.Sprintf("Reset code sent to your email: %s (Code: %s)", email, code),
				ActivePage: "forgot",
				Email:      email,
			})
		}

		// Verify reset code and update password
		log.Printf("Debug - Validating reset code '%s'", resetCode)
		userID, err = db.ValidateResetCode(resetCode)
		if err != nil {
			log.Printf("Debug - Reset code validation failed: %v", err)
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "Invalid or expired reset code",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		log.Printf("Debug - Reset code validated successfully for user ID %d", userID)

		// Validate new password
		if newPassword == "" || newPassword != confirmPassword {
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "Passwords do not match",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		if len(newPassword) < 8 {
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "Password must be at least 8 characters long",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "An error occurred. Please try again.",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		// Update password and mark reset code as used
		err = db.UpdateUserPassword(int(userID), string(hashedPassword))
		if err != nil {
			log.Printf("Error updating password: %v", err)
			return renderTemplate(c, "forgot.html", PageData{
				Title:      "Forgot Password",
				Error:      "An error occurred updating password. Please try again.",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		err = db.MarkResetCodeUsed(resetCode)
		if err != nil {
			log.Printf("Error marking reset code as used: %v", err)
			// Non-critical error, continue
		}

		return c.Redirect(http.StatusSeeOther, "/login?success=Password reset successful. Please log in with your new password.")
	}

	// GET request - show form
	return renderTemplate(c, "forgot.html", PageData{
		Title:      "Forgot Password",
		ActivePage: "forgot",
		Email:      "", // Initialize Email field to empty string
	})
}

// Show Reset Password Form handler
func showResetFormHandler(c echo.Context) error {
	token := c.Param("token")
	userID, valid := validateResetToken(token)

	if !valid {
		log.Printf("Invalid or expired reset token presented: %s", token)
		return c.Redirect(http.StatusSeeOther, "/forgot?error=Invalid or expired reset link.")
	}

	log.Printf("Showing password reset form for valid token %s (User ID: %d)", token, userID)
	data := PageData{
		Title:      "Reset Password",
		ActivePage: "reset",
		ResetToken: token,
	}
	return renderTemplate(c, "reset_password.html", data)
}

// Handle Reset Password Submission handler
func handleResetPasswordHandler(c echo.Context) error {
	token := c.Param("token")
	newPassword := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	userID, valid := validateResetToken(token)
	if !valid {
		log.Printf("Password reset attempt with invalid/expired token: %s", token)
		return c.Redirect(http.StatusSeeOther, "/forgot?error=Invalid or expired reset link.")
	}

	if newPassword == "" || newPassword != confirmPassword {
		log.Printf("Password reset failed for token %s: Passwords do not match or are empty.", token)
		data := PageData{
			Title:      "Reset Password",
			Error:      "Passwords do not match or are empty.",
			ResetToken: token,
		}
		return renderTemplate(c, "reset_password.html", data)
	}
	if len(newPassword) < 8 {
		data := PageData{
			Title:      "Reset Password",
			Error:      "Password must be at least 8 characters long.",
			ResetToken: token,
		}
		return renderTemplate(c, "reset_password.html", data)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing new password for user ID %d: %v", userID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error processing password reset.")
	}

	err = db.UpdateUserPassword(userID, string(hashedPassword))
	if err != nil {
		log.Printf("Error updating password in DB for user ID %d: %v", userID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password.")
	}

	invalidateResetToken(token)

	log.Printf("Password successfully reset for user ID %d using token %s", userID, token)
	return c.Redirect(http.StatusSeeOther, "/login?success=Password successfully reset. Please log in.")
}

// Health Check handler
func healthCheckHandler(c echo.Context) error {
	if err := db.DB.Ping(); err != nil {
		log.Printf("Health check failed: DB ping error: %v", err)
		return c.String(http.StatusServiceUnavailable, "Database connection failed")
	}
	return c.String(http.StatusOK, "OK")
}

// Logout handler (simple redirect for demo)
func logoutHandler(c echo.Context) error {
	log.Printf("User logged out.")

	// Clear the username cookie to end the session
	cookie := new(http.Cookie)
	cookie.Name = "username"
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour) // Set expiration in the past to delete the cookie
	cookie.Path = "/"
	c.SetCookie(cookie)

	return c.Redirect(http.StatusSeeOther, "/login?success=Successfully logged out.")
}

// Simple validation helpers (replace with a proper validation library if needed)
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// Basic Register Handler - Process registration form
func basicRegisterHandler(c echo.Context) error {
	// For GET requests, just show the form
	if c.Request().Method == "GET" {
		log.Printf("GET request for registration form")
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			ActivePage: "register",
		})
	}

	log.Printf("Processing POST request for registration")

	// Force parse form - critical for reading form values
	err := c.Request().ParseForm()
	if err != nil {
		log.Printf("ERROR parsing form: %v", err)
	}

	// Log all form data for debugging
	log.Printf("Form data: %+v", c.Request().Form)
	log.Printf("PostForm data: %+v", c.Request().PostForm)

	// Direct check for specific form fields
	form := c.Request().Form
	log.Printf("Username in form: %v", form.Get("username"))
	log.Printf("Email in form: %v", form.Get("email"))
	log.Printf("DOB in form: %v", form.Get("dob"))
	log.Printf("SSN in form: %v", form.Get("ssn"))
	log.Printf("Password in form: %v", len(form.Get("password")) > 0)
	log.Printf("ConfirmPassword in form: %v", len(form.Get("confirmPassword")) > 0)
	log.Printf("Security Question in form: %v", form.Get("security_question"))
	log.Printf("Security Answer in form: %v", len(form.Get("security_answer")) > 0)

	// Extract form values
	username := strings.TrimSpace(c.FormValue("username"))
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirmPassword")
	dob := strings.TrimSpace(c.FormValue("dob"))
	ssn := strings.TrimSpace(c.FormValue("ssn"))
	securityQuestion := strings.TrimSpace(c.FormValue("security_question"))
	securityAnswer := strings.TrimSpace(c.FormValue("security_answer"))

	log.Printf("Extracted values - username: [%s], email: [%s], dob: [%s], ssn: [%s], password length: %d, confirmPassword length: %d, security question: [%s], security answer present: %v",
		username, email, dob, ssn, len(password), len(confirmPassword), securityQuestion, securityAnswer != "")

	// Detailed validation - individually check each field
	missingFields := []string{}
	if username == "" {
		missingFields = append(missingFields, "username")
	}
	if email == "" {
		missingFields = append(missingFields, "email")
	}
	if password == "" {
		missingFields = append(missingFields, "password")
	}
	if confirmPassword == "" {
		missingFields = append(missingFields, "confirmPassword")
	}
	if dob == "" {
		missingFields = append(missingFields, "dob")
	}
	if ssn == "" {
		missingFields = append(missingFields, "ssn")
	}
	if securityQuestion == "" {
		missingFields = append(missingFields, "security_question")
	}
	if securityAnswer == "" {
		missingFields = append(missingFields, "security_answer")
	}

	if len(missingFields) > 0 {
		log.Printf("ERROR: Missing fields: %v", missingFields)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "All fields are required",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Validate email format
	if !isValidEmail(email) {
		log.Printf("ERROR: Invalid email format: %s", email)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Please enter a valid email address",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Check if email already exists
	emailExists, err := db.CheckEmailExists(email)
	if err != nil {
		log.Printf("ERROR: Failed to check email existence: %v", err)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "An error occurred while processing your registration. Please try again.",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	if emailExists {
		log.Printf("ERROR: Email %s already in use", email)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Email address is already registered. Please use a different email or try to reset your password.",
			ActivePage: "register",
			Username:   username,
			Email:      "", // Clear the email to prevent duplicate submission attempts
			DOB:        dob,
			SSN:        ssn,
		})
	}

	if password != confirmPassword {
		log.Printf("ERROR: Passwords do not match")
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Passwords do not match",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Error processing password",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Hash security answer (always convert to lowercase for case-insensitive comparison later)
	hashedSecurityAnswer, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(securityAnswer)), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash security answer: %v", err)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Error processing security answer",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Save to database
	log.Printf("Attempting database insert")
	query := "INSERT INTO users (username, email, password, date_of_birth, social_security) VALUES (?, ?, ?, ?, ?)"
	result, err := db.DB.Exec(query, username, email, string(hashedPassword), dob, ssn)
	if err != nil {
		log.Printf("ERROR: Database error: %v", err)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Database error: " + err.Error(),
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("ERROR: Failed to get lastInsertId: %v", err)
		return renderTemplate(c, "register.html", PageData{
			Title:      "Register",
			Error:      "Database error: " + err.Error(),
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Add security question
	err = db.AddSecurityQuestion(userID, securityQuestion, string(hashedSecurityAnswer))
	if err != nil {
		log.Printf("ERROR: Failed to save security question: %v", err)
		// Don't block registration if security question fails
		// But log the error
	}

	log.Printf("SUCCESS: User registered with ID: %d", userID)
	return c.Redirect(http.StatusSeeOther, "/login?success=Registration successful! Please log in.")
}

// --- Security Questions Reset Handler ---
func securityQuestionResetHandler(c echo.Context) error {
	if c.Request().Method == "POST" {
		username := strings.TrimSpace(c.FormValue("username"))
		questionID := c.FormValue("question_id")
		userID := c.FormValue("user_id")
		answer := c.FormValue("security_answer")
		newPassword := c.FormValue("newPassword")
		confirmPassword := c.FormValue("confirmPassword")

		log.Printf("Debug - Security question reset POST: username=%s, questionID=%s, userID=%s, answer provided=%v",
			username, questionID, userID, answer != "")

		// If we only have username, find the user and their security question
		if username != "" && questionID == "" && userID == "" {
			// Get user by username
			userRow, err := db.GetUserByUsername(username)
			if err != nil {
				log.Printf("Error getting user by username: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			var userIDInt int64
			var storedUsername string
			var storedDOB string
			var storedSSN string
			var storedPassword string
			var storedEmail string

			err = userRow.Scan(&userIDInt, &storedUsername, &storedDOB, &storedSSN, &storedPassword, &storedEmail)
			if err != nil {
				if err == sql.ErrNoRows {
					return renderTemplate(c, "security_reset.html", PageData{
						Title:      "Reset Password",
						Error:      "No account found with this username",
						ActivePage: "forgot",
						Username:   username,
					})
				}
				log.Printf("Error scanning user data: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			// Check if user has security question
			hasSecurityQ, err := db.HasSecurityQuestion(userIDInt)
			if err != nil {
				log.Printf("Error checking for security question: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			if !hasSecurityQ {
				return renderTemplate(c, "security_reset.html", PageData{
					Title:        "Reset Password",
					Error:        "This account doesn't have a security question set up. Please use email reset instead.",
					ActivePage:   "forgot",
					Username:     username,
					HasSecurityQ: false,
				})
			}

			// Get security question
			questionIDInt, question, _, err := db.GetSecurityQuestionByUserID(userIDInt)
			if err != nil {
				log.Printf("Error getting security question: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			// Show the security question form
			return renderTemplate(c, "security_reset.html", PageData{
				Title:            "Reset Password",
				ActivePage:       "forgot",
				Username:         username,
				SecurityQuestion: question,
				QuestionID:       questionIDInt,
				UserID:           userIDInt,
				HasSecurityQ:     true,
			})
		}

		// If we have the security answer and new password, process the password reset
		if answer != "" && userID != "" && questionID != "" && newPassword != "" && confirmPassword != "" {
			userIDInt, err := strconv.ParseInt(userID, 10, 64)
			if err != nil {
				log.Printf("Error parsing user ID: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "Invalid request. Please try again.",
					ActivePage: "forgot",
				})
			}

			questionIDInt, err := strconv.ParseInt(questionID, 10, 64)
			if err != nil {
				log.Printf("Error parsing question ID: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "Invalid request. Please try again.",
					ActivePage: "forgot",
				})
			}

			// Get security question to re-display if needed
			_, question, answerHash, err := db.GetSecurityQuestionByUserID(userIDInt)
			if err != nil {
				log.Printf("Error getting security question: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
				})
			}

			// Verify passwords match
			if newPassword != confirmPassword {
				return renderTemplate(c, "security_reset.html", PageData{
					Title:            "Reset Password",
					Error:            "Passwords do not match",
					ActivePage:       "forgot",
					Username:         username,
					SecurityQuestion: question,
					QuestionID:       questionIDInt,
					UserID:           userIDInt,
					HasSecurityQ:     true,
				})
			}

			if len(newPassword) < 8 {
				return renderTemplate(c, "security_reset.html", PageData{
					Title:            "Reset Password",
					Error:            "Password must be at least 8 characters long",
					ActivePage:       "forgot",
					Username:         username,
					SecurityQuestion: question,
					QuestionID:       questionIDInt,
					UserID:           userIDInt,
					HasSecurityQ:     true,
				})
			}

			// Verify security answer
			err = bcrypt.CompareHashAndPassword([]byte(answerHash), []byte(strings.ToLower(strings.TrimSpace(answer))))
			if err != nil {
				log.Printf("Invalid security answer for user ID %d", userIDInt)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:            "Reset Password",
					Error:            "Incorrect answer to security question",
					ActivePage:       "forgot",
					Username:         username,
					SecurityQuestion: question,
					QuestionID:       questionIDInt,
					UserID:           userIDInt,
					HasSecurityQ:     true,
				})
			}

			// Hash new password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("Error hashing password: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:            "Reset Password",
					Error:            "An error occurred. Please try again.",
					ActivePage:       "forgot",
					Username:         username,
					SecurityQuestion: question,
					QuestionID:       questionIDInt,
					UserID:           userIDInt,
					HasSecurityQ:     true,
				})
			}

			// Update password
			err = db.UpdateUserPassword(int(userIDInt), string(hashedPassword))
			if err != nil {
				log.Printf("Error updating password: %v", err)
				return renderTemplate(c, "security_reset.html", PageData{
					Title:            "Reset Password",
					Error:            "An error occurred updating password. Please try again.",
					ActivePage:       "forgot",
					Username:         username,
					SecurityQuestion: question,
					QuestionID:       questionIDInt,
					UserID:           userIDInt,
					HasSecurityQ:     true,
				})
			}

			log.Printf("Password successfully reset for user ID %d using security question", userIDInt)
			return c.Redirect(http.StatusSeeOther, "/login?success=Password reset successful. Please log in with your new password.")
		}

		// If we reach here, something is wrong with the form data
		return renderTemplate(c, "security_reset.html", PageData{
			Title:      "Reset Password",
			Error:      "Invalid request. Please try again.",
			ActivePage: "forgot",
		})
	}

	// GET request - show initial form
	return renderTemplate(c, "security_reset.html", PageData{
		Title:      "Reset Password with Security Question",
		ActivePage: "forgot",
	})
}

// --- Security Question Management Handler ---
func setupSecurityQuestionHandler(c echo.Context) error {
	// Check if user is logged in using cookie
	cookie, err := c.Cookie("username")
	if err != nil || cookie.Value == "" {
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to set up security questions")
	}

	// Get user information using the username from cookie
	userRow, err := db.GetUserByUsername(cookie.Value)
	if err != nil {
		log.Printf("Error getting user by username: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user information")
	}

	var userID int64
	var username, dob, ssn, password, email string
	err = userRow.Scan(&userID, &username, &dob, &ssn, &password, &email)
	if err != nil {
		log.Printf("Error scanning user data: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error retrieving user information")
	}

	// Check if the user already has a security question
	hasSecurityQ, err := db.HasSecurityQuestion(userID)
	if err != nil {
		log.Printf("Error checking for security question: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error checking security question status")
	}

	var questionText, questionID string
	if hasSecurityQ {
		// Get the existing question
		qID, question, _, err := db.GetSecurityQuestionByUserID(userID)
		if err != nil {
			log.Printf("Error getting security question: %v", err)
		} else {
			questionText = question
			questionID = fmt.Sprintf("%d", qID)
		}
	}

	// Handle POST request (user submitting a new/updated security question)
	if c.Request().Method == "POST" {
		securityQuestion := strings.TrimSpace(c.FormValue("security_question"))
		securityAnswer := strings.TrimSpace(c.FormValue("security_answer"))

		if securityQuestion == "" || securityAnswer == "" {
			return renderTemplate(c, "setup_security.html", PageData{
				Title:            "Security Question",
				Error:            "Both security question and answer are required",
				ActivePage:       "setup_security",
				UserID:           userID,
				Username:         username,
				HasSecurityQ:     hasSecurityQ,
				SecurityQuestion: questionText,
				QuestionID:       userID,
				IsLoggedIn:       true,
			})
		}

		// Hash the security answer
		hashedSecurityAnswer, err := bcrypt.GenerateFromPassword([]byte(strings.ToLower(securityAnswer)), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing security answer: %v", err)
			return renderTemplate(c, "setup_security.html", PageData{
				Title:            "Security Question",
				Error:            "Error processing security answer",
				ActivePage:       "setup_security",
				UserID:           userID,
				Username:         username,
				HasSecurityQ:     hasSecurityQ,
				SecurityQuestion: questionText,
				QuestionID:       userID,
				IsLoggedIn:       true,
			})
		}

		// Save or update the security question
		if hasSecurityQ {
			// Update existing question
			existingQID, err := strconv.ParseInt(questionID, 10, 64)
			if err != nil {
				existingQID, _, _, err = db.GetSecurityQuestionByUserID(userID)
				if err != nil {
					log.Printf("Error getting security question ID: %v", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "Error updating security question")
				}
			}

			err = db.UpdateSecurityQuestion(existingQID, securityQuestion, string(hashedSecurityAnswer))
			if err != nil {
				log.Printf("Error updating security question: %v", err)
				return renderTemplate(c, "setup_security.html", PageData{
					Title:            "Security Question",
					Error:            "Error updating security question",
					ActivePage:       "setup_security",
					UserID:           userID,
					Username:         username,
					HasSecurityQ:     hasSecurityQ,
					SecurityQuestion: questionText,
					QuestionID:       userID,
					IsLoggedIn:       true,
				})
			}

			return renderTemplate(c, "setup_security.html", PageData{
				Title:            "Security Question",
				Success:          "Security question updated successfully",
				ActivePage:       "setup_security",
				UserID:           userID,
				Username:         username,
				HasSecurityQ:     true,
				SecurityQuestion: securityQuestion,
				QuestionID:       userID,
				IsLoggedIn:       true,
			})
		} else {
			// Add new question
			err = db.AddSecurityQuestion(userID, securityQuestion, string(hashedSecurityAnswer))
			if err != nil {
				log.Printf("Error adding security question: %v", err)
				return renderTemplate(c, "setup_security.html", PageData{
					Title:        "Security Question",
					Error:        "Error saving security question",
					ActivePage:   "setup_security",
					UserID:       userID,
					Username:     username,
					HasSecurityQ: false,
					IsLoggedIn:   true,
				})
			}

			return renderTemplate(c, "setup_security.html", PageData{
				Title:            "Security Question",
				Success:          "Security question saved successfully",
				ActivePage:       "setup_security",
				UserID:           userID,
				Username:         username,
				HasSecurityQ:     true,
				SecurityQuestion: securityQuestion,
				QuestionID:       userID,
				IsLoggedIn:       true,
			})
		}
	}

	// Handle GET request (displaying the form)
	return renderTemplate(c, "setup_security.html", PageData{
		Title:            "Security Question",
		ActivePage:       "setup_security",
		UserID:           userID,
		Username:         username,
		HasSecurityQ:     hasSecurityQ,
		SecurityQuestion: questionText,
		QuestionID:       userID,
		IsLoggedIn:       true,
	})
}

// Helper function to get logged in username
func getLoggedInUsername(c echo.Context) string {
	cookie, err := c.Cookie("username")
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

// --- End Handlers ---
