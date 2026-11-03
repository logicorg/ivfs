package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

// SessionUser returns the user r's session cookie signs in at now.
func (s *Signer) SessionUser(r *http.Request, now time.Time) (uuid.UUID, error) {
	cookie, err := r.Cookie(SessionCookie)
	if errors.Is(err, http.ErrNoCookie) {
		return uuid.Nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"session cookie"}}
	}
	if err != nil {
		return uuid.Nil, internalerrors.ErrInvalidParameter{Description: "the session cookie is malformed"}
	}

	payload, ok := s.open(sessionPurpose, cookie.Value)
	if !ok || len(payload) <= len(uuid.UUID{}) {
		return uuid.Nil, internalerrors.ErrInvalidParameter{Description: "the session cookie is not signed by a current key"}
	}
	uid := uuid.UUID(payload[:len(uuid.UUID{})])
	unix, n := binary.Varint(payload[len(uid):])
	if n != len(payload)-len(uid) {
		// Signed, yet not what Session writes: a bug, not an attack.
		return uuid.Nil, internalerrors.ErrInvalidParameter{Description: "the session cookie holds no expiry; sign in again"}
	}
	exp := time.Unix(unix, 0)
	if uid == uuid.Nil {
		return uuid.Nil, internalerrors.ErrInvalidParameter{Description: "the session cookie names no user"}
	}
	if !now.Before(exp) {
		return uuid.Nil, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("the session expired at %s", exp.UTC().Format(time.RFC3339))}
	}
	return uid, nil
}
