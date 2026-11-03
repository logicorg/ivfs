package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

// TestLoadConfig checks the defaults the development compose relies on, that
// every variable overrides its default, and that the password is escaped.
func TestLoadConfig(t *testing.T) {
	tests := map[string]struct {
		env     map[string]string
		want    config
		wantErr string
	}{
		"the password alone": {
			env: map[string]string{ivfsAppPasswordVariable: "secret"},
			want: config{ //nolint: gosec // a test password
				registryDSN: "postgres://ivfs_app:secret@localhost:5432/ivfs",
				registryDB:  "ivfs",
				temporal:    client.Options{HostPort: "localhost:7233", Namespace: "default"},
			},
		},
		"every variable": {
			env: map[string]string{
				ivfsAppPasswordVariable: "p@ss/word",
				ivfsAppRoleVariable:     "app",
				registryDBVariable:      "registry",
				postgresHostVariable:    "ivfs-pg-rw:5432",
				disableSSLModeVariable:  "true",
				dsnQueryVariable:        "application_name=worker",
				healthAddrVariable:      ":8081",
				"TEMPORAL_HOST_PORT":    "temporal-frontend.temporal:7233",
				"TEMPORAL_NAMESPACE":    "ivfs",
			},
			want: config{ //nolint: gosec // a test password
				registryDSN: "postgres://app:p%40ss%2Fword@ivfs-pg-rw:5432/registry?sslmode=disable&application_name=worker",
				registryDB:  "registry",
				healthAddr:  ":8081",
				temporal:    client.Options{HostPort: "temporal-frontend.temporal:7233", Namespace: "ivfs"},
			},
		},
		"no password": {
			env:     map[string]string{ivfsAppRoleVariable: "app"},
			wantErr: "set IVFS_APP_PASSWORD to the password of the app role",
		},
		"an invalid Temporal address": {
			env:     map[string]string{ivfsAppPasswordVariable: "secret", "TEMPORAL_HOST_PORT": "temporal"},
			wantErr: "invalid TEMPORAL_HOST_PORT",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := loadConfig(func(key string) string { return tt.env[key] })
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
