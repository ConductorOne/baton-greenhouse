package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Data Structures (v3 API format) ---

// APIError defines the structure for API error responses.
type APIError struct {
	APIMessage string `json:"message"`
	Errors     []struct {
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"errors"`
}

type Candidate struct {
	ID           int64     `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email_addresses"`
	Company      string    `json:"company"`
	Title        string    `json:"title"`
	LastActivity time.Time `json:"last_activity"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Job struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	OpenedAt   time.Time  `json:"opened_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Department *struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"department,omitempty"`
	Offices []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"offices,omitempty"`
}

type Application struct {
	ID             int64     `json:"id"`
	CandidateID    int64     `json:"candidate_id"`
	JobID          int64     `json:"job_id"`
	Status         string    `json:"status"`
	AppliedAt      time.Time `json:"applied_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	Source         *struct {
		ID   int64  `json:"id"`
		Name string `json:"public_name"`
	} `json:"source,omitempty"`
	CreditedTo *struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"credited_to,omitempty"`
}

// User uses v3 field names: primary_email (not primary_email_address), deactivated (not disabled).
type User struct {
	ID           int64     `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PrimaryEmail string    `json:"primary_email"`
	Deactivated  bool      `json:"deactivated"`
	SiteAdmin    bool      `json:"site_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	EmployeeID   *string   `json:"employee_id,omitempty"`
	Emails       []string  `json:"emails,omitempty"`
	OfficeIDs    []int     `json:"office_ids,omitempty"`
	DepartmentIDs []int    `json:"department_ids,omitempty"`
}

// UserRole uses v3 field name: role_type (not type).
type UserRole struct {
	ID       int64     `json:"id"`
	RoleType string    `json:"role_type"`
	Name     string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobPermission uses v3 field names: user_id and role_id (not user_role_id).
type JobPermission struct {
	ID     int64 `json:"id"`
	JobID  int64 `json:"job_id"`
	UserID int64 `json:"user_id"`
	RoleID int64 `json:"role_id"`
}

// FutureJobPermission uses v3 field names: user_id and role_id.
type FutureJobPermission struct {
	ID           int64  `json:"id"`
	OfficeID     *int64 `json:"office_id,omitempty"`
	DepartmentID *int64 `json:"department_id,omitempty"`
	UserID       int64  `json:"user_id"`
	RoleID       int64  `json:"role_id"`
}

// TokenResponse mimics the v3 auth token response.
type TokenResponse struct {
	TokenType   string `json:"token_type"`
	AccessToken string `json:"access_token"`
	ExpiresAt   string `json:"expires_at"`
}

// --- Mock Data ---

var (
	mockCandidates   []Candidate
	mockJobs         []Job
	mockApplications []Application
	mockUsers        []User
	mockUserRoles    []UserRole

	mockUserJobPermissionsMap       map[int64][]JobPermission
	mockUserFutureJobPermissionsMap map[int64][]FutureJobPermission
	nextPermissionID                int64 = 1

	rateLimitLimit       = 40
	rateLimitRemaining   = rateLimitLimit
	rateLimitWindowStart time.Time
	rateLimitResetDelta  = 60 * time.Second
	rateLimitMutex       sync.Mutex
	rng                  *rand.Rand
)

func initMockData() {
	now := time.Now()
	rateLimitWindowStart = now
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source) //nolint:gosec // test server using non-cryptographic random data

	// Candidates
	for i := 1; i <= 150; i++ {
		mockCandidates = append(mockCandidates, Candidate{
			ID: int64(i), FirstName: fmt.Sprintf("CandidateFirstName%d", i), LastName: fmt.Sprintf("LastName%d", i),
			Email: fmt.Sprintf("candidate%d@example.com", i), Company: fmt.Sprintf("Company %d", (i%10)+1),
			Title: fmt.Sprintf("Developer %d", (i%5)+1), LastActivity: now.Add(-time.Duration(i) * time.Hour),
			CreatedAt: now.Add(-time.Duration(i*2) * time.Hour), UpdatedAt: now.Add(-time.Duration(i) * time.Hour / 2),
		})
	}

	// Jobs
	for i := 1; i <= 75; i++ {
		var closedAt *time.Time
		status := "open"
		if i%5 == 0 {
			ct := now.Add(-time.Duration(i) * 24 * time.Hour)
			closedAt = &ct
			status = "closed"
		}
		job := Job{
			ID: int64(i), Name: fmt.Sprintf("Software Engineer Level %d", i), Status: status,
			OpenedAt: now.Add(-time.Duration(i*2) * 24 * time.Hour), ClosedAt: closedAt,
			CreatedAt: now.Add(-time.Duration(i*3) * 24 * time.Hour), UpdatedAt: now.Add(-time.Duration(i) * 24 * time.Hour),
		}
		mockJobs = append(mockJobs, job)
	}

	// Applications
	for i := 1; i <= 200; i++ {
		mockApplications = append(mockApplications, Application{
			ID: int64(i), CandidateID: int64((i % 150) + 1), JobID: int64((i % 75) + 1),
			Status: []string{"active", "rejected", "hired", "new"}[i%4], AppliedAt: now.Add(-time.Duration(i*5) * time.Hour),
			LastActivityAt: now.Add(-time.Duration(i*2) * time.Hour),
			Source:         nil,
		})
	}

	// Users (v3 format)
	for i := 1; i <= 50; i++ {
		empID := fmt.Sprintf("EMP%04d", i)
		mockUsers = append(mockUsers, User{
			ID: int64(i), FirstName: fmt.Sprintf("UserFirst%d", i), LastName: fmt.Sprintf("UserLast%d", i),
			PrimaryEmail: fmt.Sprintf("user%d@example-corp.com", i), Deactivated: i%10 == 0, SiteAdmin: i%15 == 0,
			CreatedAt: now.Add(-time.Duration(i*7*24) * time.Hour), UpdatedAt: now.Add(-time.Duration(i*24) * time.Hour),
			EmployeeID: &empID,
			Emails: []string{fmt.Sprintf("user%d@example-corp.com", i)},
		})
	}

	// User Roles (v3 format with role_type)
	mockUserRoles = []UserRole{
		{ID: 1, RoleType: "job_admin", Name: "Job Admin - Marketing Department", CreatedAt: now, UpdatedAt: now},
		{ID: 2, RoleType: "deprecated_interviewer", Name: "Technical Interviewer - Backend", CreatedAt: now, UpdatedAt: now},
		{ID: 3, RoleType: "deprecated_interviewer", Name: "Cultural Fit Interviewer", CreatedAt: now, UpdatedAt: now},
		{ID: 4, RoleType: "job_admin", Name: "Job Admin - Engineering Department", CreatedAt: now, UpdatedAt: now},
		{ID: 5, RoleType: "job_admin", Name: "Job Admin - All Jobs", CreatedAt: now, UpdatedAt: now},
		{ID: 6, RoleType: "job_admin", Name: "Hiring Manager", CreatedAt: now, UpdatedAt: now},
		{ID: 7, RoleType: "deprecated_interviewer", Name: "Accounting Department", CreatedAt: now, UpdatedAt: now},
		{ID: 8, RoleType: "deprecated_interviewer", Name: "Technical Interviewer - Frontend", CreatedAt: now, UpdatedAt: now},
		{ID: 9, RoleType: "deprecated_interviewer", Name: "Executive Interviewer", CreatedAt: now, UpdatedAt: now},
	}

	// User Job Permissions & Future Job Permissions (v3 format with user_id and role_id)
	mockUserJobPermissionsMap = make(map[int64][]JobPermission)
	mockUserFutureJobPermissionsMap = make(map[int64][]FutureJobPermission)

	for _, user := range mockUsers {
		var jobPerms []JobPermission
		var futureJobPerms []FutureJobPermission
		numJobPerms := rng.Intn(len(mockJobs)/3+1) + 3
		jobIndices := rng.Perm(len(mockJobs))

		for i := 0; i < numJobPerms && i < len(mockJobs); i++ {
			job := mockJobs[jobIndices[i]]
			var userRole UserRole
			if len(mockUserRoles) > 0 {
				relevantRoles := []UserRole{}
				for _, r := range mockUserRoles {
					if r.RoleType == "job_admin" || r.RoleType == "deprecated_interviewer" {
						relevantRoles = append(relevantRoles, r)
					}
				}
				if len(relevantRoles) > 0 {
					userRole = relevantRoles[rng.Intn(len(relevantRoles))]
				} else {
					userRole = mockUserRoles[rng.Intn(len(mockUserRoles))]
				}
			} else {
				userRole = UserRole{ID: 999, RoleType: "fallback", Name: "Fallback Role"} //nolint:govet
			}
			jobPerms = append(jobPerms, JobPermission{
				ID: nextPermissionID, JobID: job.ID, UserID: user.ID, RoleID: userRole.ID,
			})
			nextPermissionID++
		}
		mockUserJobPermissionsMap[user.ID] = jobPerms

		numFutureJobPerms := rng.Intn(8) + 1
		for i := 0; i < numFutureJobPerms; i++ {
			var userRole UserRole
			if len(mockUserRoles) > 0 {
				relevantRoles := []UserRole{}
				for _, r := range mockUserRoles {
					if r.RoleType == "job_admin" {
						relevantRoles = append(relevantRoles, r)
					}
				}
				if len(relevantRoles) == 0 {
					for _, r := range mockUserRoles {
						if r.RoleType == "deprecated_interviewer" {
							relevantRoles = append(relevantRoles, r)
						}
					}
				}
				if len(relevantRoles) > 0 {
					userRole = relevantRoles[rng.Intn(len(relevantRoles))]
				} else {
					userRole = mockUserRoles[rng.Intn(len(mockUserRoles))]
				}
			} else {
				userRole = UserRole{ID: 998, RoleType: "fallback_future", Name: "Fallback Future Role"} //nolint:govet
			}
			var officeID *int64
			var deptID *int64
			if len(mockJobs) > 0 && rng.Float32() < 0.7 {
				jobWithMeta := mockJobs[rng.Intn(len(mockJobs))]
				if len(jobWithMeta.Offices) > 0 && rng.Float32() < 0.8 {
					office := jobWithMeta.Offices[rng.Intn(len(jobWithMeta.Offices))]
					officeID = &office.ID
				}
				if jobWithMeta.Department != nil && rng.Float32() < 0.8 {
					deptID = &jobWithMeta.Department.ID
				}
			}
			futureJobPerms = append(futureJobPerms, FutureJobPermission{
				ID: nextPermissionID, OfficeID: officeID,
				DepartmentID: deptID, UserID: user.ID, RoleID: userRole.ID,
			})
			nextPermissionID++
		}
		mockUserFutureJobPermissionsMap[user.ID] = futureJobPerms
	}
}

// --- Utility Functions ---

func handleRateLimiting(w http.ResponseWriter) bool {
	rateLimitMutex.Lock()
	defer rateLimitMutex.Unlock()
	if time.Since(rateLimitWindowStart) >= rateLimitResetDelta {
		rateLimitRemaining = rateLimitLimit
		rateLimitWindowStart = time.Now()
		log.Println("Rate limit window reset.")
	}
	resetTime := rateLimitWindowStart.Add(rateLimitResetDelta).Unix()
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rateLimitLimit))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))
	if rateLimitRemaining <= 0 {
		w.Header().Set("X-RateLimit-Remaining", "0")
		respondWithError(w,
			http.StatusTooManyRequests,
			"Rate limit exceeded.")
		log.Printf("Rate limit exceeded. Remaining: 0. Resets at: %s", time.Unix(resetTime, 0).Format(time.RFC1123))
		return true
	}
	rateLimitRemaining--
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rateLimitRemaining))
	log.Printf("Request processed. Rate limit remaining: %d", rateLimitRemaining)
	return false
}

func parsePagination(r *http.Request) (int, int, error) {
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	page := 1
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			return 0, 0, fmt.Errorf("invalid 'page' parameter: must be a positive integer")
		}
	}
	perPage := 50
	if perPageStr != "" {
		var err error
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil || perPage <= 0 {
			return 0, 0, fmt.Errorf("invalid 'per_page' parameter: must be a positive integer")
		}
		if perPage > 500 {
			perPage = 500
		}
	}
	return page, perPage, nil
}

func generateLinkHeader(w http.ResponseWriter, r *http.Request, currentPage, perPage, totalItems int) {
	if totalItems == 0 || perPage == 0 {
		return
	}
	baseURLPath := r.URL.Path
	buildLinkURL := func(p int) string {
		q := url.Values{}
		q.Set("page", strconv.Itoa(p))
		q.Set("per_page", strconv.Itoa(perPage))
		linkURL := url.URL{Path: baseURLPath, RawQuery: q.Encode()}
		return linkURL.String()
	}
	totalPages := int(math.Ceil(float64(totalItems) / float64(perPage)))
	var links []string
	if currentPage < totalPages {
		links = append(links, fmt.Sprintf("<%s>; rel=\"next\"", buildLinkURL(currentPage+1)))
	}
	if totalPages > 0 && currentPage != totalPages {
		links = append(links, fmt.Sprintf("<%s>; rel=\"last\"", buildLinkURL(totalPages)))
	}
	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		if w.Header().Get("Content-Type") == "" {
			http.Error(w, `{"message": "Internal Server Error while marshalling JSON", "errors": []}`, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, err = w.Write(response)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	apiErr := APIError{
		APIMessage: message,
		Errors: make([]struct {
			Message string `json:"message"`
			Field   string `json:"field"`
		}, 0),
	}

	if statusCode == http.StatusBadRequest {
		switch {
		case strings.Contains(message, "'page' parameter"):
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: "Invalid value provided for 'page' parameter.",
				Field:   "page",
			})
		case strings.Contains(message, "'per_page' parameter"):
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: "Invalid value provided for 'per_page' parameter.",
				Field:   "per_page",
			})
		case strings.Contains(message, "Invalid User ID format"):
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: message,
				Field:   "user_id",
			})
		}
	}

	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		log.Printf("Error encoding APIError JSON response: %v", err)
	}
}

// --- Handlers ---

// tokenHandler mimics the Greenhouse auth token endpoint.
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check for Basic auth header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}

	tokenResp := TokenResponse{
		TokenType:   "Bearer",
		AccessToken: "mock-jwt-token-for-testing",
		ExpiresAt:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}

	respondWithJSON(w, http.StatusOK, tokenResp)
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPage, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	totalItems := len(mockUsers)
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	var paginatedData []User
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedData = mockUsers[startIndex:endIndex]
	} else {
		paginatedData = []User{}
	}
	generateLinkHeader(w, r, page, perPage, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedData)
}

func getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	userID, err := getUserIDFromPath(r.URL.Path, "/v3/users/")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	for _, user := range mockUsers {
		if user.ID == userID {
			respondWithJSON(w, http.StatusOK, user)
			return
		}
	}
	respondWithError(w, http.StatusNotFound, fmt.Sprintf("User with ID %d not found", userID))
}

func listUserRolesHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPage, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	totalItems := len(mockUserRoles)
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	var paginatedData []UserRole
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedData = mockUserRoles[startIndex:endIndex]
	} else {
		paginatedData = []UserRole{}
	}
	generateLinkHeader(w, r, page, perPage, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedData)
}

func getUserIDFromPath(path string, prefix string) (int64, error) {
	trimmedPath := strings.TrimPrefix(path, prefix)
	trimmedPath = strings.TrimSuffix(trimmedPath, "/")
	parts := strings.Split(trimmedPath, "/")
	if len(parts) < 1 || parts[0] == "" {
		return 0, fmt.Errorf("user ID not found in path")
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid User ID format in URL path: '%s' is not a number", parts[0])
	}
	return userID, nil
}

// listUserJobPermissionsHandler serves /v3/user_job_permissions with user_id query param filter.
func listUserJobPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPageVal, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid user_id '%s'.", userIDStr))
		return
	}

	allPermissions, ok := mockUserJobPermissionsMap[userID]
	if !ok {
		allPermissions = []JobPermission{}
	}

	totalItems := len(allPermissions)
	startIndex := (page - 1) * perPageVal
	endIndex := startIndex + perPageVal
	var paginatedPermissions []JobPermission
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedPermissions = allPermissions[startIndex:endIndex]
	} else {
		paginatedPermissions = []JobPermission{}
	}

	generateLinkHeader(w, r, page, perPageVal, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedPermissions)
}

// listFutureJobPermissionsHandler serves /v3/future_job_permissions with user_id query param filter.
func listFutureJobPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPageVal, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "user_id query parameter is required")
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid user_id '%s'.", userIDStr))
		return
	}

	allFuturePermissions, ok := mockUserFutureJobPermissionsMap[userID]
	if !ok {
		allFuturePermissions = []FutureJobPermission{}
	}

	totalItems := len(allFuturePermissions)
	startIndex := (page - 1) * perPageVal
	endIndex := startIndex + perPageVal
	var paginatedFuturePermissions []FutureJobPermission
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedFuturePermissions = allFuturePermissions[startIndex:endIndex]
	} else {
		paginatedFuturePermissions = []FutureJobPermission{}
	}

	generateLinkHeader(w, r, page, perPageVal, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedFuturePermissions)
}

// revokePermissionsHandler serves POST /v3/users/{id}/revoke_permissions.
func revokePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID, err := getUserIDFromPath(r.URL.Path, "/v3/users/")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	for i, user := range mockUsers {
		if user.ID == userID {
			mockUsers[i].SiteAdmin = false
			respondWithJSON(w, http.StatusOK, mockUsers[i])
			return
		}
	}
	respondWithError(w, http.StatusNotFound, fmt.Sprintf("User with ID %d not found", userID))
}

func main() {
	initMockData()
	mux := http.NewServeMux()

	// Auth token endpoint
	mux.HandleFunc("/token", tokenHandler)

	// v3 API endpoints
	mux.HandleFunc("/v3/users", listUsersHandler)
	mux.HandleFunc("/v3/user_roles", listUserRolesHandler)
	mux.HandleFunc("/v3/user_job_permissions", listUserJobPermissionsHandler)
	mux.HandleFunc("/v3/future_job_permissions", listFutureJobPermissionsHandler)

	mux.HandleFunc("/v3/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		trimmedPath := strings.TrimPrefix(path, "/v3/users/")
		trimmedPath = strings.TrimSuffix(trimmedPath, "/")
		parts := strings.Split(trimmedPath, "/")
		if len(parts) == 1 {
			if _, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				getUserByIDHandler(w, r)
				return
			}
		} else if len(parts) == 2 && parts[1] == "revoke_permissions" {
			revokePermissionsHandler(w, r)
			return
		}
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Endpoint %s not found or invalid.", path))
	})

	port := "9191"
	log.Printf("Starting Greenhouse Harvest API v3 Mock Server on port %s", port)
	log.Printf("Test endpoints: http://localhost:%s/v3/users", port)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
