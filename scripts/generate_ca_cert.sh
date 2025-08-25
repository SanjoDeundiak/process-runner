#!/usr/bin/env bash
# Generate a self-signed certificate using an Ed25519 key with 1 year (365 days) validity.
# Usage:
#   ./generate_ca_cert.sh [COMMON_NAME]
# Environment overrides:
#   DAYS=365           # certificate validity in days
#   CERT_OUT=ca.pem
#   KEY_OUT=ca_key.pem
# Notes:
#   - Requires OpenSSL >= 1.1.1 (for ed25519 and -addext support).
#   - By default, CN is localhost and SAN includes DNS:localhost and IP:127.0.0.1

set -euo pipefail

# Defaults
DAYS="${DAYS:-365}"
CERT_OUT="${CERT_OUT:-ca.pem}"
KEY_OUT="${KEY_OUT:-ca_key.pem}"

# Check for openssl
if ! command -v openssl >/dev/null 2>&1; then
  echo "Error: openssl is not installed or not in PATH" >&2
  exit 1
fi

# Refuse to overwrite existing files unless FORCE=1
if [[ -e "$CERT_OUT" || -e "$KEY_OUT" ]]; then
  if [[ "${FORCE:-0}" != "1" ]]; then
    echo "Error: $CERT_OUT or $KEY_OUT already exists. Set FORCE=1 to overwrite." >&2
    exit 1
  fi
fi

# Generate self-signed certificate with Ed25519 key
# -nodes to avoid passphrase prompts (use with care)
set -x
openssl req \
  -x509 \
  -newkey ed25519 \
  -keyout "$KEY_OUT" \
  -out "$CERT_OUT" \
  -days "$DAYS" \
  -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
set +x

echo "\nGenerated:"
echo "  Key:  $KEY_OUT" 
echo "  Cert: $CERT_OUT"

# Print brief certificate info
openssl x509 -noout -subject -issuer -dates -in "$CERT_OUT" || true
