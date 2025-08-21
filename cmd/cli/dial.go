package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const defaultAddress = "localhost:50051"

func dial(ctx context.Context) (*grpc.ClientConn, error) {
	addr := os.Getenv("PRN_ADDRESS")
	if strings.TrimSpace(addr) == "" {
		addr = defaultAddress
	}

	keyPEM := os.Getenv("PRN_TLS_KEY")
	certPEM := os.Getenv("PRN_TLS_CERT")
	caPEM := os.Getenv("PRN_CA_TLS_CERT")
	if strings.TrimSpace(keyPEM) == "" || strings.TrimSpace(certPEM) == "" || strings.TrimSpace(caPEM) == "" {
		return nil, fmt.Errorf("missing TLS environment variables; require PRN_TLS_KEY, PRN_TLS_CERT, PRN_CA_TLS_CERT")
	}

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TLS cert/key from env: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(caPEM)) {
		return nil, fmt.Errorf("failed to parse CA cert from env")
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	}
	creds := credentials.NewTLS(cfg)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func grpcCode(err error) codes.Code {
	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown
	}
	return st.Code()
}
