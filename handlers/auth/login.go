package auth

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"SecureSignIn/db"
	"SecureSignIn/handlers/templates"
	"SecureSignIn/models"
)

// LoginHandler - Render login page
func LoginHandler(c echo.Context) error {
	// Get error parameter
	errorMsg := c.QueryParam("error")

	// If error is about needing to be logged in, and we're already on the login page,
	// don't show this confusing error
	if errorMsg == "You must be logged in to access this page" {
		errorMsg = ""
	}

	data := models.PageData{
		Title:      "Login",
		ActivePage: "login",
		Success:    c.QueryParam("success"),
		Error:      errorMsg,
	}
	return templates.RenderTemplate(c, "login.html", data)
}

// AuthHandler - Process login form
func BasicAuthHandler(c echo.Context) error {
	usernameOrEmail := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")

	if usernameOrEmail == "" || password == "" {
		data := models.PageData{
			Title:      "Login",
			Error:      "Username/Email and password cannot be empty",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return templates.RenderTemplate(c, "login.html", data)
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
		data := models.PageData{
			Title:      "Login",
			Error:      "An error occurred while checking credentials. Please try again.",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return templates.RenderTemplate(c, "login.html", data)
	}

	var userID int64 // Use int64 for potential Postgres ID
	var storedUsername string
	var storedDOB string
	var storedSSN string
	var storedPassword string
	var storedEmail string
	var storedRole string

	// Scan user data (now both email and username lookups have the same structure)
	err = userRow.Scan(&userID, &storedUsername, &storedDOB, &storedSSN, &storedPassword, &storedEmail, &storedRole)

	if err != nil {
		if err == sql.ErrNoRows {
			// User not found
			data := models.PageData{
				Title:      "Login",
				Error:      "Invalid username/email or password",
				Username:   usernameOrEmail, // Preserve the input
				ActivePage: "login",
			}
			return templates.RenderTemplate(c, "login.html", data)
		}
		log.Printf("Error scanning user row: %v", err)
		data := models.PageData{
			Title:      "Login",
			Error:      "An error occurred while checking credentials. Please try again.",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return templates.RenderTemplate(c, "login.html", data)
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		// Log failed login attempt before returning error
		log.Printf("Login failed for user: %s (Invalid Password) from %s", usernameOrEmail, c.RealIP())
		data := models.PageData{
			Title:      "Login",
			Error:      "Invalid username/email or password",
			Username:   usernameOrEmail, // Preserve the input
			ActivePage: "login",
		}
		return templates.RenderTemplate(c, "login.html", data)
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

	// Set role cookie
	roleCookie := new(http.Cookie)
	roleCookie.Name = "user_role"
	roleCookie.Value = storedRole
	roleCookie.Expires = time.Now().Add(24 * time.Hour) // Cookie expires in 24 hours
	roleCookie.Path = "/"
	c.SetCookie(roleCookie)

	return c.Redirect(http.StatusSeeOther, "/dashboard?success=Successfully logged in&user="+storedUsername)
}

// LogoutHandler - Process logout
func LogoutHandler(c echo.Context) error {
	log.Printf("User logged out.")

	// Clear the username cookie to end the session
	cookie := new(http.Cookie)
	cookie.Name = "username"
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour) // Set expiration in the past to delete the cookie
	cookie.Path = "/"
	c.SetCookie(cookie)

	// Clear the role cookie
	roleCookie := new(http.Cookie)
	roleCookie.Name = "user_role"
	roleCookie.Value = ""
	roleCookie.Expires = time.Now().Add(-1 * time.Hour) // Set expiration in the past to delete the cookie
	roleCookie.Path = "/"
	c.SetCookie(roleCookie)

	return c.Redirect(http.StatusSeeOther, "/login?success=Successfully logged out.")
} 