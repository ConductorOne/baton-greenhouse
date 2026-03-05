package models

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	sep        = regexp.MustCompile(`,|;`)
	linkNext   = `rel="next"`
	linkLast   = `rel="last"`
	cleanLinks = regexp.MustCompile(`<|>`)
)

type Link struct {
	Next string `json:"next,omitempty"`
	Last string `json:"last,omitempty"`
}

func (l *Link) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parts := sep.Split(string(text), -1)

	// v3 API may return just a single rel="next" link (2 parts) or
	// both rel="next" and rel="last" (4 parts). v1 had stricter parsing.
	if len(parts) < 2 {
		return fmt.Errorf("malformed input: %s", string(text))
	}

	for i, v := range parts {
		if strings.Contains(v, linkNext) {
			l.Next = strings.TrimSpace(cleanLinks.ReplaceAllString(parts[i-1], ""))
		}
		if strings.Contains(v, linkLast) {
			l.Last = strings.TrimSpace(cleanLinks.ReplaceAllString(parts[i-1], ""))
		}
	}

	return nil
}

func (l *Link) MarshalText() ([]byte, error) {
	next := fmt.Sprintf(`<%s>; rel="next"`, l.Next)
	last := fmt.Sprintf(`<%s>; rel="last"`, l.Last)

	if l.Next != "" {
		return []byte(strings.Join([]string{next, last}, ",")), nil
	}

	return []byte(last), nil
}
