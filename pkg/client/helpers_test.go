package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetNextLink(t *testing.T) {
	input := `<https://harvest.greenhouse.io/v1/candidates?page=2&per_page=2>; rel="next",
<https://harvest.greenhouse.io/v1/candidates?page=474&per_page=2>; rel="last"`

	expected := "https://harvest.greenhouse.io/v1/candidates?page=2&per_page=2"

	result := getNextLink(input)

	require.Equal(t, expected, result)
}
