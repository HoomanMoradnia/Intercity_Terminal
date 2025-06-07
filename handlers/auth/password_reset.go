package auth

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"SecureSignIn/db"
	"SecureSignIn/handlers/templates"
	"SecureSignIn/handlers/tokens"
	"SecureSignIn/models"
)

// ForgotHandler - Render/Process forgot password form
func ForgotHandler(c echo.Context) error {
	if c.Request().Method == "POST" {
		email := strings.TrimSpace(c.FormValue("email"))
		resetCode := c.FormValue("resetCode")
		newPassword := c.FormValue("newPassword")
		confirmPassword := c.FormValue("confirmPassword")

		log.Printf("Debug - Forgot password POST: email=%s, resetCode=%s, newPassword length=%d",
			email, resetCode, len(newPassword))

		// Validate email
		if email == "" {
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
				Title:      "Forgot Password",
				Error:      "Email is required",
				ActivePage: "forgot",
			})
		}

		// Get user by email
		row, err := db.GetUserByEmail(email)
		if err != nil {
			log.Printf("Error getting user by email: %v", err)
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
				return templates.RenderTemplate(c, "forgot.html", models.PageData{
					Title:      "Forgot Password",
					Error:      "No account found with this email address",
					ActivePage: "forgot",
					Email:      email,
				})
			}
			log.Printf("Error scanning user data: %v", err)
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
				Title:      "Forgot Password",
				Error:      "An error occurred. Please try again.",
				ActivePage: "forgot",
				Email:      email,
			})
		}

		// If reset code is not provided, generate and send one
		if resetCode == "" {
			code := tokens.GenerateVerificationCode()
			expiresAt := time.Now().Add(15 * time.Minute)

			log.Printf("Debug - Generating new reset code '%s' for user ID %d", code, userID)

			err = db.StoreResetCode(userID, code, expiresAt)
			if err != nil {
				log.Printf("Error storing reset code: %v", err)
				return templates.RenderTemplate(c, "forgot.html", models.PageData{
					Title:      "Forgot Password",
					Error:      "An error occurred generating reset code. Please try again.",
					ActivePage: "forgot",
					Email:      email,
				})
			}

			// TODO: Send email with reset code
			// For now, we'll just show it on screen for testing
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
				Title:      "Forgot Password",
				Error:      "Passwords do not match",
				ActivePage: "forgot",
				Email:      email,
				Success:    "true", // Keep showing the reset code form
			})
		}

		if len(newPassword) < 8 {
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
			return templates.RenderTemplate(c, "forgot.html", models.PageData{
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
	return templates.RenderTemplate(c, "forgot.html", models.PageData{
		Title:      "Forgot Password",
		ActivePage: "forgot",
		Email:      "", // Initialize Email field to empty string
	})
}

// ShowResetFormHandler - Show reset password form
func ShowResetFormHandler(c echo.Context) error {
	token := c.Param("token")
	userID, valid := tokens.ValidateResetToken(token)

	if !valid {
		log.Printf("Invalid or expired reset token presented: %s", token)
		return c.Redirect(http.StatusSeeOther, "/forgot?error=Invalid or expired reset link.")
	}

	log.Printf("Showing password reset form for valid token %s (User ID: %d)", token, userID)
	data := models.PageData{
		Title:      "Reset Password",
		ActivePage: "reset",
		ResetToken: token,
	}
	return templates.RenderTemplate(c, "reset_password.html", data)
}

// HandleResetPasswordHandler - Process reset password form
func HandleResetPasswordHandler(c echo.Context) error {
	token := c.Param("token")
	newPassword := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	userID, valid := tokens.ValidateResetToken(token)
	if !valid {
		log.Printf("Password reset attempt with invalid/expired token: %s", token)
		return c.Redirect(http.StatusSeeOther, "/forgot?error=Invalid or expired reset link.")
	}

	if newPassword == "" || newPassword != confirmPassword {
		log.Printf("Password reset failed for token %s: Passwords do not match or are empty.", token)
		data := models.PageData{
			Title:      "Reset Password",
			Error:      "Passwords do not match or are empty.",
			ResetToken: token,
		}
		return templates.RenderTemplate(c, "reset_password.html", data)
	}
	if len(newPassword) < 8 {
		data := models.PageData{
			Title:      "Reset Password",
			Error:      "Password must be at least 8 characters long.",
			ResetToken: token,
		}
		return templates.RenderTemplate(c, "reset_password.html", data)
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

	tokens.InvalidateResetToken(token)

	log.Printf("Password successfully reset for user ID %d using token %s", userID, token)
	return c.Redirect(http.StatusSeeOther, "/login?success=Password successfully reset. Please log in.")
}

// SecurityQuestionResetHandler - Reset password via security question
func SecurityQuestionResetHandler(c echo.Context) error {
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
					return templates.RenderTemplate(c, "security_reset.html", models.PageData{
						Title:      "Reset Password",
						Error:      "No account found with this username",
						ActivePage: "forgot",
						Username:   username,
					})
				}
				log.Printf("Error scanning user data: %v", err)
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			if !hasSecurityQ {
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
					Username:   username,
				})
			}

			// Show the security question form
			return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
					Title:      "Reset Password",
					Error:      "Invalid request. Please try again.",
					ActivePage: "forgot",
				})
			}

			questionIDInt, err := strconv.ParseInt(questionID, 10, 64)
			if err != nil {
				log.Printf("Error parsing question ID: %v", err)
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
					Title:      "Reset Password",
					Error:      "Invalid request. Please try again.",
					ActivePage: "forgot",
				})
			}

			// Get security question to re-display if needed
			_, question, answerHash, err := db.GetSecurityQuestionByUserID(userIDInt)
			if err != nil {
				log.Printf("Error getting security question: %v", err)
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
					Title:      "Reset Password",
					Error:      "An error occurred. Please try again.",
					ActivePage: "forgot",
				})
			}

			// Verify passwords match
			if newPassword != confirmPassword {
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
				return templates.RenderTemplate(c, "security_reset.html", models.PageData{
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
		return templates.RenderTemplate(c, "security_reset.html", models.PageData{
			Title:      "Reset Password",
			Error:      "Invalid request. Please try again.",
			ActivePage: "forgot",
		})
	}

	// GET request - show initial form
	return templates.RenderTemplate(c, "security_reset.html", models.PageData{
		Title:      "Reset Password with Security Question",
		ActivePage: "forgot",
	})
}

// SetupSecurityQuestionHandler - Manage security questions
func SetupSecurityQuestionHandler(c echo.Context) error {
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
	var username, dob, ssn, password, email, userRole string
	err = userRow.Scan(&userID, &username, &dob, &ssn, &password, &email, &userRole)
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
			return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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
			return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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
				return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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

			return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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
				return templates.RenderTemplate(c, "setup_security.html", models.PageData{
					Title:        "Security Question",
					Error:        "Error saving security question",
					ActivePage:   "setup_security",
					UserID:       userID,
					Username:     username,
					HasSecurityQ: false,
					IsLoggedIn:   true,
				})
			}

			return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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
	return templates.RenderTemplate(c, "setup_security.html", models.PageData{
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