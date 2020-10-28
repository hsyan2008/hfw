package client

import (
	"context"

	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
)

func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {

	httpCtx := hfw.NewHTTPContextWithGrpcOutgoingCtx(ctx)
	httpCtx.AppendPrefix("Method:" + method)

	httpCtx.Debug("req:", req)
	defer func() {
		httpCtx.Debug("error:", err, "reply:", reply)
	}()

	defer func() {
		if e := recover(); e != nil {
			httpCtx.Fatal(e, string(common.GetStack()))
		}
	}()

	return invoker(httpCtx, method, req, reply, cc, opts...)
}

func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
	streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {

	httpCtx := hfw.NewHTTPContextWithGrpcOutgoingCtx(ctx)
	httpCtx.AppendPrefix("Method:" + method)

	defer func() {
		if e := recover(); e != nil {
			httpCtx.Fatal(e, string(common.GetStack()))
		}
	}()

	return streamer(httpCtx, desc, cc, method, opts...)
}
