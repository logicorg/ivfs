package auth

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	keyA = bytes.Repeat([]byte("a"), MinKeyLen)
	keyB = bytes.Repeat([]byte("b"), MinKeyLen)
)

func TestNewSigner(t *testing.T) {
	tests := map[string]struct {
		keys    [][]byte
		wantErr string
	}{
		"no keys":         {wantErr: "no signing keys"},
		"a short key":     {keys: [][]byte{keyA, keyB[:MinKeyLen-1]}, wantErr: "signing key 1 is 31 bytes long"},
		"one key":         {keys: [][]byte{keyA}},
		"a rotation pair": {keys: [][]byte{keyA, keyB}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			signer, err := NewSigner(tc.keys)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, signer)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestSigner_open checks that only a value sealed for the same purpose, by a
// key still configured, opens.
func TestSigner_open(t *testing.T) {
	payload := []byte("payload")
	sealed := (&Signer{keys: [][]byte{keyA}}).seal("session", payload)
	payloadPart, sigPart, _ := strings.Cut(sealed, ".")

	tests := map[string]struct {
		keys    [][]byte
		purpose string
		value   string
		wantOK  bool
	}{
		"the same key":                      {keys: [][]byte{keyA}, purpose: "session", value: sealed, wantOK: true},
		"a key rotated out of first place":  {keys: [][]byte{keyB, keyA}, purpose: "session", value: sealed, wantOK: true},
		"a key no longer configured":        {keys: [][]byte{keyB}, purpose: "session", value: sealed},
		"another purpose":                   {keys: [][]byte{keyA}, purpose: "login", value: sealed},
		"a tampered payload":                {keys: [][]byte{keyA}, purpose: "session", value: "cGF5bG9hZQ." + sigPart},
		"no signature":                      {keys: [][]byte{keyA}, purpose: "session", value: payloadPart},
		"a signature that is not base64url": {keys: [][]byte{keyA}, purpose: "session", value: payloadPart + ".!!"},
		"a payload that is not base64url":   {keys: [][]byte{keyA}, purpose: "session", value: "!!." + sigPart},
		"an empty value":                    {keys: [][]byte{keyA}, purpose: "session"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := (&Signer{keys: tc.keys}).open(tc.purpose, tc.value)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.Equal(t, payload, got)
			} else {
				require.Nil(t, got)
			}
		})
	}
}
