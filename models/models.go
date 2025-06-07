package models

// PageData struct for template rendering
type PageData struct {
	Title            string
	Error            string
	Success          string
	ActivePage       string
	Users            []map[string]interface{}
	LoginLogs        []map[string]interface{}
	Trips            []map[string]interface{}
	IsLoggedIn       bool
	Username         string
	UserRole         string
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

// Booking struct
type Booking struct {
	ID          int64  `json:"id"`
	TripID      int64  `json:"trip_id"`
	Passenger   string `json:"passenger"`
	SocialID    string `json:"social_id"`
	PhoneNumber string `json:"phone_number"`
	DateOfBirth string `json:"date_of_birth"`
	BookingDate string `json:"booking_date"`
	Status      string `json:"status"`
} 