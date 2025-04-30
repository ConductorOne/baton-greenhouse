package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var DefaultHost = "https://harvest.greenhouse.io"

type Client struct {
	baseURL         string
	user            string
	onBehalfOfEmail string
	httpClient      *uhttp.BaseHttpClient
}

func makeAuthorization(user string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:", user)))

	return fmt.Sprintf("Basic %s", encoded)
}

func New(ctx context.Context, baseURL, username, onBehalfOfEmail string) (*Client, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:         baseURL,
		user:            username,
		onBehalfOfEmail: onBehalfOfEmail,
		httpClient:      uhttp.NewBaseHttpClient(httpClient),
	}, nil
}

// https://developers.greenhouse.io/harvest.html#get-list-users.
func (c *Client) ListUsers(ctx context.Context, next string) ([]models.User, *v2.RateLimitDescription, string, error) {
	joinedURL, err := url.JoinPath(c.baseURL, "v1/users")
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

	var target []models.User
	if next != "" {
		qurl = next
	}
	parsedURL, err := url.Parse(qurl)
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	request, err := c.httpClient.NewRequest(ctx, http.MethodGet, parsedURL, uhttp.WithAcceptJSONHeader(), withBasicAuth(makeAuthorization(c.user)))
	if err != nil {
		return nil, nil, "", err
	}

	rl := v2.RateLimitDescription{}
	doOptions := []uhttp.DoOption{
		uhttp.WithRatelimitData(&rl),
		uhttp.WithJSONResponse(&target),
	}

	resp, err := c.httpClient.Do(request, doOptions...)
	if err != nil {
		return nil, nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, "", fmt.Errorf("API return with unexpected status code %d %s", resp.StatusCode, resp.Status)
	}
	// see https://developers.greenhouse.io/harvest.html#pagination
	link := &models.Link{}
	err = link.UnmarshalText([]byte(resp.Header.Get("Link")))
	if err != nil {
		return nil, nil, "", fmt.Errorf("cannot unmarshal value of header Link, error: %w", err)
	}

	return target, &rl, link.Next, nil
}

// GetAdminByEmail gets a site admin user by their email address.
// If the user is not found or is not site admin, it returns an error.
func (c *Client) GetAdminByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := c.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if !user.SiteAdmin {
		return nil, fmt.Errorf("user with email '%s' is not a site admin", email)
	}

	return user, nil
}

// https://developers.greenhouse.io/harvest.html#post-add-user.
func (c *Client) CreateUserAccount(ctx context.Context, onBehalfOfID int, email, firstName, lastName string) (*models.User, error) {
	body := map[string]interface{}{
		"first_name":        firstName,
		"last_name":         lastName,
		"email":             email,
		"send_email_invite": true,
	}

	endpoint, err := url.JoinPath(c.baseURL, "v1/users")
	if err != nil {
		return nil, fmt.Errorf("failed to join path for user creation: %w", err)
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	request, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPost,
		parsedURL,
		uhttp.WithJSONBody(body),
		uhttp.WithHeader("Content-Type", "application/json"),
		uhttp.WithHeader("Authorization", makeAuthorization(c.user)),
		uhttp.WithHeader("On-Behalf-Of", strconv.Itoa(onBehalfOfID)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	var created models.User
	doOptions := []uhttp.DoOption{
		uhttp.WithJSONResponse(&created),
	}

	resp, err := c.httpClient.Do(request, doOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to send user creation request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("greenhouse API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return &created, nil
}

func (c *Client) GetOnBehalfOfEmail() string {
	return c.onBehalfOfEmail
}

// RevokeUserSiteAdmin changes the permission level of a user to "basic".
// https://developers.greenhouse.io/harvest.html#patch-change-user-permission-level.
func (c *Client) RevokeUserSiteAdmin(ctx context.Context, id int) error {
	body := map[string]interface{}{
		"user": map[string]int{
			"user_id": id,
		},
		"level": "basic",
	}

	endpoint, err := url.JoinPath(c.baseURL, "v1/users/permission_level")
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

	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodPatch,
		parsedURL,
		uhttp.WithJSONBody(body),
		uhttp.WithAcceptJSONHeader(),
		withBasicAuth(makeAuthorization(c.user)),
		uhttp.WithHeader("On-Behalf-Of", strconv.Itoa(user.ID)),
		uhttp.WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		return fmt.Errorf("failed to build revoke request: %w", err)
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			switch resp.StatusCode {
			case http.StatusForbidden:
				return fmt.Errorf("forbidden (403) - likely trying to revoke a non-human user or yourself: %s", string(respBody))
			case http.StatusNotFound:
				return fmt.Errorf("user not found (404): %s", string(respBody))
			default:
				return fmt.Errorf("unexpected status code %d during revoke: %s", resp.StatusCode, string(respBody))
			}
		}
		return fmt.Errorf("revoke request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("greenhouse API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	endpoint, err := url.JoinPath(c.baseURL, "v1/users")
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

	var users models.User
	req, err := c.httpClient.NewRequest(
		ctx,
		http.MethodGet,
		parsedURL,
		uhttp.WithAcceptJSONHeader(),
		withBasicAuth(makeAuthorization(c.user)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req, uhttp.WithJSONResponse(&users))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return &users, nil
}
