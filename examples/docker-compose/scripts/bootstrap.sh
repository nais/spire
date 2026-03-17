#!/usr/bin/env sh
set -eu

BUNDLE_PATH="/opt/spire/bootstrap/bootstrap.crt"
TOKEN_PATH="/opt/spire/bootstrap/join.token"
AGENT_SPIFFE_ID="spiffe://example.nais.io/boot/strap/on"

echo "Waiting for SPIRE server API..."
for _ in $(seq 1 120); do
  if /opt/spire/bin/spire-server bundle show >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

/opt/spire/bin/spire-server bundle show > "$BUNDLE_PATH"

TOKEN="$(
  /opt/spire/bin/spire-server token generate -spiffeID "$AGENT_SPIFFE_ID" \
  | awk '/Token:/ {print $2}'
)"

if [ -z "$TOKEN" ]; then
  echo "Failed to generate join token" >&2
  exit 1
fi

printf '%s' "$TOKEN" > "$TOKEN_PATH"
echo "Bootstrap complete."
