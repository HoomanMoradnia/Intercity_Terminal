package routes

import (
	"github.com/labstack/echo/v4"

	"SecureSignIn/handlers/auth"
	"SecureSignIn/handlers/dashboard"
	"SecureSignIn/handlers/middleware"
	"SecureSignIn/handlers/templates"
)

// RegisterRoutes registers all application routes
func RegisterRoutes(e *echo.Echo) {
	// Initialize templates
	templates.InitTemplates()
	
	// Middleware
	e.Use(middleware.LogAndRecover)

	// Static files
	e.Static("/static", "static")

	// Basic routes
	e.GET("/", dashboard.IndexHandler)
	e.GET("/health", dashboard.HealthCheckHandler)
	
	// Auth routes
	e.GET("/login", auth.LoginHandler)
	e.POST("/auth", auth.BasicAuthHandler)
	e.GET("/logout", auth.LogoutHandler)
	
	// Registration routes
	e.GET("/register", auth.RegisterHandler)
	e.POST("/register", auth.BasicRegisterHandler)
	
	// Password reset routes
	e.GET("/forgot", auth.ForgotHandler)
	e.POST("/forgot", auth.ForgotHandler)
	e.GET("/reset/:token", auth.ShowResetFormHandler)
	e.POST("/reset/:token", auth.HandleResetPasswordHandler)
	e.GET("/security-reset", auth.SecurityQuestionResetHandler)
	e.POST("/security-reset", auth.SecurityQuestionResetHandler)
	
	// Authenticated routes (requires login)
	e.GET("/dashboard", middleware.RequireLogin(dashboard.DashboardHandler))
	e.GET("/setup-security", middleware.RequireLogin(auth.SetupSecurityQuestionHandler))
	e.POST("/setup-security", middleware.RequireLogin(auth.SetupSecurityQuestionHandler))
	
	// Trip planning routes - accessible to all authenticated users
	e.GET("/trip-plan", middleware.RequireLogin(dashboard.DashboardHandler)) // Placeholder route
	
	// Role-specific routes
	// Operator routes
	operatorGroup := e.Group("/operator")
	operatorGroup.Use(middleware.RequireRole([]string{"Operator", "Manager", "Admin"}))
	operatorGroup.GET("/dashboard", dashboard.DashboardHandler)
	operatorGroup.GET("/trips", dashboard.OperatorGetTripByRouteHandler)
	operatorGroup.GET("/bookings", dashboard.OperatorGetBookingsHandler)
	operatorGroup.POST("/bookings/create", dashboard.OperatorCreateBookingHandler)
	
	// Manager routes
	managerGroup := e.Group("/manager")
	managerGroup.Use(middleware.RequireRole([]string{"Manager", "Admin"}))
	
	// Accountant routes
	accountantGroup := e.Group("/accountant")
	accountantGroup.Use(middleware.RequireRole([]string{"Accountant", "Admin"}))
	
	// Admin routes
	adminGroup := e.Group("/admin")
	adminGroup.Use(middleware.RequireRole([]string{"Admin"}))
	adminGroup.GET("/dashboard", dashboard.AdminDashboardHandler)
	adminGroup.GET("/users", dashboard.AdminUsersHandler)
	adminGroup.POST("/users/create", dashboard.AdminCreateUserHandler)
	adminGroup.POST("/users/update", dashboard.AdminUpdateUserHandler)
	adminGroup.DELETE("/users/:id", dashboard.AdminDeleteUserHandler)
	adminGroup.POST("/users/password", dashboard.AdminUpdatePasswordHandler)
	adminGroup.POST("/users/username", dashboard.AdminUpdateUsernameHandler)
	
	// Vehicle management routes
	adminGroup.GET("/vehicles", dashboard.AdminVehiclesHandler)
	adminGroup.GET("/vehicles/:id", dashboard.AdminGetVehicleByIDHandler)
	adminGroup.POST("/vehicles/create", dashboard.AdminCreateVehicleHandler)
	adminGroup.POST("/vehicles/update", dashboard.AdminUpdateVehicleHandler)
	adminGroup.DELETE("/vehicles/:id", dashboard.AdminDeleteVehicleHandler)
	
	// Trip management routes
	adminGroup.GET("/trips", dashboard.AdminTripsHandler)
	adminGroup.POST("/trips/create", dashboard.AdminCreateTripHandler)
	adminGroup.POST("/trips/update", dashboard.AdminUpdateTripHandler)
	adminGroup.DELETE("/trips/:id", dashboard.AdminDeleteTripHandler)
	adminGroup.GET("/trips/:id/capacity", dashboard.AdminTripCapacityHandler)
	
	// Booking management routes
	adminGroup.GET("/bookings", dashboard.AdminBookingsHandler)
	adminGroup.POST("/bookings/create", dashboard.AdminCreateBookingHandler)
	adminGroup.POST("/bookings/status", dashboard.AdminUpdateBookingStatusHandler)
	adminGroup.DELETE("/bookings/:id", dashboard.AdminDeleteBookingHandler)
	
	// Reports routes
	adminGroup.GET("/reports/data", dashboard.AdminReportsDataHandler)
	adminGroup.GET("/reports/export", dashboard.AdminReportsExportHandler)
	
	// Backup endpoints
	adminGroup.POST("/backup", dashboard.AdminBackupHandler)
	adminGroup.GET("/backup/download", dashboard.AdminBackupDownloadHandler)
} 