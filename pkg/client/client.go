// Package client provides a client for interacting with the Greenhouse Harvest API.
package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	baseURL = "https://harvest.greenhouse.io"

	usersEPv1                    = "v1/users"
	userRolesEPv1                = "v1/user_roles"
	userJobPermissionsEPv1       = "v1/users/%d/permissions/jobs"
	userFutureJobPermissionsEPv1 = "v1/users/%d/permissions/future_jobs"
)

// GreenhouseClient is a client for the Greenhouse Harvest API.
type GreenhouseClient struct {
	user            string
	onBehalfOfEmail string
	httpClient      *uhttp.BaseHttpClient
}

func makeAuthorization(user string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:", user)))

	return fmt.Sprintf("Basic %s", encoded)
}

// ListUsers implemented based on the docs https://developers.greenhouse.io/harvest.html#get-list-users.
func (c *GreenhouseClient) ListUsers(ctx context.Context, next string) ([]models.User, *v2.RateLimitDescription, string, error) {
	joinedURL, err := url.JoinPath(baseURL, usersEPv1)
	if err != nil {
		return nil, nil, "", err
	}

	params := map[string]interface{}{
		"per_page":        500,
		"page":            1,
		"user_attributes": true,
	}
	qurl, err := urlAddQuery(joinedURL, params)
	if err != nil {
		return nil, nil, "", err
	}
	if next != "" {
		qurl = next
	}

	parsedURL, err := url.Parse(qurl)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	var users []models.User
	rl := &v2.RateLimitDescription{}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &users, rl)
	if err != nil {
		return nil, nil, "", err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	// see https://developers.greenhouse.io/harvest.html#pagination
	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	return users, rl, link.Next, nil
}

// GetAdminByEmail gets a site admin user by their email address.
// If the user is not found or is not site admin, it returns an error.
func (c *GreenhouseClient) GetAdminByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := c.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !user.SiteAdmin {
		return nil, fmt.Errorf("user with email '%s' is not a site admin", email)
	}

	return user, nil
}

// CreateUserAccount creates a new user account in Greenhouse.
// https://developers.greenhouse.io/harvest.html#post-add-user.
func (c *GreenhouseClient) CreateUserAccount(ctx context.Context, onBehalfOfID int, email, firstName, lastName string) (*models.User, error) {
	body := map[string]interface{}{
		"first_name":        firstName,
		"last_name":         lastName,
		"email":             email,
		"send_email_invite": true,
	}

	endpoint, err := url.JoinPath(baseURL, usersEPv1)
	if err != nil {
		return nil, fmt.Errorf("failed to join path for user creation: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	var created models.User
	resp, err := c.doRequest(ctx, http.MethodPost, parsedURL, &onBehalfOfID, body, &created, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	return &created, nil
}

// GetOnBehalfOfEmail returns the email used for on-behalf-of requests.
func (c *GreenhouseClient) GetOnBehalfOfEmail() string {
	return c.onBehalfOfEmail
}

// RevokeUserSiteAdmin changes the permission level of a user to "basic".
// https://developers.greenhouse.io/harvest.html#patch-change-user-permission-level.
func (c *GreenhouseClient) RevokeUserSiteAdmin(ctx context.Context, id int) error {
	body := map[string]interface{}{
		"user": map[string]int{
			"user_id": id,
		},
		"level": "basic",
	}

	endpoint, err := url.JoinPath(baseURL, "v1/users/permission_level")
	if err != nil {
		return fmt.Errorf("failed to join path for revoke: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	user, err := c.GetAdminByEmail(ctx, c.onBehalfOfEmail)
	if err != nil {
		return fmt.Errorf("failed to get user by email: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPatch, parsedURL, &user.ID, body, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	return nil
}

// GetUserByEmail retrieves a user by their email address.
func (c *GreenhouseClient) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	endpoint, err := url.JoinPath(baseURL, usersEPv1)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	params := map[string]interface{}{
		"email":           email,
		"user_attributes": true,
	}
	queryURL, err := urlAddQuery(endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to add query params: %w", err)
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	var user models.User
	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &user, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	return &user, nil
}

// GetJobPermissionsOfAUser receives the ID of a User and requests all the Job Permissions it has.
// Docs link: https://developers.greenhouse.io/harvest.html#get-list-job-permissions
//
// According to the documentation, this endpoint is only intended for use with 'Job Admin'
// and/or 'Interviewer' users, as these roles are assigned on a per-job basis.
// Users that are 'Site Admins' have permissions on all public jobs and will return an empty array.
// 'Basic users' cannot be assigned to any jobs and will also return an empty array.
func (c *GreenhouseClient) GetJobPermissionsOfAUser(ctx context.Context, tokens *JobPermissionPaginationTokens, userID int) ([]models.JobPermission, *v2.RateLimitDescription, error) {
	var jobPermissions []models.JobPermission
	var rateLimitData v2.RateLimitDescription
	var queryURL string

	if tokens.JobPermissionsToken == RequestCompleted {
		return nil, nil, nil
	}

	endpointURL, err := url.JoinPath(baseURL, fmt.Sprintf(userJobPermissionsEPv1, userID))
	if err != nil {
		return nil, nil, err
	}

	nextToken := tokens.JobPermissionsToken
	if nextToken == "" {
		params := map[string]interface{}{
			"per_page": 500,
			"page":     1,
		}
		queryURL, err = urlAddQuery(endpointURL, params)
		if err != nil {
			return nil, nil, err
		}
	} else {
		queryURL = nextToken
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &jobPermissions, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	// Pagination Docs https://developers.greenhouse.io/harvest.html#pagination
	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, &rateLimitData, fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	if link.Next == "" {
		tokens.JobPermissionsToken = RequestCompleted
	} else {
		tokens.JobPermissionsToken = link.Next
	}

	return jobPermissions, &rateLimitData, nil
}

// GetFutureJobPermissionsOfAUser receives the ID of a User and requests all the Future Job Permissions it has.
// Docs link: https://developers.greenhouse.io/harvest.html#get-list-future-job-permissions
func (c *GreenhouseClient) GetFutureJobPermissionsOfAUser(ctx context.Context, tokens *JobPermissionPaginationTokens, userID int) ([]models.FutureJobPermission, *v2.RateLimitDescription, error) {
	var futureJobPermissions []models.FutureJobPermission
	var rateLimitData v2.RateLimitDescription
	var queryURL string

	if tokens.FutureJobPermissionsToken == RequestCompleted {
		return nil, nil, nil
	}

	endpointURL, err := url.JoinPath(baseURL, fmt.Sprintf(userFutureJobPermissionsEPv1, userID))
	if err != nil {
		return nil, nil, err
	}

	nextToken := tokens.FutureJobPermissionsToken
	if nextToken == "" {
		params := map[string]interface{}{
			"per_page": 500,
			"page":     1,
		}
		queryURL, err = urlAddQuery(endpointURL, params)
		if err != nil {
			return nil, nil, err
		}
	} else {
		queryURL = nextToken
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &futureJobPermissions, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	// Pagination Docs https://developers.greenhouse.io/harvest.html#pagination
	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, &rateLimitData, fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	if link.Next == "" {
		tokens.FutureJobPermissionsToken = RequestCompleted
	} else {
		tokens.FutureJobPermissionsToken = link.Next
	}

	return futureJobPermissions, &rateLimitData, nil
}

// ListUserRoles - Docs link: https://developers.greenhouse.io/harvest.html#get-list-user-roles
func (c *GreenhouseClient) ListUserRoles(ctx context.Context, nextPageURL string) ([]models.UserRole, *v2.RateLimitDescription, string, error) {
	var userRoles []models.UserRole
	var rateLimitData v2.RateLimitDescription

	endpointURL, err := url.JoinPath(baseURL, userRolesEPv1)
	if err != nil {
		return nil, nil, "", err
	}

	params := map[string]interface{}{
		"per_page": 500,
		"page":     1,
	}
	queryURL, err := urlAddQuery(endpointURL, params)
	if err != nil {
		return nil, nil, "", err
	}
	if nextPageURL != "" {
		queryURL = nextPageURL
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &userRoles, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, "", err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	// Pagination Docs https://developers.greenhouse.io/harvest.html#pagination
	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	return userRoles, &rateLimitData, link.Next, nil
}

// RetrieveUserData - Docs link: https://developers.greenhouse.io/harvest.html#get-retrieve-user
func (c *GreenhouseClient) RetrieveUserData(ctx context.Context, userID string) (*models.User, *v2.RateLimitDescription, error) {
	var userData *models.User
	var rateLimitData v2.RateLimitDescription

	endpointURL, err := url.JoinPath(baseURL, usersEPv1, userID)
	if err != nil {
		return nil, nil, err
	}

	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &userData, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close() //nolint:errcheck // standard pattern, error is not critical

	return userData, &rateLimitData, nil
}

// doRequest builds and sends an HTTP request to the Greenhouse API with the given method, URL,
// optional "on-behalf-of" header, and request body. It unmarshalls the JSON response into the
// provided target and handles Greenhouse-specific API error formatting.
//
// The HTTP response body is automatically read and closed inside the underlying HTTP client,
// so callers do not need to call resp.Body.Close() manually.
//
// If a rate limit description object is provided, it is populated with rate limit data from the response.
func (c *GreenhouseClient) doRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	onBehalfOfID *int,
	body any,
	target any,
	rl *v2.RateLimitDescription,
) (*http.Response, error) {
	var apiErr models.APIError

	opts := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		withBasicAuth(makeAuthorization(c.user)),
	}

	if onBehalfOfID != nil {
		opts = append(opts, uhttp.WithHeader("On-Behalf-Of", strconv.Itoa(*onBehalfOfID)))
	}
	if body != nil {
		opts = append(opts, uhttp.WithJSONBody(body))
		opts = append(opts, uhttp.WithHeader("Content-Type", "application/json"))
	}

	req, err := c.httpClient.NewRequest(ctx, method, url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	doOpts := []uhttp.DoOption{
		uhttp.WithJSONResponse(target),
		uhttp.WithErrorResponse(&apiErr),
	}
	if rl != nil {
		doOpts = append(doOpts, uhttp.WithRatelimitData(rl))
	}

	res, err := c.httpClient.Do(req, doOpts...)
	if err != nil {
		if len(apiErr.Errors) > 0 {
			errDetail := apiErr.Errors[0].Message
			if apiErr.Errors[0].Field != "" {
				errDetail += fmt.Sprintf(" (field: %s)", apiErr.Errors[0].Field)
			}

			return res, errors.Join(err, fmt.Errorf("greenhouse API error: %s", errDetail))
		}

		if apiErr.APIMessage != "" {
			return res, errors.Join(err, fmt.Errorf("greenhouse API message error: %w", err))
		}

		return res, errors.Join(err, fmt.Errorf("request failed: %w", err))
	}

	return res, nil
}

// New creates a new GreenhouseClient with the given credentials.
func New(ctx context.Context, username, onBehalfOfEmail string) (*GreenhouseClient, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &GreenhouseClient{
		user:            username,
		onBehalfOfEmail: onBehalfOfEmail,
		httpClient:      uhttp.NewBaseHttpClient(httpClient),
	}, nil
}
