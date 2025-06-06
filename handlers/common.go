package handlers

import (
	"database/sql"

	"github.com/labstack/echo/v4"
)

// PageData struct for template rendering
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

// Helper to convert *sql.Rows to []map[string]interface{}
func RowsToMap(rows *sql.Rows) ([]map[string]interface{}, error) {
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			rowMap[colName] = *val
		}
		results = append(results, rowMap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Helper function to get logged in username
func GetLoggedInUsername(c echo.Context) string {
	cookie, err := c.Cookie("username")
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
} 