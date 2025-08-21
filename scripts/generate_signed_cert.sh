#!/usr/bin/env bash
# Generate a CA-signed TLS certificate using an Ed25519 key with 1 year (365 days) validity.
# Similar style to generate_ca_cert.sh but signs with an existing CA (./ca.pem and ./ca_key.pem).
#
# Usage:
#   ./scripts/generate_signed_cert.sh [SPIFFE_ID]
#     - If SPIFFE_ID is omitted, defaults to spiffe://server
#
# Environment overrides:
#   DAYS=365             # certificate validity in days
#   CERT_OUT=server.pem  # output certificate path
#   KEY_OUT=server_key.pem  # output private key path (PKCS#8 Ed25519)
#   CSR_OUT=server.csr   # output CSR path (temporary; removed unless KEEP_CSR=1)
#   CA_CERT=./ca.pem     # CA certificate
#   CA_KEY=./ca_key.pem  # CA private key
#   KEEP_CSR=0           # set to 1 to keep CSR
#   FORCE=0              # set to 1 to overwrite existing outputs
#   SPIFFE_ID=           # optional, e.g., spiffe://server
#
# Notes:
#   - Requires OpenSSL >= 1.1.1 (for Ed25519 and -addext support).
#   - This script encodes identity in the Subject Alternative Name as a SPIFFE URI (URI:spiffe://...). CN is not used for identity.

set -euo pipefail

# Inputs and defaults
SPIFFE_ID="${SPIFFE_ID:-${1:-spiffe://server}}"
DAYS="${DAYS:-365}"
CERT_OUT="${CERT_OUT:-server.pem}"
KEY_OUT="${KEY_OUT:-server_key.pem}"
CSR_OUT="${CSR_OUT:-server.csr}"
CA_CERT="${CA_CERT:-./ca.pem}"
CA_KEY="${CA_KEY:-./ca_key.pem}"

# Check for openssl
if ! command -v openssl >/dev/null 2>&1; then
  echo "Error: openssl is not installed or not in PATH" >&2
  exit 1
fi

# Check CA files
if [[ ! -f "$CA_CERT" ]]; then
  echo "Error: CA certificate not found at $CA_CERT" >&2
  exit 1
fi
if [[ ! -f "$CA_KEY" ]]; then
  echo "Error: CA key not found at $CA_KEY" >&2
  exit 1
fi

# Refuse to overwrite existing files unless FORCE=1
if [[ -e "$CERT_OUT" || -e "$KEY_OUT" ]]; then
  if [[ "${FORCE:-0}" != "1" ]]; then
    echo "Error: $CERT_OUT or $KEY_OUT already exists. Set FORCE=1 to overwrite." >&2
    exit 1
  fi
fi

# Generate Ed25519 key and CSR, then sign with CA
set -x
# 1) Private key
openssl genpkey -algorithm Ed25519 -out "$KEY_OUT"

# 2) CSR with SPIFFE SAN
openssl req \
  -new \
  -key "$KEY_OUT" \
  -out "$CSR_OUT" \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,URI:$SPIFFE_ID"

# 3) Create a temporary extensions file to ensure SAN and proper usage are present in the issued cert
EXTFILE="$(mktemp)"
cat >"$EXTFILE" <<EOF
[v3_req]
subjectAltName=DNS:localhost,IP:127.0.0.1,URI:$SPIFFE_ID
basicConstraints=CA:FALSE
keyUsage=digitalSignature
extendedKeyUsage=serverAuth,clientAuth
EOF

# 4) Sign CSR with CA
openssl x509 \
  -req \
  -in "$CSR_OUT" \
  -CA "$CA_CERT" \
  -CAkey "$CA_KEY" \
  -set_serial 0x$(openssl rand -hex 20) \
  -out "$CERT_OUT" \
  -days "$DAYS" \
  -extfile "$EXTFILE" \
  -extensions v3_req

# Cleanup temp extfile
rm -f "$EXTFILE"
set +x

# Optionally remove CSR
if [[ "${KEEP_CSR:-0}" != "1" ]]; then
  rm -f "$CSR_OUT" || true
fi

printf "\nGenerated:\n"
echo "  Key:   $KEY_OUT"
echo "  Cert:  $CERT_OUT"

# Print brief certificate info
openssl x509 -noout -subject -issuer -dates -in "$CERT_OUT" || true
