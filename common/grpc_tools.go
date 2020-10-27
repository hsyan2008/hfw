package common

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const GrpcTraceIDKey = "trace_id"
const GrpcHTTPTraceIDKey = "Trace_id"

func GetTraceIDFromIncomingContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	traceIDs := md.Get(GrpcTraceIDKey)
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return GetPureUUID()
}

func GetTraceIDFromOutgoingContext(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	traceIDs := md.Get(GrpcTraceIDKey)
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return GetPureUUID()
}
