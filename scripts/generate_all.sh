#!/usr/bin/env bash
# Generate CA self-signed certificate, as well as certificates for server, client1, and client2

set -euo pipefail

export FORCE=1

CERT_OUT=ca.pem KEY_OUT=ca_key.pem ./scripts/generate_ca_cert.sh

CERT_OUT=server.pem KEY_OUT=server_key.pem CA_CERT=ca.pem CA_KEY=ca_key.pem ./scripts/generate_signed_cert.sh spiffe://server
CERT_OUT=client1.pem KEY_OUT=client1_key.pem CA_CERT=ca.pem CA_KEY=ca_key.pem ./scripts/generate_signed_cert.sh spiffe://client1
CERT_OUT=client2.pem KEY_OUT=client2_key.pem CA_CERT=ca.pem CA_KEY=ca_key.pem ./scripts/generate_signed_cert.sh spiffe://client2

# Generate wrong certs for auth tests
CERT_OUT=ca_fake.pem KEY_OUT=ca_fake_key.pem ./scripts/generate_ca_cert.sh
CERT_OUT=client_fake.pem KEY_OUT=client_fake_key.pem CA_CERT=ca_fake.pem CA_KEY=ca_fake_key.pem ./scripts/generate_signed_cert.sh spiffe://client_fake
CERT_OUT=server_fake.pem KEY_OUT=server_fake_key.pem CA_CERT=ca_fake.pem CA_KEY=ca_fake_key.pem ./scripts/generate_signed_cert.sh spiffe://server
