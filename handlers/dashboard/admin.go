package dashboard

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"SecureSignIn/db"
	"SecureSignIn/handlers/templates"
	"SecureSignIn/models"
	"SecureSignIn/utils"
)

// AdminDashboardHandler - Handler for admin dashboard
func AdminDashboardHandler(c echo.Context) error {
	// Make sure user is logged in and has admin role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Admin dashboard access attempted without valid session")
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
	}

	username := usernameCookie.Value
	
	// Check role
	roleCookie, err := c.Cookie("user_role")
	if err != nil || roleCookie.Value != "Admin" {
		log.Printf("Admin dashboard access attempted by non-admin user: %s", username)
		return c.Redirect(http.StatusSeeOther, "/dashboard?error=You do not have permission to access the admin dashboard")
	}

	log.Printf("Admin dashboard accessed by user: %s", username)

	// Prepare page data
	data := models.PageData{
		Title:        "Admin Dashboard",
		ActivePage:   "admin",
		IsLoggedIn:   true,
		Username:     username,
		UserRole:     "Admin",
		Success:      c.QueryParam("success"),
		Error:        c.QueryParam("error"),
	}

	return templates.RenderTemplate(c, "admin_dashboard.html", data)
}

// AdminUsersHandler - Handler for admin user management
func AdminUsersHandler(c echo.Context) error {
	// Make sure user is logged in and has admin role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Admin users page access attempted without valid session")
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
	}

	username := usernameCookie.Value
	
	// Check role
	roleCookie, err := c.Cookie("user_role")
	if err != nil || roleCookie.Value != "Admin" {
		log.Printf("Admin users page access attempted by non-admin user: %s", username)
		return c.Redirect(http.StatusSeeOther, "/dashboard?error=You do not have permission to access the admin users page")
	}

	// Fetch all users
	allUsers, err := db.GetAllUsers()
	if err != nil {
		log.Printf("Error getting all users: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve users"})
	}
	defer allUsers.Close()

	// Convert rows to array of maps
	var users []map[string]interface{}
	for allUsers.Next() {
		var id int64
		var usernameStr, email, passwordHash, createdAt, dateOfBirth, socialSecurity, role string
		if err := allUsers.Scan(&id, &usernameStr, &email, &passwordHash, &createdAt, &dateOfBirth, &socialSecurity, &role); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, map[string]interface{}{ 
			"id": id,
			"username": usernameStr,
			"email": email,
			"created_at": createdAt,
			"role": role,
		})
	}

	return c.JSON(http.StatusOK, users)
}

// ManagerUsersHandler - Handler for manager user management
func ManagerUsersHandler(c echo.Context) error {
	// Make sure user is logged in and has manager role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Manager users page access attempted without valid session")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in to access this page"})
	}
	username := usernameCookie.Value

	// Check role
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Manager" && roleCookie.Value != "Admin") {
		log.Printf("Manager users page access attempted by non-manager user: %s", username)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have permission to access the manager users page"})
	}

	// Fetch all users
	allUsers, err := db.GetAllUsers()
	if err != nil {
		log.Printf("Error getting all users: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve users"})
	}
	defer allUsers.Close()

	// Convert rows to array of maps
	var users []map[string]interface{}
	for allUsers.Next() {
		var id int64
		var usernameStr, email, passwordHash, createdAt, dateOfBirth, socialSecurity, role string
		if err := allUsers.Scan(&id, &usernameStr, &email, &passwordHash, &createdAt, &dateOfBirth, &socialSecurity, &role); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, map[string]interface{}{ 
			"id": id,
			"username": usernameStr,
			"email": email,
			"created_at": createdAt,
			"role": role,
		})
	}

	return c.JSON(http.StatusOK, users)
}

// AdminCreateUserHandler - Handler for creating new users
func AdminCreateUserHandler(c echo.Context) error {
	// Make sure user is logged in and has admin role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Admin create user attempted without valid session")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "You must be logged in to perform this action",
		})
	}
	
	// Check role
	roleCookie, err := c.Cookie("user_role")
	if err != nil || roleCookie.Value != "Admin" {
		log.Printf("Admin create user attempted by non-admin user: %s", usernameCookie.Value)
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You do not have permission to perform this action",
		})
	}

	// Parse JSON request
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Basic validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username, email, and password are required",
		})
	}

	// Validate username and password format
	if ok, msg := utils.IsValidUsername(req.Username); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}
	if ok, msg := utils.IsValidPassword(req.Password); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}

	// Check valid role
	if req.Role != "Operator" && req.Role != "Manager" && req.Role != "Accountant" && req.Role != "Admin" {
		req.Role = "Operator" // Default to Operator if invalid role
	}

	// Create user
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}
	
	// Add user to database
	userID, err := db.AddUser(req.Username, string(hashedPassword), "", "", req.Email, req.Role)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}
	
	log.Printf("User created successfully by admin. ID: %d, Username: %s, Role: %s", userID, req.Username, req.Role)
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User created successfully",
		"user_id": userID,
	})
}

// ManagerCreateUserHandler - Handler for creating new users by manager (cannot assign Admin role)
func ManagerCreateUserHandler(c echo.Context) error {
	// Make sure user is logged in and has manager role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Manager create user attempted without valid session")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in to perform this action"})
	}
	// Check role
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Manager" && roleCookie.Value != "Admin") {
		log.Printf("Manager create user attempted by non-manager user: %s", usernameCookie.Value)
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have permission to perform this action"})
	}

	// Parse JSON request
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Basic validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username, email, and password are required"})
	}

	// Validate username and password format
	if ok, msg := utils.IsValidUsername(req.Username); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}
	if ok, msg := utils.IsValidPassword(req.Password); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}

	// Restrict role: cannot assign Admin
	switch req.Role {
	case "Operator", "Manager", "Accountant":
	// allowed
	default:
		req.Role = "Operator"
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	// Add user to database
	userID, err := db.AddUser(req.Username, string(hashedPassword), "", "", req.Email, req.Role)
	if err != nil {
		log.Printf("Error creating user by manager: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	log.Printf("User created successfully by manager. ID: %d, Username: %s, Role: %s", userID, req.Username, req.Role)
	return c.JSON(http.StatusOK, map[string]interface{}{"message": "User created successfully", "user_id": userID})
}

// AdminUpdateUserHandler - Handler for updating a user's role
func AdminUpdateUserHandler(c echo.Context) error {
	// Ensure request body contains id and role
	var req struct {
		ID   int64  `json:"id"`
		Role string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	// Validate role
	switch req.Role {
	case "Operator", "Manager", "Accountant", "Admin":
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role"})
	}
	// Update role in database
	if err := db.UpdateUserRole(req.ID, req.Role); err != nil {
		log.Printf("Error updating role for user %d: %v", req.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user role"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Role updated successfully"})
}

// ManagerUpdateUserHandler - Handler for updating a user's role by manager (cannot assign Admin role)
func ManagerUpdateUserHandler(c echo.Context) error {
	// Ensure request body contains id and role
	var req struct {
		ID   int64  `json:"id"`
		Role string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	// Restrict role: cannot assign Admin
	switch req.Role {
	case "Operator", "Manager", "Accountant":
	// allowed
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role"})
	}
	// Update role in database
	if err := db.UpdateUserRole(req.ID, req.Role); err != nil {
		log.Printf("Error updating role for user %d by manager: %v", req.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user role"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Role updated successfully"})
}

// AdminDeleteUserHandler - Handler for deleting a user
func AdminDeleteUserHandler(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}
	if err := db.DeleteUser(userID); err != nil {
		log.Printf("Error deleting user %d: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete user"})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

// AdminUpdatePasswordHandler - Handler for updating a user's password
func AdminUpdatePasswordHandler(c echo.Context) error {
	// Ensure request body contains id and password
	var req struct {
		ID       int64  `json:"id"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Validate password
	if len(req.Password) < 6 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Password must be at least 6 characters"})
	}
	
	// Validate password format
	if ok, msg := utils.IsValidPassword(req.Password); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}
	
	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update password"})
	}
	
	// Update password in database
	if err := db.UpdateUserPassword(int(req.ID), string(hashedPassword)); err != nil {
		log.Printf("Error updating password for user %d: %v", req.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update password"})
	}
	
	log.Printf("Password updated successfully for user ID: %d", req.ID)
	return c.JSON(http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

// AdminUpdateUsernameHandler - Handler for updating a user's username
func AdminUpdateUsernameHandler(c echo.Context) error {
	// Ensure request body contains id and username
	var req struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Validate username
	if len(req.Username) < 3 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Username must be at least 3 characters"})
	}
	
	// Validate username format
	if ok, msg := utils.IsValidUsername(req.Username); !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": msg})
	}
	
	// Update username in database
	if err := db.UpdateUsername(req.ID, req.Username); err != nil {
		log.Printf("Error updating username for user %d: %v", req.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	log.Printf("Username updated successfully for user ID: %d to '%s'", req.ID, req.Username)
	return c.JSON(http.StatusOK, map[string]string{"message": "Username updated successfully"})
}

// --- Vehicle Management Handlers ---

// AdminVehiclesHandler - Handler for getting all vehicles
func AdminVehiclesHandler(c echo.Context) error {
	// Make sure user is logged in and has admin role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Admin vehicles page access attempted without valid session")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "You must be logged in to access this page",
		})
	}
	
	// Check role: allow Admin and Manager
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Admin" && roleCookie.Value != "Manager") {
		log.Printf("Admin vehicles page access attempted by unauthorized user: %s", usernameCookie.Value)
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You do not have permission to access the vehicles page",
		})
	}

	// Fetch vehicles (optionally filter by departure and arrival times)
	dep := c.QueryParam("departure")
	arr := c.QueryParam("arrival")
	var allVehicles *sql.Rows
	if dep != "" && arr != "" {
		allVehicles, err = db.GetAvailableVehicles(dep, arr)
	} else {
		allVehicles, err = db.GetAllVehicles()
	}
	if err != nil {
		log.Printf("Error getting vehicles: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve vehicles",
		})
	}
	defer allVehicles.Close()

	// Convert rows to array of maps
	var vehicles []map[string]interface{}
	for allVehicles.Next() {
		var id int64
		var vehicleNumber, vehicleType, status, lastMaintenance, nextMaintenance, createdAt, notes string
		var capacity int
		
		if err := allVehicles.Scan(&id, &vehicleNumber, &vehicleType, &capacity, &status, 
								 &lastMaintenance, &nextMaintenance, &createdAt, &notes); err != nil {
			log.Printf("Error scanning vehicle row: %v", err)
			continue
		}
		
		vehicles = append(vehicles, map[string]interface{}{
			"id":                  id,
			"vehicle_number":      vehicleNumber,
			"type":                vehicleType,
			"capacity":            capacity,
			"status":              status,
			"last_maintenance":    lastMaintenance,
			"next_maintenance":    nextMaintenance,
			"created_at":          createdAt,
			"notes":               notes,
		})
	}

	return c.JSON(http.StatusOK, vehicles)
}

// AdminCreateVehicleHandler - Handler for creating a new vehicle
func AdminCreateVehicleHandler(c echo.Context) error {
	// Make sure user is logged in and has admin role
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Admin create vehicle attempted without valid session")
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "You must be logged in to perform this action",
		})
	}
	
	// Check role: allow Admin and Manager
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Admin" && roleCookie.Value != "Manager") {
		log.Printf("Vehicle creation attempted by unauthorized user: %s", usernameCookie.Value)
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "You do not have permission to perform this action",
		})
	}

	// Parse JSON request
	var req struct {
		VehicleNumber    string `json:"vehicle_number"`
		Type             string `json:"type"`
		Capacity         int    `json:"capacity"`
		Status           string `json:"status"`
		LastMaintenance  string `json:"last_maintenance"`
		NextMaintenance  string `json:"next_maintenance"`
		Notes            string `json:"notes"`
	}
	
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Basic validation
	if req.VehicleNumber == "" || req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Vehicle number and type are required",
		})
	}

	// Set default status if not provided
	if req.Status == "" {
		req.Status = "Active"
	}

	// Add vehicle to database
	vehicleID, err := db.AddVehicle(req.VehicleNumber, req.Type, req.Capacity, req.Status, 
								  req.LastMaintenance, req.NextMaintenance, req.Notes)
	if err != nil {
		log.Printf("Error creating vehicle: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create vehicle: " + err.Error(),
		})
	}
	
	log.Printf("Vehicle created successfully by admin. ID: %d, Vehicle Number: %s", vehicleID, req.VehicleNumber)
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Vehicle created successfully",
		"vehicle_id": vehicleID,
	})
}

// AdminUpdateVehicleHandler - Handler for updating a vehicle
func AdminUpdateVehicleHandler(c echo.Context) error {
	// Ensure request body contains required fields
	var req struct {
		ID               int64  `json:"id"`
		VehicleNumber    string `json:"vehicle_number"`
		Type             string `json:"type"`
		Capacity         int    `json:"capacity"`
		Status           string `json:"status"`
		LastMaintenance  string `json:"last_maintenance"`
		NextMaintenance  string `json:"next_maintenance"`
		Notes            string `json:"notes"`
	}
	
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Validate required fields
	if req.VehicleNumber == "" || req.Type == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Vehicle number and type are required"})
	}

	// Prevent renaming/updating if vehicle has active bookings
	bookingCount, err := db.GetVehicleBookingsCount(req.ID)
	if err != nil {
		log.Printf("Error checking vehicle booking count: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate vehicle bookings"})
	}
	if bookingCount > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot modify vehicle assigned to trips with active bookings"})
	}

	// Update vehicle in database
	err = db.UpdateVehicle(req.ID, req.VehicleNumber, req.Type, req.Capacity, 
						  req.Status, req.LastMaintenance, req.NextMaintenance, req.Notes)
	if err != nil {
		log.Printf("Error updating vehicle %d: %v", req.ID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	log.Printf("Vehicle updated successfully. ID: %d, Vehicle Number: %s", req.ID, req.VehicleNumber)
	return c.JSON(http.StatusOK, map[string]string{"message": "Vehicle updated successfully"})
}

// AdminDeleteVehicleHandler - Handler for deleting a vehicle
func AdminDeleteVehicleHandler(c echo.Context) error {
	idParam := c.Param("id")
	vehicleID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid vehicle ID"})
	}
	
	// Prevent deletion if vehicle has active bookings
	bookingCount, err := db.GetVehicleBookingsCount(vehicleID)
	if err != nil {
		log.Printf("Error checking vehicle booking count before delete: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate vehicle bookings"})
	}
	if bookingCount > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot delete vehicle assigned to trips with active bookings"})
	}
	
	if err := db.DeleteVehicle(vehicleID); err != nil {
		log.Printf("Error deleting vehicle %d: %v", vehicleID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete vehicle"})
	}
	
	log.Printf("Vehicle deleted successfully. ID: %d", vehicleID)
	return c.JSON(http.StatusOK, map[string]string{"message": "Vehicle deleted successfully"})
}

// AdminGetVehicleByIDHandler handles retrieving a specific vehicle by ID
func AdminGetVehicleByIDHandler(c echo.Context) error {
	// Ensure admin or manager role
	username, err := c.Cookie("username")
	if err != nil || username.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	r, err := c.Cookie("user_role")
	if err != nil || (r.Value != "Admin" && r.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}

	// Get the vehicle ID from the URL parameter
	id := c.Param("id")
	vehicleID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid vehicle ID"})
	}

	// Fetch the vehicle
	row, err := db.GetVehicleByID(vehicleID)
	if err != nil {
		log.Printf("Error getting vehicle by ID: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve vehicle",
		})
	}

	// Parse the vehicle data
	var vehicle struct {
		ID                 int64  `json:"id"`
		VehicleNumber      string `json:"vehicle_number"`
		Type               string `json:"type"`
		Capacity           int    `json:"capacity"`
		Status             string `json:"status"`
		LastMaintenanceDate string `json:"last_maintenance_date"`
		NextMaintenanceDate string `json:"next_maintenance_date"`
		CreatedAt          string `json:"created_at"`
		Notes              string `json:"notes"`
	}

	err = row.Scan(
		&vehicle.ID,
		&vehicle.VehicleNumber,
		&vehicle.Type,
		&vehicle.Capacity,
		&vehicle.Status,
		&vehicle.LastMaintenanceDate,
		&vehicle.NextMaintenanceDate,
		&vehicle.CreatedAt,
		&vehicle.Notes,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Vehicle not found"})
		}
		log.Printf("Error scanning vehicle data: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to parse vehicle data",
		})
	}

	return c.JSON(http.StatusOK, vehicle)
}

// --- Trip Management Handlers ---

// AdminTripsHandler - Handler for listing all trips
func AdminTripsHandler(c echo.Context) error {
	// Ensure admin
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Check role: allow Admin and Manager
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Admin" && roleCookie.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}

	rows, err := db.GetAllTrips()
	if err != nil {
		log.Printf("Error retrieving trips: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve trips"})
	}
	defer rows.Close()

	var trips []map[string]interface{}
	for rows.Next() {
		var id, vehicleID int64
		var origin, destination, departure, arrival, createdAt, vehicleNumber string
		if err := rows.Scan(&id, &origin, &destination, &vehicleID, &departure, &arrival, &createdAt, &vehicleNumber); err != nil {
			log.Printf("Error scanning trip: %v", err)
			continue
		}
		trips = append(trips, map[string]interface{}{
			"id": id,
			"origin": origin,
			"destination": destination,
			"vehicle_id": vehicleID,
			"vehicle_number": vehicleNumber,
			"departure_time": departure,
			"arrival_time": arrival,
			"created_at": createdAt,
		})
	}
	return c.JSON(http.StatusOK, trips)
}

// AdminCreateTripHandler - Handler to create a new trip
func AdminCreateTripHandler(c echo.Context) error {
	// Ensure admin
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Check role: allow Admin and Manager
	roleCookie, err := c.Cookie("user_role")
	if err != nil || (roleCookie.Value != "Admin" && roleCookie.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}

	var req struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		VehicleID   int64  `json:"vehicle_id"`
		Departure   string `json:"departure_time"`
		Arrival     string `json:"arrival_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.Origin == "" || req.Destination == "" || req.VehicleID == 0 || req.Departure == "" || req.Arrival == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "All fields are required"})
	}
	
	// Validate origin and destination are not the same
	if req.Origin == req.Destination {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Origin and destination cannot be the same city"})
	}
	
	id, err := db.AddTrip(req.Origin, req.Destination, req.VehicleID, req.Departure, req.Arrival)
	if err != nil {
		log.Printf("Error creating trip: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Trip created", "trip_id": id})
}

// AdminUpdateTripHandler - Handler to update a trip
func AdminUpdateTripHandler(c echo.Context) error {
	var req struct {
		ID          int64  `json:"id"`
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
		VehicleID   int64  `json:"vehicle_id"`
		Departure   string `json:"departure_time"`
		Arrival     string `json:"arrival_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.ID == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Trip ID required"})
	}
	
	// Validate origin and destination are not the same
	if req.Origin == req.Destination {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Origin and destination cannot be the same city"})
	}
	
	// Check if trip has active bookings
	bookingCount, err := db.GetTripBookingsCount(req.ID)
	if err != nil {
		log.Printf("Error checking trip booking count: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate trip bookings"})
	}

	// If trip has bookings, we need to verify origin/destination haven't changed
	if bookingCount > 0 {
		// Get current trip data
		var currentOrigin, currentDestination string
		err := db.DB.QueryRow("SELECT origin, destination FROM trips WHERE id = ?", req.ID).Scan(&currentOrigin, &currentDestination)
		if err != nil {
			log.Printf("Error retrieving current trip data: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve current trip data"})
		}
		
		// Prevent changing origin or destination when bookings exist
		if currentOrigin != req.Origin || currentDestination != req.Destination {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot change origin or destination for trips with active bookings"})
		}
		
		// Allow other changes to proceed (vehicle, schedule times)
		log.Printf("Trip ID %d has %d active bookings. Allowing updates to fields other than origin/destination.", req.ID, bookingCount)
	}
	
	// Prevent conflicts: ensure selected vehicle is available for the new schedule (excluding this trip)
	available, availErr := db.IsVehicleAvailableForTripEdit(req.VehicleID, req.Departure, req.Arrival, req.ID)
	if availErr != nil {
		log.Printf("Error checking vehicle availability: %v", availErr)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error checking vehicle availability"})
	}
	if !available {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Selected vehicle is not available for the new schedule"})
	}
	if err := db.UpdateTrip(req.ID, req.Origin, req.Destination, req.VehicleID, req.Departure, req.Arrival); err != nil {
		log.Printf("Error updating trip: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Trip updated"})
}

// AdminDeleteTripHandler - Handler to delete a trip
func AdminDeleteTripHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid trip ID"})
	}
	
	// Prevent deletion if trip has active bookings
	bookingCount, err := db.GetTripBookingsCount(id)
	if err != nil {
		log.Printf("Error checking trip booking count before delete: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate trip bookings"})
	}
	if bookingCount > 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot delete trip with active bookings"})
	}
	
	if err := db.DeleteTrip(id); err != nil {
		log.Printf("Error deleting trip: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Delete failed"})
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Trip deleted"})
}

// --- Booking Management Handlers ---
// AdminBookingsHandler - Handler for listing all bookings
func AdminBookingsHandler(c echo.Context) error {
	// Ensure admin
	username, err := c.Cookie("username")
	if err != nil || username.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Check role: allow Admin and Manager
	r, err := c.Cookie("user_role")
	if err != nil || (r.Value != "Admin" && r.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}

	// Get filter parameters
	filter := make(map[string]string)
	if p := c.QueryParam("passenger"); p != "" {
		filter["passenger"] = p
	}
	if o := c.QueryParam("origin"); o != "" {
		filter["origin"] = o
	}
	if d := c.QueryParam("destination"); d != "" {
		filter["destination"] = d
	}
	if s := c.QueryParam("status"); s != "" {
		filter["status"] = s
	}
	// Date filters
	if df := c.QueryParam("date_from"); df != "" {
		filter["date_from"] = df
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		filter["date_to"] = dt
	}
	
	// Get sort parameters
	orderBy := c.QueryParam("sort_by")
	orderDir := c.QueryParam("sort_dir")
	// Pagination parameters
	pageParam := c.QueryParam("page")
	pageSizeParam := c.QueryParam("page_size")
	page := 1
	pageSize := 10
	if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 {
		pageSize = ps
	}
	
	// Get bookings
	rows, err := db.GetFilteredBookings(filter, orderBy, orderDir)
	if err != nil {
		log.Printf("Error retrieving bookings: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve bookings"})
	}
	defer rows.Close()

	var bookings []map[string]interface{}
	for rows.Next() {
		var id, tripID int64
		var passenger, socialID, phoneNumber, dateOfBirth, bookingDate, status, origin, destination, departureTime string
		if err := rows.Scan(&id, &tripID, &passenger, &socialID, &phoneNumber, &dateOfBirth, &bookingDate, &status, &origin, &destination, &departureTime); err != nil {
			log.Printf("Error scanning booking: %v", err)
			continue
		}
		bookings = append(bookings, map[string]interface{}{
			"id": id,
			"trip_id": tripID,
			"passenger": passenger,
			"social_id": socialID,
			"phone_number": phoneNumber,
			"date_of_birth": dateOfBirth,
			"booking_date": bookingDate,
			"status": status,
			"origin": origin,
			"destination": destination,
			"departure_time": departureTime,
		})
	}
	// Paginate results
	totalCount := len(bookings)
	start := (page - 1) * pageSize
	if start > totalCount {
		start = totalCount
	}
	end := start + pageSize
	if end > totalCount {
		end = totalCount
	}
	paginated := bookings[start:end]
	return c.JSON(http.StatusOK, map[string]interface{}{ "total": totalCount, "bookings": paginated })
}

// AdminCreateBookingHandler - Handler to create a new booking
func AdminCreateBookingHandler(c echo.Context) error {
	// Ensure admin
	username, err := c.Cookie("username")
	if err != nil || username.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Check role: allow Admin and Manager
	r, err := c.Cookie("user_role")
	if err != nil || (r.Value != "Admin" && r.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}

	var req struct {
		TripID      int64  `json:"trip_id"`
		Passenger   string `json:"passenger"`
		SocialID    string `json:"social_id"`
		PhoneNumber string `json:"phone_number"`
		DateOfBirth string `json:"date_of_birth"`
		Status      string `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.TripID == 0 || req.Passenger == "" || req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Trip, passenger name, and status are required"})
	}
	
	// Check if the trip has available seats
	isAvailable, err := db.CheckTripAvailability(req.TripID)
	if err != nil {
		log.Printf("Error checking trip availability: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to check trip availability: %v", err)})
	}
	
	if !isAvailable {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Trip is fully booked. No seats available."})
	}
	
	id, err := db.AddBooking(req.TripID, req.Passenger, req.SocialID, req.PhoneNumber, req.DateOfBirth, req.Status)
	if err != nil {
		log.Printf("Error creating booking: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{ "message": "Booking created", "booking_id": id })
}

// AdminUpdateBookingStatusHandler - Handler to update booking status
func AdminUpdateBookingStatusHandler(c echo.Context) error {
	var req struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	if req.ID == 0 || req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID and status required"})
	}
	
	// Get current booking status before updating
	var currentStatus string
	var tripID int64
	err := db.DB.QueryRow("SELECT status, trip_id FROM bookings WHERE id = ?", req.ID).Scan(&currentStatus, &tripID)
	if err != nil {
		log.Printf("Error retrieving current booking status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve booking information"})
	}
	
	// Update the booking status
	if err := db.UpdateBookingStatus(req.ID, req.Status); err != nil {
		log.Printf("Error updating booking status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	
	// Special processing for status changes related to capacity
	// If booking was not cancelled before but is now, or vice versa
	if (currentStatus != "Cancelled" && req.Status == "Cancelled") || 
	   (currentStatus == "Cancelled" && req.Status != "Cancelled") {
		// Log the capacity change
		capacity, err := db.GetTripVehicleCapacity(tripID)
		if err != nil {
			log.Printf("Warning: Could not get trip capacity after status change: %v", err)
		}
		
		count, err := db.GetTripBookingsCount(tripID)
		if err != nil {
			log.Printf("Warning: Could not get booking count after status change: %v", err)
		}
		
		log.Printf("Trip ID %d: Status changed from %s to %s. Capacity: %d, Active bookings: %d", 
			tripID, currentStatus, req.Status, capacity, count)
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Booking status updated"})
}

// AdminDeleteBookingHandler - Handler to delete a booking
func AdminDeleteBookingHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid booking ID"})
	}
	
	// Get booking information before deleting
	var status string
	var tripID int64
	err = db.DB.QueryRow("SELECT status, trip_id FROM bookings WHERE id = ?", id).Scan(&status, &tripID)
	if err != nil {
		log.Printf("Error retrieving booking information: %v", err)
		// Continue with deletion attempt even if this fails
	}
	
	if err := db.DeleteBooking(id); err != nil {
		log.Printf("Error deleting booking: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Delete failed"})
	}
	
	// Log capacity change if booking was not cancelled (as cancelled bookings don't affect capacity)
	if err == nil && status != "Cancelled" && tripID > 0 {
		capacity, err := db.GetTripVehicleCapacity(tripID)
		if err != nil {
			log.Printf("Warning: Could not get trip capacity after booking deletion: %v", err)
		}
		
		count, err := db.GetTripBookingsCount(tripID)
		if err != nil {
			log.Printf("Warning: Could not get booking count after booking deletion: %v", err)
		}
		
		log.Printf("Trip ID %d: Booking deleted. Current capacity: %d, Active bookings: %d", 
			tripID, capacity, count)
	}
	
	return c.JSON(http.StatusOK, map[string]string{"message": "Booking deleted"})
}

// AdminTripCapacityHandler - Handler to get trip capacity information
func AdminTripCapacityHandler(c echo.Context) error {
	// Parse trip ID from path parameter
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid trip ID"})
	}
	
	// Get vehicle capacity for the trip
	capacity, err := db.GetTripVehicleCapacity(id)
	if err != nil {
		log.Printf("Error getting trip vehicle capacity: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get trip capacity"})
	}
	
	// Get current booking count
	bookingsCount, err := db.GetTripBookingsCount(id)
	if err != nil {
		log.Printf("Error getting trip bookings count: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get bookings count"})
	}
	
	// Calculate available seats
	available := capacity - bookingsCount
	if available < 0 {
		available = 0 // Safeguard against negative values
	}
	
	// Return capacity information
	return c.JSON(http.StatusOK, map[string]interface{}{
		"capacity": capacity,
		"booked": bookingsCount,
		"available": available,
		"is_available": available > 0,
	})
}

// AdminReportsDataHandler - Handler for getting reports data
func AdminReportsDataHandler(c echo.Context) error {
	// Ensure user is logged in; group middleware enforces role
	if cookie, err := c.Cookie("username"); err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Parse query params
	reportType := c.QueryParam("report")
	from := c.QueryParam("from")
	to := c.QueryParam("to")
	var columns []string
	var rowsData []map[string]interface{}
	var err error
	switch reportType {
	case "booking_summary":
		columns = []string{"date", "bookings"}
		rowsData, err = db.GetBookingSummary(from, to)
	case "route_performance":
		columns = []string{"origin", "destination", "bookings"}
		rowsData, err = db.GetRoutePerformance(from, to)
	case "cancellation_summary":
		columns = []string{"date", "bookings", "cancellations", "cancellation_rate"}
		rowsData, err = db.GetCancellationSummary(from, to)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unknown report type"})
	}
	if err != nil {
		log.Printf("Error generating report data: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate report data"})
	}
	// Return JSON
	return c.JSON(http.StatusOK, map[string]interface{}{"columns": columns, "rows": rowsData})
}

// AdminReportsExportHandler - Handler for exporting reports to XLSX
func AdminReportsExportHandler(c echo.Context) error {
	// Ensure user is logged in; group middleware enforces role
	if cookie, err := c.Cookie("username"); err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Parse params
	reportType := c.QueryParam("report")
	from := c.QueryParam("from")
	to := c.QueryParam("to")
	var columns []string
	var rowsData []map[string]interface{}
	var err error
	switch reportType {
	case "booking_summary":
		columns = []string{"date", "bookings"}
		rowsData, err = db.GetBookingSummary(from, to)
	case "route_performance":
		columns = []string{"origin", "destination", "bookings"}
		rowsData, err = db.GetRoutePerformance(from, to)
	case "cancellation_summary":
		columns = []string{"date", "bookings", "cancellations", "cancellation_rate"}
		rowsData, err = db.GetCancellationSummary(from, to)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unknown report type"})
	}
	if err != nil {
		log.Printf("Error generating report for export: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate report for export"})
	}
	// Create Excel file
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	// Write header
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}
	// Write data rows
	for r, row := range rowsData {
		for cidx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(cidx+1, r+2)
			f.SetCellValue(sheet, cell, row[col])
		}
	}
	// Set headers for download
	filename := fmt.Sprintf("%s_%s_to_%s.xlsx", reportType, from, to)
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	// Stream file
	err = f.Write(c.Response().Writer)
	if err != nil {
		log.Printf("Error writing Excel file: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to write Excel file"})
	}
	return nil
}

// AdminBackupHandler - Handler to create a database backup
func AdminBackupHandler(c echo.Context) error {
	// Ensure admin
	if cookie, err := c.Cookie("username"); err != nil || cookie.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	// Check role: allow Admin and Manager
	if role, err := c.Cookie("user_role"); err != nil || (role.Value != "Admin" && role.Value != "Manager") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Permission denied"})
	}
	// Perform backup
	dbPath := os.Getenv("SQLITE_DB_PATH")
	backupPath, err := utils.BackupDatabase(dbPath)
	if err != nil {
		log.Printf("Error creating backup: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Backup created", "path": backupPath})
}

// AdminBackupDownloadHandler - Handler to download the latest backup file
func AdminBackupDownloadHandler(c echo.Context) error {
	dbPath := os.Getenv("SQLITE_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "securesignin.db")
	}
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	backupFile := filepath.Join(backupDir, "securesignin.db.bak")
	return c.File(backupFile)
} 