package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

func TestSigner_SessionUser(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())
	issued := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	session := newSigner(t, keyA).Session(uid, issued)

	tests := map[string]struct {
		keys    [][]byte
		cookie  string
		now     time.Time
		wantErr string
	}{
		"a fresh session": {
			keys: [][]byte{keyA}, cookie: session.Value, now: issued,
		},
		"a second before it expires": {
			keys: [][]byte{keyA}, cookie: session.Value, now: issued.Add(auth.SessionLifetime - time.Second),
		},
		"signed by a key rotated out of first place": {
			keys: [][]byte{keyB, keyA}, cookie: session.Value, now: issued,
		},
		"no cookie": {
			keys: [][]byte{keyA}, now: issued,
			wantErr: "missing required parameters: session cookie",
		},
		"when it expires": {
			keys: [][]byte{keyA}, cookie: session.Value, now: issued.Add(auth.SessionLifetime),
			wantErr: "the session expired at 2026-10-10T20:00:00Z",
		},
		"signed by a key no longer configured": {
			keys: [][]byte{keyB}, cookie: session.Value, now: issued,
			wantErr: "not signed by a current key",
		},
		"another user's id with this user's signature": {
			keys: [][]byte{keyA}, now: issued,
			cookie:  swapUser(t, session.Value, uuid.Must(uuid.NewV7()), issued),
			wantErr: "not signed by a current key",
		},
		"signed for the nil user": {
			keys: [][]byte{keyA}, cookie: newSigner(t, keyA).Session(uuid.Nil, issued).Value, now: issued,
			wantErr: "names no user",
		},
		"garbage": {
			keys: [][]byte{keyA}, cookie: "garbage", now: issued,
			wantErr: "not signed by a current key",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/currencies", nil)
			if tc.cookie != "" {
				r.Header.Set("Cookie", auth.SessionCookie+"="+tc.cookie)
			}

			got, err := newSigner(t, tc.keys...).SessionUser(r, tc.now)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Equal(t, uuid.Nil, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, uid, got)
		})
	}
}

// swapUser replaces the user in a session value and keeps its signature,
// as someone who knows another uid but no key would. Only the uid differs:
// the expiry is the one issued signed.
func swapUser(t *testing.T, value string, uid uuid.UUID, issued time.Time) string {
	t.Helper()
	otherPayload, _, _ := strings.Cut(newSigner(t, keyB).Session(uid, issued).Value, ".")
	_, sig, _ := strings.Cut(value, ".")
	return otherPayload + "." + sig
}
