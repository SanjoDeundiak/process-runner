# Process runner
This is a utility to run processes remotely.

## Design
Refer to [design.md](./DESIGN.md)

## Local testing

### Generate a secret key
```sh
openssl genpkey -algorithm Ed25519 -out server_key.pem
chmod 600 server_key.pem
```

### Generate a self-signed certificate
```sh
openssl req -new -x509 -key server_key.pem -out server.pem \
    -days 7 \
    -subj "/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
```

### Export secret key as an env var
```sh
export TLS_KEY="$(cat ./server_key.pem)"
```