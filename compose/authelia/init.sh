#!/bin/sh
# Generates the development Authelia's keys and secrets on its first start,
# into a directory git ignores: nothing secret is committed, and each checkout
# has its own. Each file is left alone once it exists, since a new storage key
# would lock Authelia out of its own database.
set -eu
cd /secrets

# Self-signed: Authelia only accepts an https URL for itself. The gateway
# trusts it through OIDC_CA_FILE, readable by the user running it, hence 644.
if [ ! -f tls.crt ]; then
  authelia crypto certificate rsa generate --directory . \
    --common-name 127.0.0.1 --sans 127.0.0.1,localhost \
    --file.private-key tls.key --file.certificate tls.crt
  chmod 644 tls.crt
fi

[ -f oidc.key ] || authelia crypto pair rsa generate --directory . \
  --file.private-key oidc.key --file.public-key oidc.pub

for name in jwt_secret session_secret storage_key hmac_secret; do
  [ -f "$name" ] || authelia crypto rand --length 64 --file "$name"
done
