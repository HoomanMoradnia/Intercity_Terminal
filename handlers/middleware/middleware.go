package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

// LogAndRecover is a middleware to recover from panics and log errors
func LogAndRecover(handler echo.HandlerFunc) echo.HandlerFunc {
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