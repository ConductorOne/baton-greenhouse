// Package client provides a client for interacting with the Greenhouse Harvest API v3.
package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	baseURL = "https://harvest.greenhouse.io"
	authURL = "https://auth.greenhouse.io/token"

	usersEP              = "v3/users"
	userRolesEP          = "v3/user_roles"
	userJobPermissionsEP = "v3/user_job_permissions"
	futureJobPermsEP     = "v3/future_job_permissions"
	revokePermissionsEP  = "v3/users/%d/revoke_permissions"
	deactivateUserEP     = "v3/users/%d/deactivate"
)

// GreenhouseClient is a client for the Greenhouse Harvest API v3.
type GreenhouseClient struct {
	clientID          string
	clientSecret      string
	onBehalfOfUserID  string
	httpClient        *uhttp.BaseHttpClient

	tokenMu     sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// ensureToken obtains or refreshes the OAuth2 bearer token using the client credentials flow.
// POST https://auth.greenhouse.io/token with Basic auth (client_id:client_secret)
// and grant_type=client_credentials&sub=<user_id>.
func (c *GreenhouseClient) ensureToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock.
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	formData := url.Values{
		"grant_type": {"client_credentials"},
	}
	if c.onBehalfOfUserID != "" {
		formData.Set("sub", c.onBehalfOfUserID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	basicAuth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp models.TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	expiry, err := time.Parse(time.RFC3339, tokenResp.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to parse token expiry: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = expiry

	l := ctxzap.Extract(ctx)
	l.Debug("obtained new Greenhouse API token", zap.Time("expires_at", expiry))

	return c.accessToken, nil
}

// ListUsers fetches all users from the Harvest API v3.
// https://harvestdocs.greenhouse.io/reference/get_v3-users
func (c *GreenhouseClient) ListUsers(ctx context.Context, next string) ([]models.User, *v2.RateLimitDescription, string, error) {
	var queryURL string

	if next != "" {
		queryURL = next
	} else {
		joinedURL, err := url.JoinPath(baseURL, usersEP)
		if err != nil {
			return nil, nil, "", err
		}
		params := map[string]interface{}{
			"per_page": 500,
		}
		queryURL, err = urlAddQuery(joinedURL, params)
		if err != nil {
			return nil, nil, "", err
		}
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	var users []models.User
	rl := &v2.RateLimitDescription{}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, &users, rl)
	if err != nil {
		return nil, nil, "", err
	}
	defer resp.Body.Close()

	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	return users, rl, link.Next, nil
}

// GetUserByID retrieves a single user by their ID.
func (c *GreenhouseClient) GetUserByID(ctx context.Context, userID string) (*models.User, *v2.RateLimitDescription, error) {
	var userData *models.User
	var rateLimitData v2.RateLimitDescription

	endpointURL, err := url.JoinPath(baseURL, usersEP, userID)
	if err != nil {
		return nil, nil, err
	}

	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, &userData, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close()

	return userData, &rateLimitData, nil
}

// CreateUserAccount creates a new user account in Greenhouse via the v3 API.
// https://harvestdocs.greenhouse.io/reference/post_v3-users
func (c *GreenhouseClient) CreateUserAccount(ctx context.Context, email, firstName, lastName string) (*models.User, error) {
	body := map[string]interface{}{
		"first_name":        firstName,
		"last_name":         lastName,
		"email":             email,
		"send_email_invite": true,
	}

	endpoint, err := url.JoinPath(baseURL, usersEP)
	if err != nil {
		return nil, fmt.Errorf("failed to join path for user creation: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	var created models.User
	resp, err := c.doRequest(ctx, http.MethodPost, parsedURL, body, &created, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &created, nil
}

// RevokeUserSiteAdmin revokes all permissions from a user (setting them to basic).
// v3 endpoint: POST /v3/users/{id}/revoke_permissions
func (c *GreenhouseClient) RevokeUserSiteAdmin(ctx context.Context, id int) error {
	endpoint, err := url.JoinPath(baseURL, fmt.Sprintf(revokePermissionsEP, id))
	if err != nil {
		return fmt.Errorf("failed to join path for revoke: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, parsedURL, nil, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ListUserJobPermissions lists user job permissions from the v3 API.
// In v3, job permissions are a top-level resource at /v3/user_job_permissions
// and can be filtered by user_id.
func (c *GreenhouseClient) ListUserJobPermissions(ctx context.Context, tokens *JobPermissionPaginationTokens, userID int) ([]models.JobPermission, *v2.RateLimitDescription, error) {
	var jobPermissions []models.JobPermission
	var rateLimitData v2.RateLimitDescription
	var queryURL string

	if tokens.JobPermissionsToken == RequestCompleted {
		return nil, nil, nil
	}

	nextToken := tokens.JobPermissionsToken
	if nextToken == "" {
		endpointURL, err := url.JoinPath(baseURL, userJobPermissionsEP)
		if err != nil {
			return nil, nil, err
		}
		// In v3, filter by user_id as a query parameter on the top-level endpoint.
		params := map[string]interface{}{
			"per_page": 500,
			"user_id":  userID,
		}
		queryURL, err = urlAddQuery(endpointURL, params)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Cursor-based pagination: the next URL is used as-is (no additional params allowed).
		queryURL = nextToken
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, &jobPermissions, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close()

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

// ListFutureJobPermissions lists future job permissions from the v3 API.
// In v3, future job permissions are a top-level resource at /v3/future_job_permissions
// and can be filtered by user_id.
func (c *GreenhouseClient) ListFutureJobPermissions(ctx context.Context, tokens *JobPermissionPaginationTokens, userID int) ([]models.FutureJobPermission, *v2.RateLimitDescription, error) {
	var futureJobPermissions []models.FutureJobPermission
	var rateLimitData v2.RateLimitDescription
	var queryURL string

	if tokens.FutureJobPermissionsToken == RequestCompleted {
		return nil, nil, nil
	}

	nextToken := tokens.FutureJobPermissionsToken
	if nextToken == "" {
		endpointURL, err := url.JoinPath(baseURL, futureJobPermsEP)
		if err != nil {
			return nil, nil, err
		}
		params := map[string]interface{}{
			"per_page": 500,
			"user_id":  userID,
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

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, &futureJobPermissions, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, err
	}
	defer resp.Body.Close()

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

// ListUserRoles fetches user roles from the Harvest API v3.
// https://harvestdocs.greenhouse.io/reference/get_v3-user-roles
func (c *GreenhouseClient) ListUserRoles(ctx context.Context, nextPageURL string) ([]models.UserRole, *v2.RateLimitDescription, string, error) {
	var userRoles []models.UserRole
	var rateLimitData v2.RateLimitDescription

	var queryURL string
	if nextPageURL != "" {
		queryURL = nextPageURL
	} else {
		endpointURL, err := url.JoinPath(baseURL, userRolesEP)
		if err != nil {
			return nil, nil, "", err
		}
		params := map[string]interface{}{
			"per_page": 500,
		}
		queryURL, err = urlAddQuery(endpointURL, params)
		if err != nil {
			return nil, nil, "", err
		}
	}

	parsedURL, err := url.Parse(queryURL)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, parsedURL, nil, &userRoles, &rateLimitData)
	if err != nil {
		return nil, &rateLimitData, "", err
	}
	defer resp.Body.Close()

	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	return userRoles, &rateLimitData, link.Next, nil
}

// doRequest builds and sends an HTTP request to the Greenhouse API v3 with the given method, URL,
// and request body. It uses Bearer token authentication (OAuth2 JWT).
func (c *GreenhouseClient) doRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	body any,
	target any,
	rl *v2.RateLimitDescription,
) (*http.Response, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain access token: %w", err)
	}

	var apiErr models.APIError

	opts := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token)),
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
			return res, errors.Join(err, fmt.Errorf("greenhouse API message error: %s", apiErr.APIMessage))
		}

		return res, fmt.Errorf("request failed: %w", err)
	}

	return res, nil
}

// GetOnBehalfOfUserID returns the user ID used for on-behalf-of requests.
func (c *GreenhouseClient) GetOnBehalfOfUserID() string {
	return c.onBehalfOfUserID
}

// New creates a new GreenhouseClient with the given OAuth2 credentials.
func New(ctx context.Context, clientID, clientSecret, onBehalfOfUserID string) (*GreenhouseClient, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &GreenhouseClient{
		clientID:         clientID,
		clientSecret:     clientSecret,
		onBehalfOfUserID: onBehalfOfUserID,
		httpClient:       uhttp.NewBaseHttpClient(httpClient),
	}, nil
}
