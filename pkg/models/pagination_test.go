package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalLink(t *testing.T) {
	link := &Link{Next: "https://example.com/?page=2", Last: "https://example.com/?page=1"}

	expected := []byte(`<https://example.com/?page=2>; rel="next",<https://example.com/?page=1>; rel="last"`)
	result, err := link.MarshalText()

	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestUnmarshalLink(t *testing.T) {
	input := []byte(`<https://example.com/?page=2>; rel="next",<https://example.com/?page=1>; rel="last"`)
	link := &Link{}

	err := link.UnmarshalText(input)
	require.NoError(t, err)

	require.Equal(t, "https://example.com/?page=2", link.Next)
	require.Equal(t, "https://example.com/?page=1", link.Last)
}

func TestUnmarshalLinkOnlyLast(t *testing.T) {
	input := []byte(`<https://example.com/?page=1>; rel="last"`)
	link := &Link{}

	err := link.UnmarshalText(input)
	require.NoError(t, err)

	require.Equal(t, "", link.Next)
	require.Equal(t, "https://example.com/?page=1", link.Last)
}
