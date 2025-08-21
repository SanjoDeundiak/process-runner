package main

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type spiffeIdContextKey struct{}

func extractSpiffeIdFromContext(ctx context.Context) *string {
	if v := ctx.Value(spiffeIdContextKey{}); v != nil {
		if spiffeId, ok := v.(string); ok {
			return &spiffeId
		}
	}
	return nil
}

func extractSpiffeIdFromTls(ctx context.Context) *string {
	// First, check if it was already injected into context.
	if v := extractSpiffeIdFromContext(ctx); v != nil {
		return v
	}

	p, ok := peer.FromContext(ctx)
	if !ok || p == nil {
		return nil
	}

	ti, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil
	}

	state := ti.State

	if len(state.PeerCertificates) == 0 || state.PeerCertificates[0] == nil {
		return nil
	}

	leaf := state.PeerCertificates[0]

	// Find the first SPIFFE URI SAN
	for _, uri := range leaf.URIs {
		if uri == nil {
			continue
		}
		if uri.Scheme == "spiffe" {
			// Return trust domain (host) part as our ID, e.g., spiffe://client1 -> "client1"
			return &uri.Host
		}
	}

	return nil
}

func injectSpiffeId(ctx context.Context, spiffeId string) context.Context {
	ctx = context.WithValue(ctx, spiffeIdContextKey{}, spiffeId)

	return ctx
}

// injectSpiffeIdUnary extracts the SPIFFE ID from the TLS certificate.
func injectSpiffeIdUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	spiffeId := extractSpiffeIdFromTls(ctx)

	if spiffeId == nil {
		return nil, status.Error(codes.Unauthenticated, "client must have SPIFFE ID")
	}

	ctx = injectSpiffeId(ctx, *spiffeId)

	return handler(ctx, req)
}

type streamWithCtx struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *streamWithCtx) Context() context.Context { return s.ctx }

// injectSpiffeIdStream extracts the SPIFFE ID from the TLS certificate.
func injectSpiffeIdStream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()

	spiffeId := extractSpiffeIdFromTls(ctx)

	if spiffeId == nil {
		return status.Error(codes.Unauthenticated, "client must have SPIFFE ID")
	}

	ctx = injectSpiffeId(ctx, *spiffeId)

	return handler(srv, &streamWithCtx{ServerStream: ss, ctx: ctx})
}

// TODO: Could have been implemented as a middleware
func (s *ProcessRunnerServiceServer) checkOwnership(ctx context.Context, processIdentifier string) error {
	spiffeId := extractSpiffeIdFromContext(ctx)

	if spiffeId == nil {
		return status.Error(codes.Unauthenticated, "client must have SPIFFE ID")
	}

	s.mu.RLock()
	realOwner := s.ownersMap[processIdentifier]
	s.mu.RUnlock()

	if realOwner != *spiffeId {
		return status.Error(codes.PermissionDenied, "Only original owner can access the resource")
	}

	return nil
}
