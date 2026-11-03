package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/logicorg/ivfs/internal/pkg/temporalenv"
	"go.temporal.io/sdk/client"
)

// The environment the worker is configured from.
const (
	ivfsAppPasswordVariable = "IVFS_APP_PASSWORD" //nolint: gosec
	ivfsAppRoleVariable     = "IVFS_APP_ROLE"
	registryDBVariable      = "IVFS_REGISTRY_DATABASE"
	postgresHostVariable    = "POSTGRES_HOST"
	dsnQueryVariable        = "DSN_QUERY"
	disableSSLModeVariable  = "DISABLE_SSL_MODE"
	// Unset, the worker serves no probes, as in the development compose.
	healthAddrVariable = "IVFS_HEALTH_ADDR"
)

// The defaults are what compose/initdb/ivfs-init.sh creates: the role that
// owns the registry and, since it creates them, every user's database.
const (
	defaultPostgresHost = "localhost:5432"
	defaultAppRole      = "ivfs_app"
	defaultRegistryDB   = "ivfs"
	disableSSLMode      = "sslmode=disable"
)

// config is what the worker needs to reach PostgreSQL and Temporal.
type config struct {
	// registryDSN connects to the registry; the users' databases are on the
	// same server, as the same role.
	registryDSN string
	registryDB  string
	healthAddr  string
	temporal    client.Options
}

// loadConfig reads the configuration through getenv.
func loadConfig(getenv func(string) string) (config, error) {
	password := getenv(ivfsAppPasswordVariable)
	if password == "" {
		return config{}, fmt.Errorf("set %s to the password of the %s role", ivfsAppPasswordVariable,
			withDefault(getenv(ivfsAppRoleVariable), defaultAppRole))
	}

	var query []string
	if getenv(disableSSLModeVariable) == "true" {
		query = append(query, disableSSLMode)
	}
	if extra := getenv(dsnQueryVariable); extra != "" {
		query = append(query, extra)
	}

	registryDB := withDefault(getenv(registryDBVariable), defaultRegistryDB)
	// url.URL escapes the password, which a generated one may need.
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(withDefault(getenv(ivfsAppRoleVariable), defaultAppRole), password),
		Host:     withDefault(getenv(postgresHostVariable), defaultPostgresHost),
		Path:     "/" + registryDB,
		RawQuery: strings.Join(query, "&"),
	}

	temporal, err := temporalenv.Options(getenv)
	if err != nil {
		return config{}, err
	}
	return config{
		registryDSN: dsn.String(),
		registryDB:  registryDB,
		healthAddr:  getenv(healthAddrVariable),
		temporal:    temporal,
	}, nil
}

func withDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
