package interceptor

import (
	"context"
	"fmt"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryServerInterceptor(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("trace_id:%s Grpc server: %s panic: %#v",
				GetTraceIDFromIncomingContext(ctx), info.FullMethod, e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return handler(ctx, req)
}

func StreamServerInterceptor(
	srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("trace_id:%s Grpc server: %s panic: %#v",
				GetTraceIDFromIncomingContext(ss.Context()), info.FullMethod, e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return handler(srv, ss)
}
func GetTraceIDFromIncomingContext(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	traceIDs := md.Get("trace_id")
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return ""
}

func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("trace_id:%s Grpc client: %s panic: %#v",
				GetTraceIDFromOutgoingContext(ctx), method, e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return invoker(ctx, method, req, reply, cc, opts...)
}

func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
	streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("trace_id:%s Grpc client: %s panic: %#v",
				GetTraceIDFromOutgoingContext(ctx), method, e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return streamer(ctx, desc, cc, method, opts...)
}

func GetTraceIDFromOutgoingContext(ctx context.Context) string {
	md, _ := metadata.FromOutgoingContext(ctx)
	traceIDs := md.Get("trace_id")
	if len(traceIDs) > 0 {
		return traceIDs[0]
	}
	return ""
}
