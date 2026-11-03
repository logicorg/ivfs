package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSigner_loginStateFrom(t *testing.T) {
	signer := &Signer{keys: [][]byte{keyA}}
	began := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	st := loginState{State: "state", Nonce: "nonce", Verifier: "verifier"}
	sealed, err := signer.loginStateCookie(st, began)
	require.NoError(t, err)

	tests := map[string]struct {
		value  string
		now    time.Time
		wantOK bool
	}{
		"right after it began":       {value: sealed.Value, now: began, wantOK: true},
		"a second before it expires": {value: sealed.Value, now: began.Add(loginLifetime - time.Second), wantOK: true},
		"when it expires":            {value: sealed.Value, now: began.Add(loginLifetime)},
		"a session cookie's value":   {value: signer.Session(uuid.Must(uuid.NewV7()), began).Value, now: began},
		"a signed value that is not a login state": {
			value: signer.seal(loginPurpose, []byte("not json")), now: began,
		},
		"no cookie": {now: began},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, CallbackPath, nil)
			if tc.value != "" {
				r.Header.Set("Cookie", loginCookie+"="+tc.value)
			}

			got, ok := signer.loginStateFrom(r, tc.now)
			require.Equal(t, tc.wantOK, ok)
			if !tc.wantOK {
				require.Equal(t, loginState{}, got)
				return
			}
			require.Equal(t, st.State, got.State)
			require.Equal(t, st.Nonce, got.Nonce)
			require.Equal(t, st.Verifier, got.Verifier)
		})
	}
}
