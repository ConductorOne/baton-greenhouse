package client

import (
	"context"
	"encoding/base64"
	"fmt"
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
	defer resp.Body.Close()

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

	var created models.User
	_, err = c.doRequest(ctx, http.MethodPost, parsedURL, &onBehalfOfID, body, &created, nil)
	if err != nil {
		return nil, err
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
	_, err = c.doRequest(ctx, http.MethodPatch, parsedURL, &user.ID, body, nil, nil)
	if err != nil {
		return err
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

	var user models.User
	_, err = c.doRequest(ctx, http.MethodGet, parsedURL, nil, nil, &user, nil)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *Client) doRequest(
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
			return res, fmt.Errorf("greenhouse API error: %s", errDetail)
		}
		if apiErr.APIMessage != "" {
			return res, fmt.Errorf("greenhouse API error: %s", apiErr.APIMessage)
		}
		return res, fmt.Errorf("request failed: %w", err)
	}

	return res, nil
}
