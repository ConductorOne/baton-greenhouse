package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUrlAddQuery(t *testing.T) {
	params := map[string]interface{}{
		"per_page": 500,
		"user_id":  123,
	}

	result, err := urlAddQuery("https://harvest.greenhouse.io/v3/user_job_permissions", params)
	require.NoError(t, err)
	require.Contains(t, result, "per_page=500")
	require.Contains(t, result, "user_id=123")
}

func TestSerializeDeserializeTokens(t *testing.T) {
	tokens := JobPermissionPaginationTokens{ //nolint:gosec // pagination token URLs, not credentials
		JobPermissionsToken:       "https://harvest.greenhouse.io/v3/user_job_permissions?cursor=abc123",
		FutureJobPermissionsToken: RequestCompleted,
	}

	serialized, err := SerializeTokens(tokens)
	require.NoError(t, err)
	require.NotEmpty(t, serialized)

	deserialized, err := DeserializeTokens(serialized)
	require.NoError(t, err)
	require.Equal(t, tokens.JobPermissionsToken, deserialized.JobPermissionsToken)
	require.Equal(t, tokens.FutureJobPermissionsToken, deserialized.FutureJobPermissionsToken)
}

func TestDeserializeEmptyToken(t *testing.T) {
	tokens, err := DeserializeTokens("")
	require.NoError(t, err)
	require.Empty(t, tokens.JobPermissionsToken)
	require.Empty(t, tokens.FutureJobPermissionsToken)
}
