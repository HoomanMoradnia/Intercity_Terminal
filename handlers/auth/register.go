package auth

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"SecureSignIn/db"
	"SecureSignIn/handlers/templates"
	"SecureSignIn/handlers/utils"
	"SecureSignIn/models"
)

// RegisterHandler - Render registration page
func RegisterHandler(c echo.Context) error {
	data := models.PageData{
		Title:      "Register",
		ActivePage: "register",
		Email:      "",
		Username:   "",
		DOB:        "",
		SSN:        "",
	}
	return templates.RenderTemplate(c, "register.html", data)
}

// BasicRegisterHandler - Process registration form
func BasicRegisterHandler(c echo.Context) error {
	// For GET requests, just show the form
	if c.Request().Method == "GET" {
		log.Printf("GET request for registration form")
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
	if !utils.IsValidEmail(email) {
		log.Printf("ERROR: Invalid email format: %s", email)
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
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
		return templates.RenderTemplate(c, "register.html", models.PageData{
			Title:      "Register",
			Error:      "Error processing security answer",
			ActivePage: "register",
			Username:   username,
			Email:      email,
			DOB:        dob,
			SSN:        ssn,
		})
	}

	// Add user to database
	userID, err := db.AddUser(username, string(hashedPassword), dob, ssn, email, "Operator")
	if err != nil {
		log.Printf("ERROR: Failed to add user: %v", err)
		return templates.RenderTemplate(c, "register.html", models.PageData{
			Title:      "Register",
			Error:      "Failed to create user account. Please try again.",
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