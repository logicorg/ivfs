package presettypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDefault checks that the default preset passes the validation of the
// workflow that applies it, and that each call returns a fresh copy.
func TestDefault(t *testing.T) {
	preset := Default()
	require.NoError(t, preset.Validate())

	preset.Categories[0].Children[0].Name = "changed"
	preset.Currencies[0].Code = "XXX"
	require.Equal(t, "Rent or mortgage", Default().Categories[0].Children[0].Name)
	require.Equal(t, "EUR", Default().Currencies[0].Code)
}
