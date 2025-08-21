package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultAddress = "localhost:50051"

// GRPCServer encapsulates TLS/mTLS configuration, gRPC server instance and listener.
type GRPCServer struct {
	lis net.Listener
	s   *grpc.Server
}

// NewGRPCServer constructs a TLS-enabled gRPC server that requires client certs (mTLS),
// registers the ProcessRunnerServiceServer, and prepares it to serve on the provided address.
func NewGRPCServer() (*GRPCServer, error) {
	addr := os.Getenv("PRN_ADDRESS")
	if strings.TrimSpace(addr) == "" {
		addr = defaultAddress
	}

	keyPEM := os.Getenv("PRN_TLS_KEY")
	certPEM := os.Getenv("PRN_TLS_CERT")
	caPEM := os.Getenv("PRN_CA_TLS_CERT")
	if keyPEM == "" || certPEM == "" || caPEM == "" {
		return nil, fmt.Errorf("missing TLS environment variables; require PRN_TLS_KEY, PRN_TLS_CERT, PRN_CA_TLS_CERT")
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to load server key pair: %w", err)
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM([]byte(caPEM)); !ok {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	creds := credentials.NewTLS(tlsConfig)
	s := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(injectSpiffeIdUnary), grpc.StreamInterceptor(injectSpiffeIdStream))

	server, err := NewProcessRunnerServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create service server: %w", err)
	}
	protov1.RegisterProcessRunnerServiceServer(s, server)

	return &GRPCServer{lis: lis, s: s}, nil
}

// Serve starts serving gRPC on the configured listener.
func (g *GRPCServer) Serve() error {
	return g.s.Serve(g.lis)
}

// Addr returns the network address the server is bound to.
func (g *GRPCServer) Addr() net.Addr { return g.lis.Addr() }

// Stop gracefully stops the gRPC server.
func (g *GRPCServer) Stop() { g.s.GracefulStop() }
