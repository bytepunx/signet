package auth

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// metadataFromContext extracts gRPC incoming metadata from the context.
// Centralised here so both token.go and interceptors can use it without
// importing the grpc/metadata package in multiple files.
func metadataFromContext(ctx context.Context) (metadata.MD, bool) {
	return metadata.FromIncomingContext(ctx)
}
