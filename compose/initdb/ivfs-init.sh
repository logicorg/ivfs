#!/bin/sh
# Creates the role that owns every ivfs database, and the registry it owns
# (ADR 0013). The admin superuser is needed for nothing else.
#
# PostgreSQL runs this once, on an empty data directory. It is safe to run
# again by hand on one that predates it:
#   docker compose exec postgres-ivfs /docker-entrypoint-initdb.d/ivfs-init.sh
set -eu

: "${IVFS_APP_PASSWORD:?set IVFS_APP_PASSWORD to the password of the ivfs_app role}"

# CREATE DATABASE cannot run in a DO block, so each statement is generated
# only when its object is missing and run with \gexec.
psql -v ON_ERROR_STOP=1 -v password="$IVFS_APP_PASSWORD" \
	--username "$POSTGRES_USER" --dbname postgres <<'SQL'
SELECT format('CREATE ROLE ivfs_app LOGIN CREATEDB PASSWORD %L', :'password')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ivfs_app')
\gexec
SELECT 'CREATE DATABASE ivfs OWNER ivfs_app'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'ivfs')
\gexec
SQL
