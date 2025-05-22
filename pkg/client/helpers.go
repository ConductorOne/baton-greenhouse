package client

import (
	"encoding/json"
	"fmt"
	liburl "net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// RequestCompleted is a placeholder for the PermissionsToken when the query is completed.
const RequestCompleted = "request_completed_no_more_pages"

var findNextURL = regexp.MustCompile(`\<([^>]+)\>`)

func withBasicAuth(val string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", val)
}

func urlAddQuery(url string, params map[string]interface{}) (string, error) {
	p := liburl.Values{}
	for k, v := range params {
		switch value := v.(type) {
		case string:
			p.Add(k, value)
		case int:
			p.Add(k, strconv.Itoa(value))
		case bool:
			p.Add(k, strconv.FormatBool(value))
		default:
			continue
		}
	}

	parsed, err := liburl.Parse(url)
	if err != nil {
		return "", fmt.Errorf("cannot parse URL, error: %w", err)
	}

	parsed.RawQuery = p.Encode()

	return parsed.String(), nil
}

func getNextLink(raw string) string {
	found := strings.Replace(findNextURL.FindString(raw), "<", "", 1)
	return strings.Replace(found, ">", "", 1)
}

// JobPermissionPaginationTokens
// This pagination method could be modified to use the SDKs pagination bag instead.
// Probably, that would be the best way to go. It's something to analyze.
type JobPermissionPaginationTokens struct {
	JobPermissionsToken       string
	FutureJobPermissionsToken string
}

func SerializeTokens(tokens JobPermissionPaginationTokens) (string, error) {
	b, err := json.Marshal(tokens)
	if err != nil {
		return "", fmt.Errorf("cannot serialize custom pagination tokens: %w", err)
	}

	return string(b), nil
}

func DeserializeTokens(stringToken string) (JobPermissionPaginationTokens, error) {
	var tokens JobPermissionPaginationTokens
	if stringToken == "" {
		return tokens, nil
	}

	err := json.Unmarshal([]byte(stringToken), &tokens)
	if err != nil {
		return tokens, fmt.Errorf("cannot deserialize custom pagination tokens: %w", err)
	}

	return tokens, nil
}
