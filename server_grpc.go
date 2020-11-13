package hfw

import (
	"context"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	defer httpCtx.Cancel()
	httpCtx.AppendPrefix("Method:" + info.FullMethod)

	httpCtx.Debug("Req:", req)
	defer func() {
		if err == nil {
			httpCtx.Debug("Res:", resp)
		} else if status.Code(err) == codes.Canceled {
			httpCtx.Warn("Err:", err)
		} else {
			httpCtx.Warn("Req:", req, "Err:", err)
		}
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
	defer httpCtx.Cancel()
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
