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
	Next string
	Last string
}

func (l *Link) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	parts := sep.Split(string(text), -1)

	if len(parts) < 2 || len(parts) == 3 {
		return fmt.Errorf("malformed input: %s", string(text))
	}

	for i, v := range parts {
		if strings.Contains(v, linkNext) {
			l.Next = string(cleanLinks.ReplaceAllString(parts[i-1], ""))
		}
		if strings.Contains(v, linkLast) {
			l.Last = string(cleanLinks.ReplaceAllString(parts[i-1], ""))
		}
	}

	if l.Next == l.Last && len(parts) == 4 {
		return fmt.Errorf("malformed input: %s", string(text))
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
