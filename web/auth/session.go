package auth

import (
	"encoding/binary"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionCookie is the session's cookie. The __Host- prefix makes the
	// browser refuse it unless it is Secure, on /, and without a Domain, so
	// no other host or path can plant one.
	SessionCookie = "__Host-session"
	// SessionLifetime is how long a session lasts. It is never extended:
	// the cookie cannot be revoked, so its lifetime is the revocation delay.
	SessionLifetime = 12 * time.Hour

	sessionPurpose = "session"
)

// Session returns the cookie that signs uid in from now until
// SessionLifetime later.
func (s *Signer) Session(uid uuid.UUID, now time.Time) *http.Cookie {
	exp := now.Add(SessionLifetime)
	payload := binary.AppendVarint(uid[:len(uid):len(uid)], exp.Unix())
	return &http.Cookie{
		Name:     SessionCookie,
		Value:    s.seal(sessionPurpose, payload),
		Path:     "/",
		Expires:  exp,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearSession returns the cookie that signs the browser out.
func ClearSession() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookie,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
