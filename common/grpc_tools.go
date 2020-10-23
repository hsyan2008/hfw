package common

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func GetTraceIDFromIncomingContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	traceIDs := md.Get("trace_id")
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return uuid.New().String()
}

func GetTraceIDFromOutgoingContext(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	traceIDs := md.Get("trace_id")
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return uuid.New().String()
}
