package institutiontypes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetrieveInstitutionParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  RetrieveInstitutionParams
		wantErr bool
	}{
		{
			name:    "valid id",
			params:  RetrieveInstitutionParams{ID: 1},
			wantErr: false,
		},
		{
			name:    "invalid id",
			params:  RetrieveInstitutionParams{ID: -1},
			wantErr: true,
		},
		{
			name:    "valid name",
			params:  RetrieveInstitutionParams{Name: "Bank of Test"},
			wantErr: false,
		},
		{
			name:    "invalid params",
			params:  RetrieveInstitutionParams{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, "expected no error for valid parameters: %v", tt.params)
			}
		})
	}
}

// Validation used to register a global cuego constraint on every call, which
// made each call several times slower than the previous one.
func TestRetrieveInstitutionParams_Validate_StaysFast(t *testing.T) {
	start := time.Now()
	for range 40 {
		params := RetrieveInstitutionParams{ID: 1}
		assert.NoError(t, params.Validate())
	}

	assert.Less(t, time.Since(start), 5*time.Second)
}
