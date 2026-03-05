package models

import "time"

// User represents a Greenhouse user from the Harvest API v3.
// v3 field changes from v1: primary_email_address → primary_email, disabled → deactivated,
// offices/departments replaced by office_ids/department_ids.
type User struct {
	ID             int       `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	PrimaryEmail   string    `json:"primary_email"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	Deactivated    bool      `json:"deactivated"`
	SiteAdmin      bool      `json:"site_admin,omitempty"`
	Emails         []string  `json:"emails,omitempty"`
	EmployeeID     string    `json:"employee_id"`
	OfficeIDs      []int     `json:"office_ids,omitempty"`
	DepartmentIDs  []int     `json:"department_ids,omitempty"`
	JobTitle       string    `json:"job_title,omitempty"`
}

// UserRole represents a role from the Harvest API v3.
// v3 field changes from v1: type → role_type.
type UserRole struct {
	ID        int       `json:"id,omitempty"`
	RoleType  string    `json:"role_type,omitempty"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// JobPermission represents a user job permission from the Harvest API v3.
// In v3 this is a top-level resource at /v3/user_job_permissions with user_id and role_id fields.
type JobPermission struct {
	ID        int  `json:"id,omitempty"`
	JobID     int  `json:"job_id,omitempty"`
	UserID    int  `json:"user_id,omitempty"`
	RoleID    int  `json:"role_id,omitempty"`
	Automated *bool `json:"automated,omitempty"`
}

// FutureJobPermission represents a future job permission from the Harvest API v3.
// In v3 this is a top-level resource at /v3/future_job_permissions.
type FutureJobPermission struct {
	ID           int    `json:"id,omitempty"`
	OfficeID     *int   `json:"office_id,omitempty"`
	DepartmentID *int   `json:"department_id,omitempty"`
	UserID       int    `json:"user_id,omitempty"`
	RoleID       int    `json:"role_id,omitempty"`
}

// TokenResponse represents the response from the Greenhouse auth token endpoint.
type TokenResponse struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"` //nolint:gosec // struct field for API response deserialization
	ExpiresAt   string `json:"expires_at"`
}
