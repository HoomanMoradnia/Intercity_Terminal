package dashboard

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"SecureSignIn/db"
	"SecureSignIn/handlers"
	"SecureSignIn/handlers/templates"
	"SecureSignIn/models"
	"database/sql"
	"strconv"
)

// Handler - For logged in users
func DashboardHandler(c echo.Context) error {
	// Make sure user is logged in first by checking for username cookie
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("Dashboard access attempted without valid session")
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in to access this page")
	}

	username := usernameCookie.Value
	log.Printf("Dashboard accessed by user: %s", username)

	// Get the user role
	roleCookie, err := c.Cookie("user_role")
	var userRole string
	if err != nil || roleCookie.Value == "" {
		// If role cookie not found, default to Operator
		userRole = "Operator"
		log.Printf("Warning: No role cookie found for user %s, defaulting to Operator", username)
	} else {
		userRole = roleCookie.Value
	}

	// Fetch all users
	allUsers, err := db.GetAllUsers()
	if err != nil {
		log.Printf("Error getting all users: %v", err)
		return templates.RenderTemplate(c, "dashboard.html", models.PageData{
			Title:      "Dashboard",
			Error:      "Error retrieving user data",
			ActivePage: "dashboard",
			IsLoggedIn: true,
			Username:   username,
			UserRole:   userRole,
		})
	}
	userMaps, err := handlers.RowsToMap(allUsers)
	if err != nil {
		log.Printf("Error converting user rows to map: %v", err)
	}

	// Fetch login history
	loginHistory, err := db.GetLoginHistory()
	if err != nil {
		log.Printf("Error getting login history: %v", err)
		return templates.RenderTemplate(c, "dashboard.html", models.PageData{
			Title:      "Dashboard",
			Error:      "Error retrieving login history",
			ActivePage: "dashboard",
			IsLoggedIn: true,
			Username:   username,
			UserRole:   userRole,
		})
	}
	historyMaps, err := handlers.RowsToMap(loginHistory)
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

	// Prepare common data
	data := models.PageData{
		Title:        "Dashboard",
		ActivePage:   "dashboard",
		Users:        userMaps,
		LoginLogs:    historyMaps,
		IsLoggedIn:   true,
		Username:     username,
		UserRole:     userRole,
		HasSecurityQ: hasSecurityQ,
		Success:      c.QueryParam("success"),
		Error:        c.QueryParam("error"),
	}

	// Render appropriate dashboard based on role
	switch userRole {
	case "Operator":
		return templates.RenderTemplate(c, "operator_dashboard.html", data)
	case "Manager":
		return templates.RenderTemplate(c, "manager_dashboard.html", data)
	case "Accountant":
		return templates.RenderTemplate(c, "accountant_dashboard.html", data)
	case "Admin":
		return templates.RenderTemplate(c, "admin_dashboard.html", data)
	default:
		// Default to operator dashboard if role is unknown
		log.Printf("Warning: Unknown role '%s' for user %s, defaulting to Operator dashboard", userRole, username)
		return templates.RenderTemplate(c, "operator_dashboard.html", data)
	}
}

// IndexHandler - Redirects appropriately
func IndexHandler(c echo.Context) error {
	data := models.PageData{
		Title:      "Home",
		ActivePage: "home",
	}
	return templates.RenderTemplate(c, "index.html", data)
}

// HealthCheckHandler - Health check endpoint
func HealthCheckHandler(c echo.Context) error {
	if err := db.DB.Ping(); err != nil {
		log.Printf("Health check failed: DB ping error: %v", err)
		return c.String(http.StatusServiceUnavailable, "Database connection failed")
	}
	return c.String(http.StatusOK, "OK")
}

// TripPlanHandler - Handler to show upcoming trips for the next week
func TripPlanHandler(c echo.Context) error {
	// Ensure user is logged in
	usernameCookie, err := c.Cookie("username")
	if err != nil || usernameCookie.Value == "" {
		log.Printf("TripPlan access attempted without valid session")
		return c.Redirect(http.StatusSeeOther, "/login?error=You must be logged in")
	}
	username := usernameCookie.Value
	// Get user role
	roleCookie, err := c.Cookie("user_role")
	var userRole string
	if err != nil || roleCookie.Value == "" {
		userRole = "Operator"
	} else {
		userRole = roleCookie.Value
	}
	// Query upcoming trips for next 7 days
	rows, err := db.DB.Query(`
		SELECT t.id, t.origin, t.destination, t.departure_time, t.arrival_time, v.vehicle_number
		FROM trips t
		JOIN vehicles v ON t.vehicle_id = v.id
		WHERE DATE(t.departure_time) >= DATE('now')
		  AND DATE(t.departure_time) <= DATE('now', '+7 days')
		ORDER BY t.departure_time
	`)
	if err != nil {
		log.Printf("Error querying trips for TripPlan: %v", err)
		return templates.RenderTemplate(c, "trip_plan.html", models.PageData{
			Title:      "Trip Plan",
			ActivePage: "trip-plan",
			Error:      "Failed to load trips",
			IsLoggedIn: true,
			Username:   username,
			UserRole:   userRole,
		})
	}
	defer rows.Close()
	tripMaps, err := handlers.RowsToMap(rows)
	if err != nil {
		log.Printf("Error converting trip rows: %v", err)
	}
	data := models.PageData{
		Title:      "Trip Plan",
		ActivePage: "trip-plan",
		Trips:      tripMaps,
		IsLoggedIn: true,
		Username:   username,
		UserRole:   userRole,
		Success:    c.QueryParam("success"),
		Error:      c.QueryParam("error"),
	}
	return templates.RenderTemplate(c, "trip_plan.html", data)
}

// OperatorCreateBookingHandler handles booking creation by operators
func OperatorCreateBookingHandler(c echo.Context) error {
	// Check login and role
	username, err := c.Cookie("username")
	if err != nil || username.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}

	var req models.Booking
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Validate
	if req.TripID == 0 || req.Passenger == "" || req.Status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Trip, passenger name, and status are required"})
	}

	// Add booking
	id, err := db.AddBooking(req.TripID, req.Passenger, req.SocialID, req.PhoneNumber, req.DateOfBirth, req.Status)
	if err != nil {
		log.Printf("Error creating booking: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Booking created successfully",
		"booking_id": id,
	})
}

// OperatorGetTripByRouteHandler finds a trip by route
func OperatorGetTripByRouteHandler(c echo.Context) error {
	// Check login
	username, err := c.Cookie("username")
	if err != nil || username.Value == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}
	
	// Get query parameters
	origin := c.QueryParam("origin")
	destination := c.QueryParam("destination")
	
	if origin == "" || destination == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Origin and destination are required"})
	}
	
	// Find trip
	tripID, err := db.FindTripByRoute(origin, destination)
	if err != nil {
		log.Printf("Error finding trip: %v", err)
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"trip_id": tripID,
	})
}

// OperatorGetBookingsHandler returns bookings with optional filtering and pagination
func OperatorGetBookingsHandler(c echo.Context) error {
	// Check login
	_, err := c.Cookie("username")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
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
	pageSize := 10 // Default page size
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

	// Process results
	var bookings []map[string]interface{}
	for rows.Next() {
		var id, tripID int64
		var passenger, socialID, phoneNumber, dateOfBirth, bookingDate, status, origin, destination, departureTime string
		if err := rows.Scan(&id, &tripID, &passenger, &socialID, &phoneNumber, &dateOfBirth, &bookingDate, &status, &origin, &destination, &departureTime); err != nil {
			log.Printf("Error scanning booking: %v", err)
			continue
		}
		bookings = append(bookings, map[string]interface{}{
			"id": id, "trip_id": tripID, "passenger": passenger, "social_id": socialID, "phone_number": phoneNumber,
			"date_of_birth": dateOfBirth, "booking_date": bookingDate, "status": status, "origin": origin,
			"destination": destination, "departure_time": departureTime,
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

	return c.JSON(http.StatusOK, map[string]interface{}{"total": totalCount, "bookings": paginated})
}

// OperatorTripsHandler returns all trips
func OperatorTripsHandler(c echo.Context) error {
	// Check login
	_, err := c.Cookie("username")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "You must be logged in"})
	}

	// Fetch all trips
	allTrips, err := db.GetAllTrips()
	if err != nil {
		log.Printf("Error getting all trips: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve trips"})
	}
	defer allTrips.Close()

	var trips []map[string]interface{}
	for allTrips.Next() {
		var id, vehicleID int64
		var origin, destination, departureTime, arrivalTime, createdAt string
		var vehicleNumber sql.NullString // Use sql.NullString for vehicleNumber
		if err := allTrips.Scan(&id, &origin, &destination, &vehicleID, &departureTime, &arrivalTime, &createdAt, &vehicleNumber); err != nil {
			log.Printf("Error scanning trip row: %v", err)
			continue
		}

		trip := map[string]interface{}{
			"id":             id,
			"origin":         origin,
			"destination":    destination,
			"vehicle_id":     vehicleID,
			"departure_time": departureTime,
			"arrival_time":   arrivalTime,
			"created_at":     createdAt,
		}
		if vehicleNumber.Valid {
			trip["vehicle_number"] = vehicleNumber.String
		} else {
			trip["vehicle_number"] = "N/A"
		}
		trips = append(trips, trip)
	}

	return c.JSON(http.StatusOK, trips)
}

// OperatorUpdateTripHandler handles trip updates
func OperatorUpdateTripHandler(c echo.Context) error {
	var req struct {
		ID            int64  `json:"id"`
		Origin        string `json:"origin"`
		Destination   string `json:"destination"`
		VehicleID     int64  `json:"vehicle_id"`
		DepartureTime string `json:"departure_time"`
		ArrivalTime   string `json:"arrival_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	available, err := db.IsVehicleAvailableForTripEdit(req.VehicleID, req.DepartureTime, req.ArrivalTime, req.ID)
	if err != nil {
		log.Printf("Error checking vehicle availability for trip update: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to validate vehicle availability"})
	}
	if !available {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Vehicle is not available for the selected time range"})
	}

	if err := db.UpdateTrip(req.ID, req.Origin, req.Destination, req.VehicleID, req.DepartureTime, req.ArrivalTime); err != nil {
		log.Printf("Error updating trip: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Trip updated"})
}

// OperatorDeleteTripHandler handles trip deletion
func OperatorDeleteTripHandler(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid trip ID"})
	}

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

// OperatorTripCapacityHandler returns trip capacity details
func OperatorTripCapacityHandler(c echo.Context) error {
	idParam := c.Param("id")
	tripID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid trip ID"})
	}

	capacity, err := db.GetTripVehicleCapacity(tripID)
	if err != nil {
		log.Printf("Error getting trip capacity for trip ID %d: %v", tripID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	booked, err := db.GetTripBookingsCount(tripID)
	if err != nil {
		log.Printf("Error getting trip booking count for trip ID %d: %v", tripID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	available := capacity - booked
	isAvailable := available > 0

	return c.JSON(http.StatusOK, map[string]interface{}{
		"capacity":     capacity,
		"booked":       booked,
		"available":    available,
		"is_available": isAvailable,
	})
}

// OperatorDeleteBookingHandler handles booking deletion
func OperatorDeleteBookingHandler(c echo.Context) error {
	bookingID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid booking ID"})
	}

	if err := db.DeleteBooking(bookingID); err != nil {
		log.Printf("Error deleting booking: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Delete failed"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Booking deleted"})
}

// OperatorCreateTripHandler handles trip creation
func OperatorCreateTripHandler(c echo.Context) error {
	var req struct {
		Origin        string `json:"origin"`
		Destination   string `json:"destination"`
		VehicleID     int64  `json:"vehicle_id"`
		DepartureTime string `json:"departure_time"`
		ArrivalTime   string `json:"arrival_time"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if req.Origin == "" || req.Destination == "" || req.VehicleID == 0 || req.DepartureTime == "" || req.ArrivalTime == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "All fields are required"})
	}

	id, err := db.AddTrip(req.Origin, req.Destination, req.VehicleID, req.DepartureTime, req.ArrivalTime)
	if err != nil {
		log.Printf("Error creating trip: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "Trip created", "trip_id": id})
} 