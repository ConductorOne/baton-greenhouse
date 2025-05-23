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

// --- Data Structures ---

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
	Email        string    `json:"email_addresses"` // Simplified
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

type User struct {
	ID         int64     `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
	Disabled   bool      `json:"disabled"`
	SiteAdmin  bool      `json:"site_admin"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	EmployeeID *string   `json:"employee_id,omitempty"`
}

type UserRole struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type JobPermission struct {
	ID         int64 `json:"id"`
	JobID      int64 `json:"job_id"`
	UserRoleID int64 `json:"user_role_id"`
}

type FutureJobPermission struct {
	ID                   int64   `json:"id"`
	OfficeID             *int64  `json:"office_id,omitempty"`
	ExternalOfficeID     *string `json:"external_office_id,omitempty"`
	DepartmentID         *int64  `json:"department_id,omitempty"`
	ExternalDepartmentID *string `json:"external_department_id,omitempty"`
	UserRoleID           int64   `json:"user_role_id"`
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
	rng = rand.New(source)

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
		if i%3 == 0 {
			job.Department = nil
			// &struct {
			//	ID   int64
			//	Name string
			// }{ID: int64((i % 3) + 100), Name: fmt.Sprintf("Engineering Dept %d", (i%3)+100)}
		}
		if i%2 == 0 {
			job.Offices = nil
			//	[]struct {
			//	ID   int64
			//	Name string
			// }{
			//	{ID: int64((i % 2) + 200), Name: fmt.Sprintf("Office Location %d", (i%2)+200)},
			// }
			// if i%4 == 0 && len(job.Offices) > 0 {
			//	job.Offices = append(job.Offices, struct {
			//		ID   int64
			//		Name string
			//	}{ID: int64(((i%2)+1)%2 + 200), Name: fmt.Sprintf("Aux Office %d", (((i%2)+1)%2 + 200))})
			// }
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
			//	&struct {
			//	ID   int64
			//	Name string
			// }{ID: int64((i % 4) + 1), Name: fmt.Sprintf("Source %d", (i%4)+1)},
			// CreditedTo: &struct {
			//	ID   int64
			//	Name string
			// }{ID: int64((i % 10) + 1), Name: fmt.Sprintf("Recruiter %d", (i%10)+1)},
		})
	}

	// Users
	for i := 1; i <= 50; i++ {
		empID := fmt.Sprintf("EMP%04d", i)
		mockUsers = append(mockUsers, User{
			ID: int64(i), FirstName: fmt.Sprintf("UserFirst%d", i), LastName: fmt.Sprintf("UserLast%d", i),
			Email: fmt.Sprintf("user%d@example-corp.com", i), Disabled: i%10 == 0, SiteAdmin: i%15 == 0,
			CreatedAt: now.Add(-time.Duration(i*7*24) * time.Hour), UpdatedAt: now.Add(-time.Duration(i*24) * time.Hour), EmployeeID: &empID,
		})
	}

	// User Roles
	mockUserRoles = []UserRole{
		{ID: 1, Type: "job_admin", Name: "Job Admin - Marketing Department"},
		{ID: 2, Type: "interviewer", Name: "Technical Interviewer - Backend"},
		{ID: 3, Type: "interviewer", Name: "Cultural Fit Interviewer"},
		{ID: 4, Type: "job_admin", Name: "Job Admin - Engineering Department"},
		{ID: 5, Type: "job_admin", Name: "Job Admin - All Jobs"},
		{ID: 6, Type: "job_admin", Name: "Hiring Manager"},
		{ID: 7, Type: "interviewer", Name: "Accounting Department"},
		{ID: 8, Type: "interviewer", Name: "Technical Interviewer - Frontend"},
		{ID: 9, Type: "interviewer", Name: "Executive Interviewer"},
		// {ID: 10, Type: "system", Name: "Basic"},
		// {ID: 11, Type: "system", Name: "Site Admin"},
	}

	// User Job Permissions & Future Job Permissions
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
					if r.Type == "job_admin" || r.Type == "interviewer" {
						relevantRoles = append(relevantRoles, r)
					}
				}
				if len(relevantRoles) > 0 {
					userRole = relevantRoles[rng.Intn(len(relevantRoles))]
				} else {
					userRole = mockUserRoles[rng.Intn(len(mockUserRoles))]
				}
			} else {
				userRole = UserRole{ID: 999, Type: "fallback", Name: "Fallback Role"}
			}
			jobPerms = append(jobPerms, JobPermission{
				ID: nextPermissionID, JobID: job.ID, UserRoleID: userRole.ID,
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
					if r.Type == "job_admin" {
						relevantRoles = append(relevantRoles, r)
					}
				}
				if len(relevantRoles) == 0 {
					for _, r := range mockUserRoles {
						if r.Type == "interviewer" {
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
				userRole = UserRole{ID: 998, Type: "fallback_future", Name: "Fallback Future Role"}
			}
			var officeID *int64
			var extOfficeID *string
			var deptID *int64
			var extDeptID *string
			if len(mockJobs) > 0 && rng.Float32() < 0.7 {
				jobWithMeta := mockJobs[rng.Intn(len(mockJobs))]
				if len(jobWithMeta.Offices) > 0 && rng.Float32() < 0.8 {
					office := jobWithMeta.Offices[rng.Intn(len(jobWithMeta.Offices))]
					officeID = &office.ID
					extOfficeStr := fmt.Sprintf("ext-office-%d", office.ID)
					extOfficeID = &extOfficeStr
				}
				if jobWithMeta.Department != nil && rng.Float32() < 0.8 {
					deptID = &jobWithMeta.Department.ID
					extDeptStr := fmt.Sprintf("ext-dept-%d", jobWithMeta.Department.ID)
					extDeptID = &extDeptStr
				}
			}
			futureJobPerms = append(futureJobPerms, FutureJobPermission{
				ID: nextPermissionID, OfficeID: officeID, ExternalOfficeID: extOfficeID,
				DepartmentID: deptID, ExternalDepartmentID: extDeptID, UserRoleID: userRole.ID,
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

func parsePagination(r *http.Request) (page int, perPage int, err error) {
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	page = 1
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			return 0, 0, fmt.Errorf("invalid 'page' parameter: must be a positive integer")
		}
	}
	perPage = 50
	if perPageStr != "" {
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil || perPage <= 0 {
			return 0, 0, fmt.Errorf("invalid 'per_page' parameter: must be a positive integer")
		}
		if perPage > 100 {
			perPage = 100
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
	if currentPage > 1 {
		links = append(links, fmt.Sprintf("<%s>; rel=\"prev\"", buildLinkURL(currentPage-1)))
		if currentPage > 1 {
			links = append(links, fmt.Sprintf("<%s>; rel=\"first\"", buildLinkURL(1)))
		}
	}
	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		if w.Header().Get("X-RateLimit-Limit") != "" && statusCode == http.StatusTooManyRequests {
			return
		}
		// If marshalling fails, we need to send a generic server error if no response has been started
		// Check if headers already sent
		if w.Header().Get("Content-Type") == "" { // A simple check
			http.Error(w, `{"message": "Internal Server Error while marshalling JSON", "errors": []}`, http.StatusInternalServerError)
		}
		return
	}
	if w.Header().Get("X-RateLimit-Limit") != "" && statusCode != http.StatusTooManyRequests {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
	} else if w.Header().Get("X-RateLimit-Limit") == "" { // Should only happen if handleRateLimiting was skipped
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
	} // For 429, headers (including Content-Type) and status code are assumed to be set by respondWithError.

	_, err = w.Write(response)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Note: WriteHeader should ideally be called only once.
	// If handleRateLimiting calls this for 429, it will set the status code.
	// We ensure it's set here for other error paths too.
	// A robust way is to check if `w.Header().Get("Status-Code-Written") == "true"` if we had such a mechanism.
	// For now, this might call WriteHeader again if already called by rate limiter,
	// but net/http's ResponseWriter typically ignores subsequent WriteHeader calls.
	w.WriteHeader(statusCode)

	apiErr := APIError{
		APIMessage: message,
		// Initialize with an empty slice. It will be marshalled as "errors": []
		Errors: make([]struct {
			Message string `json:"message"`
			Field   string `json:"field"`
		}, 0),
	}

	// Attempt to populate the Errors slice for specific, identifiable field errors
	if statusCode == http.StatusBadRequest {
		if strings.Contains(message, "'page' parameter") {
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: "Invalid value provided for 'page' parameter.", // More specific if possible
				Field:   "page",
			})
		} else if strings.Contains(message, "'per_page' parameter") {
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: "Invalid value provided for 'per_page' parameter.", // More specific if possible
				Field:   "per_page",
			})
		} else if strings.Contains(message, "Invalid User ID format") {
			apiErr.Errors = append(apiErr.Errors, struct {
				Message string `json:"message"`
				Field   string `json:"field"`
			}{
				Message: message,   // Use the original detailed message
				Field:   "user_id", // Or more generally "path_parameter"
			})
		}
		// Add more cases here if other specific field errors can be identified from the message string
	}
	// For general 404s, 429s, 500s, the top-level APIMessage is usually sufficient, and 'errors' list stays empty.

	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		log.Printf("Error encoding APIError JSON response: %v", err)
		// If encoding this fails, as a last resort, write plain text.
		// This might happen if headers were already flushed.
		// http.Error(w, `{"message":"Failed to encode error details."}`, http.StatusInternalServerError) // Avoid this if headers are already sent
	}
}

// --- Handlers ---

func listCandidatesHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPage, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	totalItems := len(mockCandidates)
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	var paginatedData []Candidate
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedData = mockCandidates[startIndex:endIndex]
	} else {
		paginatedData = []Candidate{}
	}
	generateLinkHeader(w, r, page, perPage, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedData)
}

func listJobsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPage, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	totalItems := len(mockJobs)
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	var paginatedData []Job
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedData = mockJobs[startIndex:endIndex]
	} else {
		paginatedData = []Job{}
	}
	generateLinkHeader(w, r, page, perPage, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedData)
}

func listApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPage, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	totalItems := len(mockApplications)
	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	var paginatedData []Application
	if startIndex < totalItems {
		if endIndex > totalItems {
			endIndex = totalItems
		}
		paginatedData = mockApplications[startIndex:endIndex]
	} else {
		paginatedData = []Application{}
	}
	generateLinkHeader(w, r, page, perPage, totalItems)
	respondWithJSON(w, http.StatusOK, paginatedData)
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
	userID, err := getUserIDFromPath(r.URL.Path, "/v1/users/")
	if err != nil {
		// Pass the specific error message from getUserIDFromPath for better detail
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
	} // err.Error() used as message
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
	} // Generic message
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	// More specific message for parsing error
	if err != nil {
		return 0, fmt.Errorf("invalid User ID format in URL path: '%s' is not a number", parts[0])
	}
	return userID, nil
}

func listUserJobPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPageVal, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/users/"), "/")
	if len(pathParts) < 1 {
		respondWithError(w, http.StatusBadRequest, "Invalid path format for user job permissions.")
		return
	}
	userID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid User ID '%s' in path for job permissions.", pathParts[0]))
		return
	}

	userExists := false
	for _, u := range mockUsers {
		if u.ID == userID {
			userExists = true
			break
		}
	}
	if !userExists {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("User ID %d not found when listing job permissions.", userID))
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

func listUserFutureJobPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	if handleRateLimiting(w) {
		return
	}
	page, perPageVal, err := parsePagination(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/users/"), "/")
	if len(pathParts) < 1 {
		respondWithError(w, http.StatusBadRequest, "Invalid path format for user future job permissions.")
		return
	}
	userID, err := strconv.ParseInt(pathParts[0], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid User ID '%s' in path for future job permissions.", pathParts[0]))
		return
	}

	userExists := false
	for _, u := range mockUsers {
		if u.ID == userID {
			userExists = true
			break
		}
	}
	if !userExists {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("User ID %d not found when listing future job permissions.", userID))
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

func main() {
	initMockData()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/candidates", listCandidatesHandler)
	mux.HandleFunc("/v1/jobs", listJobsHandler)
	mux.HandleFunc("/v1/applications", listApplicationsHandler)
	mux.HandleFunc("/v1/users", listUsersHandler)
	mux.HandleFunc("/v1/user_roles", listUserRolesHandler)

	mux.HandleFunc("/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		trimmedPath := strings.TrimPrefix(path, "/v1/users/")
		trimmedPath = strings.TrimSuffix(trimmedPath, "/")
		parts := strings.Split(trimmedPath, "/")
		if len(parts) == 1 {
			// Attempt to parse as ID. If it's not a number, it's not a valid user ID path.
			if _, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				getUserByIDHandler(w, r)
				return
			}
		} else if len(parts) == 3 && parts[1] == "permissions" {
			if parts[2] == "jobs" {
				listUserJobPermissionsHandler(w, r)
				return
			}
			if parts[2] == "future_jobs" {
				listUserFutureJobPermissionsHandler(w, r)
				return
			}
		}
		// If not a recognized pattern, fall through to NotFound.
		// Pass the original message from error sources for more context.
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("Endpoint %s not found or invalid.", path))
	})

	/*
		INFORMATION ABOUT THE AVAILABLE ENDPOINTS COULD BE DISPLAYED TO THE REQUESTER IF NEEDED
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				respondWithError(w, http.StatusNotFound, fmt.Sprintf("Endpoint %s not found.", r.URL.Path))
				return
			}
			fmt.Fprint(w, "Greenhouse Harvest API v1 Mock Server is running.")
			fmt.Fprint(w, "Available endpoints (GET):")
			fmt.Fprint(w, "  /v1/candidates")
			fmt.Fprint(w, "  /v1/jobs")
			fmt.Fprint(w, "  /v1/applications")
			fmt.Fprint(w, "  /v1/users (List)")
			fmt.Fprint(w, "  /v1/users/{id} (Specific user)")
			fmt.Fprint(w, "  /v1/user_roles (List of UserRole objects with type and name)")
			fmt.Fprint(w, "  /v1/users/{user_id}/permissions/jobs (Paginated list of JobPermission objects)")
			fmt.Fprint(w, "  /v1/users/{user_id}/permissions/future_jobs (Paginated list of FutureJobPermission objects)")
			fmt.Fprint(w, "Supports 'page' and 'per_page' query parameters for list endpoints.")
			fmt.Fprint(w, fmt.Sprintf("Rate limit: %d requests per %v.", rateLimitLimit, rateLimitResetDelta))
		})
	*/
	port := "9191"
	log.Printf("Starting Greenhouse Harvest API v1 Mock Server on port %s", port)
	log.Printf("Test pagination error: http://localhost:%s/v1/users?page=abc", port)
	log.Printf("Test User Not Found error: http://localhost:%s/v1/users/99999", port)

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
