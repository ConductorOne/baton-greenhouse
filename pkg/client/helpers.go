package client

import (
	"fmt"
	liburl "net/url"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

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
