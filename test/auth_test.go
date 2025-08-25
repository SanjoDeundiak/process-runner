package test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	protov1 "github.com/SanjoDeundiak/process-runner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestServerApp_ServerExpectsTls(t *testing.T) {
	addr := getAvailableAddress(t)

	stop := startServer(t, addr, false)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err == nil {
		t.Fatalf("expected insecure dial to fail, but it succeeded")
	}

}

func TestServerApp_ServerExpectsClientCert(t *testing.T) {
	addr := getAvailableAddress(t)

	stop := startServer(t, addr, false)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(newTLSCredsNoClient(t)), grpc.WithBlock())
	if err == nil {
		t.Fatalf("expected TLS without client cert to fail, but it succeeded")
	}

}

func TestServerApp_ServerExpectCorrectClientCa(t *testing.T) {
	addr := getAvailableAddress(t)

	stop := startServer(t, addr, false)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(newTLSCredsWithFakeClient(t)), grpc.WithBlock())
	if err == nil {
		t.Fatalf("expected TLS with client cert signed by unknown CA to fail, but it succeeded")
	}

}

func TestServerApp_ClientExpectsCorrectServerCa(t *testing.T) {
	addr := getAvailableAddress(t)

	stop := startServer(t, addr, true)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(newTLSCredsWithClient(t)), grpc.WithBlock())
	if err == nil {
		t.Fatalf("expected TLS with server cert signed by unknown CA to fail, but it succeeded")
	}
}

func TestServerApp_CorrectConfigSucceeds(t *testing.T) {
	addr := getAvailableAddress(t)

	stop := startServer(t, addr, false)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(newTLSCredsWithClient(t)), grpc.WithBlock())
	if err != nil {
		t.Fatalf("TLS dial with client cert failed: %v", err)
	}
	defer conn.Close()

	c := protov1.NewProcessRunnerServiceClient(conn)
	// Simple call: Status on a missing id should reach server and return NotFound
	_, err = c.Start(ctx, &protov1.StartRequest{Command: "sleep", Args: []string{"1"}})
	if err != nil {
		t.Fatalf("expected no error from server, got %v (err=%v)", status.Code(err), err)
	}
}

func caPool(t *testing.T) *x509.CertPool {
	t.Helper()

	root := repoRoot(t)
	caPEM := readFile(t, filepath.Join(root, "ca.pem"))
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(caPEM)) {
		t.Fatalf("failed to parse CA cert from env")
	}

	return pool
}

func newTLSCredsNoClient(t *testing.T) credentials.TransportCredentials {
	t.Helper()

	caPool := caPool(t)

	cfg := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.NoClientCert,
	}

	return credentials.NewTLS(cfg)
}

func readCert(t *testing.T, name string) tls.Certificate {
	t.Helper()

	root := repoRoot(t)
	clientPEM := readFile(t, filepath.Join(root, fmt.Sprintf("%s.pem", name)))
	clientKey := readFile(t, filepath.Join(root, fmt.Sprintf("%s_key.pem", name)))

	cert, err := tls.X509KeyPair([]byte(clientPEM), []byte(clientKey))
	if err != nil {
		t.Fatalf("failed to parse TLS cert/key from env: %v", err)
	}

	return cert
}

func newTLSCredsWithFakeClient(t *testing.T) credentials.TransportCredentials {
	t.Helper()

	caPool := caPool(t)

	cert := readCert(t, "client_fake")

	cfg := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(cfg)
}

func newTLSCredsWithClient(t *testing.T) credentials.TransportCredentials {
	t.Helper()

	caPool := caPool(t)

	cert := readCert(t, "client1")

	cfg := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(cfg)
}
