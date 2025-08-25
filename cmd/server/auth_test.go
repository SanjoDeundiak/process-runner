package main

import (
	"context"
	"testing"
)

func TestContext_HasSpiffeId(t *testing.T) {
	ctx := context.Background()

	expected := "TEST"
	newCtx := injectSpiffeId(ctx, expected)
	actual := extractSpiffeIdFromTls(newCtx)

	if actual == nil {
		t.Fatalf("expected %s, got nil", expected)
	}

	if expected != *actual {
		t.Fatalf("expected %s, got %s", expected, *actual)
	}
}
