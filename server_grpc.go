package hfw

import (
	"context"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/server"
	"google.golang.org/grpc"
)

func StartGrpcServer(config configs.AllConfig) (s *grpc.Server, err error) {
	return server.NewServer(config.Server, grpc.UnaryInterceptor(UnaryServerInterceptor),
		grpc.StreamInterceptor(StreamServerInterceptor))
}

func UnaryServerInterceptor(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {

	httpCtx := NewHTTPContextWithGrpcIncomingCtx(ctx)
	httpCtx.AppendPrefix("Method:" + info.FullMethod)

	httpCtx.Debug("req:", req)
	defer func() {
		httpCtx.Debug("error:", err, "resp:", resp)
	}()

	defer func() {
		if e := recover(); e != nil {
			httpCtx.Fatal(e, string(common.GetStack()))
		}
	}()

	return handler(httpCtx, req)
}

func StreamServerInterceptor(
	srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) (err error) {

	httpCtx := NewHTTPContextWithGrpcIncomingCtx(ss.Context())
	httpCtx.AppendPrefix("Method:" + info.FullMethod)

	defer func() {
		if e := recover(); e != nil {
			httpCtx.Fatal(e, string(common.GetStack()))
		}
	}()

	return handler(srv, WarpServerStream(ss, httpCtx))
}

func WarpServerStream(ss grpc.ServerStream, httpCtx *HTTPContext) *GrpcServerStream {
	return &GrpcServerStream{
		ServerStream: ss,
		ctx:          httpCtx,
	}
}

type GrpcServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *GrpcServerStream) Context() context.Context {
	return w.ctx
}
