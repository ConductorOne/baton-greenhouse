package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

type Client struct {
	baseURL    string
	user       string
	httpClient *uhttp.BaseHttpClient
}

func makeAuthorization(user string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:", user)))

	return fmt.Sprintf("Basic %s", encoded)
}

func (c *Client) ListUsers(ctx context.Context) ([]*models.User, *v2.RateLimitDescription, error) {
	joinedURL, err := url.JoinPath(c.baseURL, "v1/users")
	if err != nil {
		return nil, nil, err
	}

	params := map[string]interface{}{}
	qurl, err := urlAddQuery(joinedURL, params)
	if err != nil {
		return nil, nil, err
	}

	var target []*models.User
	parsedURL, err := url.Parse(qurl)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse URL, error: %w", err)
	}

	request, err := c.httpClient.NewRequest(ctx, http.MethodGet, parsedURL, uhttp.WithAcceptJSONHeader(), withBasicAuth(makeAuthorization(c.user)))
	if err != nil {
		return nil, nil, err
	}

	var rl *v2.RateLimitDescription
	doOptions := []uhttp.DoOption{
		uhttp.WithRatelimitData(rl),
		uhttp.WithJSONResponse(target),
	}

	resp, err := c.httpClient.Do(request, doOptions...)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("API return with unexpected status code %d %s", resp.StatusCode, resp.Status)
	}

	return target, rl, nil
}
