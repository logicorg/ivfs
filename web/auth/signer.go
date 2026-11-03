// Package auth signs users in to the gateway (ADR 0013): it keeps who they
// are in a signed session cookie, and lets a request reach the API only as
// the user its session names.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// MinKeyLen is the shortest signing key a Signer accepts: HMAC-SHA256 gains
// nothing from more, and a shorter key is guessable.
const MinKeyLen = 32

// Signer signs the gateway's cookies with HMAC-SHA256. It signs with its
// first key and accepts any, so a key can be rotated in before the old one
// is dropped, without signing everyone out.
type Signer struct {
	keys [][]byte
}

// NewSigner returns a Signer over keys, the newest first.
func NewSigner(keys [][]byte) (*Signer, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("no signing keys: configure at least one of %d bytes or more", MinKeyLen)
	}
	for i, key := range keys {
		if len(key) < MinKeyLen {
			return nil, fmt.Errorf("signing key %d is %d bytes long: use %d bytes or more", i, len(key), MinKeyLen)
		}
	}
	return &Signer{keys: keys}, nil
}

// seal returns payload and its signature as a cookie value. The purpose is
// signed too, so a value sealed for one cookie never opens as another.
func (s *Signer) seal(purpose string, payload []byte) string {
	enc := base64.RawURLEncoding
	return enc.EncodeToString(payload) + "." + enc.EncodeToString(mac(s.keys[0], purpose, payload))
}

// open returns the payload of value if one of the keys sealed it for
// purpose.
func (s *Signer) open(purpose, value string) ([]byte, bool) {
	enc := base64.RawURLEncoding
	encodedPayload, encodedSig, ok := strings.Cut(value, ".")
	if !ok {
		return nil, false
	}
	payload, err := enc.DecodeString(encodedPayload)
	if err != nil {
		return nil, false
	}
	sig, err := enc.DecodeString(encodedSig)
	if err != nil {
		return nil, false
	}
	for _, key := range s.keys {
		if hmac.Equal(sig, mac(key, purpose, payload)) {
			return payload, true
		}
	}
	return nil, false
}

func mac(key []byte, purpose string, payload []byte) []byte {
	h := hmac.New(sha256.New, key)
	// The NUL keeps purpose "a" with payload "\x00b" apart from purpose
	// "a\x00" with payload "b"; no purpose holds one.
	h.Write([]byte(purpose))
	h.Write([]byte{0})
	h.Write(payload)
	return h.Sum(nil)
}
