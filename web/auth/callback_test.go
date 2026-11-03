package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

func TestHandlers_callback(t *testing.T) {
	idp := newFakeProvider(t, false)
	uid := uuid.Must(uuid.NewV7())
	otherKey := newKey(t)

	tests := map[string]struct {
		group string
		// grant changes what the user approved at the provider.
		grant func(*grant)
		// answer changes the provider's answer to the callback.
		answer func(url.Values)
		// noState sends the callback without the sign-in's cookie, as
		// another browser would.
		noState   bool
		ensureErr error
		// wantEnsure is the subject the user is set up for; empty means
		// nobody is.
		wantEnsure string
		wantStatus int
		wantDetail string
	}{
		"a sign-in": {
			wantEnsure: "alice", wantStatus: http.StatusSeeOther,
		},
		"a sign-in by a member of the group": {
			group: "ivfs-users", grant: func(g *grant) { g.groups = []string{"admins", "ivfs-users"} },
			wantEnsure: "alice", wantStatus: http.StatusSeeOther,
		},
		"a sign-in by someone outside the group": {
			group: "ivfs-users", grant: func(g *grant) { g.groups = []string{"admins"} },
			wantStatus: http.StatusForbidden, wantDetail: "not in the group ivfs-users",
		},
		"a sign-in without a groups claim": {
			group:      "ivfs-users",
			wantStatus: http.StatusForbidden, wantDetail: "not in the group ivfs-users",
		},
		"an answer in another browser": {
			noState:    true,
			wantStatus: http.StatusBadRequest, wantDetail: "expired or began in another browser",
		},
		"an answer for another sign-in": {
			answer:     func(q url.Values) { q.Set("state", "forged") },
			wantStatus: http.StatusBadRequest, wantDetail: "for another sign-in",
		},
		"a forged error": {
			answer:     func(q url.Values) { q.Set("state", "forged"); q.Set("error", "access_denied") },
			wantStatus: http.StatusBadRequest, wantDetail: "for another sign-in",
		},
		"a refusal": {
			answer:     func(q url.Values) { q.Del("code"); q.Set("error", "access_denied") },
			wantStatus: http.StatusUnauthorized, wantDetail: "refused the sign-in: access_denied",
		},
		"a code from someone else's sign-in": {
			grant:      func(g *grant) { g.challenge = s256("someone else's verifier") },
			wantStatus: http.StatusUnauthorized, wantDetail: "did not accept the sign-in",
		},
		"an unknown code": {
			answer:     func(q url.Values) { q.Set("code", "unknown") },
			wantStatus: http.StatusUnauthorized, wantDetail: "did not accept the sign-in",
		},
		"an ID token signed by another key": {
			grant:      func(g *grant) { g.key = otherKey },
			wantStatus: http.StatusUnauthorized, wantDetail: "the ID token is not valid",
		},
		"an ID token for another client": {
			grant:      func(g *grant) { g.audience = "another-client" },
			wantStatus: http.StatusUnauthorized, wantDetail: "the ID token is not valid",
		},
		"an ID token for another sign-in": {
			grant:      func(g *grant) { g.nonce = "replayed" },
			wantStatus: http.StatusUnauthorized, wantDetail: "the ID token is for another sign-in",
		},
		"an account that cannot be set up yet": {
			ensureErr:  errors.New("provisioning worker down"),
			wantEnsure: "alice", wantStatus: http.StatusServiceUnavailable, wantDetail: "could not be set up yet",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var ensured []string
			s := newSignIn(t, idp, tc.group, func(_ context.Context, issuer, subject string) (uuid.UUID, error) {
				require.Equal(t, idp.issuer(), issuer)
				ensured = append(ensured, subject)
				return uid, tc.ensureErr
			})
			login := s.serve(http.MethodGet, auth.LoginPath)
			require.Equal(t, http.StatusFound, login.Code, login.Body.String())
			code, state := idp.authorize(t, login.Header().Get("Location"), tc.grant)
			answer := url.Values{"code": {code}, "state": {state}}
			if tc.answer != nil {
				tc.answer(answer)
			}
			var cookies []*http.Cookie
			if !tc.noState {
				cookies = append(cookies, cookie(login, "__Secure-login"))
			}

			w := s.serve(http.MethodGet, auth.CallbackPath+"?"+answer.Encode(), cookies...)

			require.Equal(t, tc.wantStatus, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tc.wantDetail)
			if tc.wantEnsure != "" {
				require.Equal(t, []string{tc.wantEnsure}, ensured)
			} else {
				require.Empty(t, ensured)
			}
			cleared := cookie(w, "__Secure-login")
			require.NotNil(t, cleared, "every answer ends the sign-in")
			require.Negative(t, cleared.MaxAge)

			session := cookie(w, auth.SessionCookie)
			if tc.wantStatus != http.StatusSeeOther {
				require.Nil(t, session)
				return
			}
			require.Equal(t, "/", w.Header().Get("Location"))
			require.NotNil(t, session)
			r, err := http.NewRequest(http.MethodGet, publicURL, nil)
			require.NoError(t, err)
			r.AddCookie(session)
			got, err := s.signer.SessionUser(r, time.Now())
			require.NoError(t, err)
			require.Equal(t, uid, got)
		})
	}
}

// TestHandlers_callback_Replay checks that a provider's answer finishes one
// sign-in only: its code is spent, so replaying it signs nobody in.
func TestHandlers_callback_Replay(t *testing.T) {
	idp := newFakeProvider(t, false)
	s := newSignIn(t, idp, "", func(context.Context, string, string) (uuid.UUID, error) {
		return uuid.Must(uuid.NewV7()), nil
	})
	login := s.serve(http.MethodGet, auth.LoginPath)
	code, state := idp.authorize(t, login.Header().Get("Location"), nil)
	target := auth.CallbackPath + "?" + url.Values{"code": {code}, "state": {state}}.Encode()

	require.Equal(t, http.StatusSeeOther, s.serve(http.MethodGet, target, cookie(login, "__Secure-login")).Code)
	replay := s.serve(http.MethodGet, target, cookie(login, "__Secure-login"))
	require.Equal(t, http.StatusUnauthorized, replay.Code, replay.Body.String())
	require.Nil(t, cookie(replay, auth.SessionCookie))
}
