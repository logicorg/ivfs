package auth_test

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

var (
	keyA = bytes.Repeat([]byte("a"), auth.MinKeyLen)
	keyB = bytes.Repeat([]byte("b"), auth.MinKeyLen)
)

func newSigner(t *testing.T, keys ...[]byte) *auth.Signer {
	t.Helper()
	signer, err := auth.NewSigner(keys)
	require.NoError(t, err)
	return signer
}

// TestSigner_Session checks the attributes that keep the cookie away from
// scripts, other hosts, and cross-site posts.
func TestSigner_Session(t *testing.T) {
	now := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	cookie := newSigner(t, keyA).Session(uuid.Must(uuid.NewV7()), now)

	require.Equal(t, auth.SessionCookie, cookie.Name)
	require.NotEmpty(t, cookie.Value)
	require.Equal(t, "/", cookie.Path)
	require.Empty(t, cookie.Domain)
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	require.Equal(t, now.Add(auth.SessionLifetime), cookie.Expires)
	require.NoError(t, cookie.Valid())
}

func TestClearSession(t *testing.T) {
	cookie := auth.ClearSession()

	require.Equal(t, auth.SessionCookie, cookie.Name)
	require.Empty(t, cookie.Value)
	require.Negative(t, cookie.MaxAge)
	require.True(t, cookie.Secure)
	require.Equal(t, "/", cookie.Path)
}
