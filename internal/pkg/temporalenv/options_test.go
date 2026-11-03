package temporalenv

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

// TestOptions checks the defaults the development compose relies on, that
// both variables override them, and that a host without a port is refused.
func TestOptions(t *testing.T) {
	tests := map[string]struct {
		env     map[string]string
		want    client.Options
		wantErr string
	}{
		"nothing set": {
			env:  map[string]string{},
			want: client.Options{HostPort: "localhost:7233", Namespace: "default"},
		},
		"both set": {
			env: map[string]string{
				HostPortVariable:  "temporal-frontend.temporal:7233",
				NamespaceVariable: "ivfs",
			},
			want: client.Options{HostPort: "temporal-frontend.temporal:7233", Namespace: "ivfs"},
		},
		"a host without a port": {
			env:     map[string]string{HostPortVariable: "temporal-frontend"},
			wantErr: `invalid TEMPORAL_HOST_PORT "temporal-frontend"`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := Options(func(key string) string { return tt.env[key] })
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
