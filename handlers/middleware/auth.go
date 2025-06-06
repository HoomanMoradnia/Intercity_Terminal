package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// RequireLogin middleware checks if the user is logged in
func RequireLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("username")
		if err != nil || cookie.Value == "" {
			log.Printf("Access denied: No valid session cookie found for %s", c.Path())
			return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
		}
		return next(c)
	}
}

// RequireRole middleware checks if the user has the required role
func RequireRole(roles []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if user is logged in
			usernameCookie, err := c.Cookie("username")
			if err != nil || usernameCookie.Value == "" {
				log.Printf("Access denied: No valid session cookie found for %s", c.Path())
				return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
			}

			// Check if user has the required role
			roleCookie, err := c.Cookie("user_role")
			if err != nil || roleCookie.Value == "" {
				log.Printf("Access denied: No valid role cookie found for %s (user: %s)", c.Path(), usernameCookie.Value)
				return c.Redirect(http.StatusSeeOther, "/dashboard?error=You do not have the required permissions")
			}

			// Check if user's role is in the allowed roles
			hasRequiredRole := false
			userRole := roleCookie.Value
			for _, role := range roles {
				if strings.EqualFold(userRole, role) {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				log.Printf("Access denied: User %s has role %s but needs one of %v for %s", 
					usernameCookie.Value, userRole, roles, c.Path())
				return c.Redirect(http.StatusSeeOther, "/dashboard?error=You do not have the required permissions")
			}

			// Set the user role in the context
			c.Set("user_role", userRole)
			return next(c)
		}
	}
} 