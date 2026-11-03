.PHONY: sql-up
sql-up:
	docker compose -f docker-compose.yaml up -d

test:
	go test ./... -coverprofile=cover.out

coverage: test
	go tool cover -html=cover.out

# Run the web gateway (ADR 0007) on 127.0.0.1:8080, against Temporal on
# TEMPORAL_HOST_PORT or localhost:7233, signing users in through the
# compose's Authelia (ADR 0013), and pricing assets with the manual values
# and the compose's Yahoo provider (ADR 0014). The environment or the command
# line overrides each default: make web IVFS_PUBLIC_URL=http://localhost:5173
# serves make ui-dev. The session key is new on every run, so a restart signs everyone
# out, and Authelia's own session signs them straight back in.
.PHONY: web
web: export IVFS_PUBLIC_URL ?= http://localhost:8080
web: export IVFS_SESSION_KEYS ?= $(shell openssl rand -base64 32)
web: export OIDC_ISSUER ?= https://127.0.0.1:9091
web: export OIDC_CLIENT_ID ?= ivfs
web: export OIDC_CLIENT_SECRET ?= ivfs-dev-client-secret
web: export OIDC_GROUP ?= ivfs
web: export OIDC_CA_FILE ?= $(CURDIR)/compose/authelia/secrets/tls.crt
web: export OIDC_LOGOUT_URL ?= https://127.0.0.1:9091/logout
web: export IVFS_ASSET_VALUE_PROVIDERS ?= manual,yahoo=http://127.0.0.1:8081
web:
	go run ./web

# The web UI's pnpm, at the version web/ui/package.json pins, run through
# corepack from web/ui so corepack finds that pin.
PNPM = cd web/ui && COREPACK_ENABLE_DOWNLOAD_PROMPT=0 corepack pnpm

# Build the web UI into web/ui/dist/app, which the gateway embeds the next
# time it is built.
.PHONY: ui
ui:
	$(PNPM) install --frozen-lockfile
	$(PNPM) build

# Run the web UI's tests.
.PHONY: ui-test
ui-test:
	$(PNPM) install --frozen-lockfile
	$(PNPM) test

# Serve the web UI with hot reload on http://localhost:5173, forwarding /api
# and /auth to the gateway that make web IVFS_PUBLIC_URL=http://localhost:5173
# runs.
.PHONY: ui-dev
ui-dev:
	$(PNPM) install --frozen-lockfile
	$(PNPM) dev

# Regenerate the web API's OpenAPI document from its Go types, and the UI's
# TypeScript client from the document.
.PHONY: api
api:
	go run ./web -openapi web/api/openapi.json
	$(PNPM) install --frozen-lockfile
	$(PNPM) generate

# Fail if the committed OpenAPI document or TypeScript client is not what the
# Go types generate.
.PHONY: api-check
api-check: api
	git diff --exit-code -- web/api/openapi.json web/ui/src/api
