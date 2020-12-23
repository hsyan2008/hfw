package hfw

import (
	"context"
	"net"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery"
	"github.com/hsyan2008/hfw/grpc/server"
	"github.com/hsyan2008/hfw/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

//如果是https+证书grpc，请配置好Server并使用NewGrpcServer+hfw.Run
//如果是无证书grpc，请配置好Server和GrpcServer并使用RunGrpc

func NewGrpcServer(config configs.AllConfig) (s *grpc.Server, err error) {
	return server.NewServer(config.Server.ServerConfig, grpc.UnaryInterceptor(UnaryServerInterceptor),
		grpc.StreamInterceptor(StreamServerInterceptor))
}

func RunGrpc(s *grpc.Server, config configs.GrpcServerConfig) error {
	lis, err := net.Listen("tcp", config.Address)
	if err != nil {
		logger.Fatal("grpc StartServer:", err)
		return err
	}

	//注册服务
	r, err := discovery.RegisterServer(config.ServerConfig, common.GetListendAddrForRegister(lis.Addr().String(), config.Address))
	if err != nil {
		return err
	}
	if r != nil {
		defer r.UnRegister()
	}

	go func() {
		signal.GetSignalContext().WgAdd()
		defer signal.GetSignalContext().WgDone()
		select {
		case <-signal.GetSignalContext().Ctx.Done():
			signal.GetSignalContext().Info("grpc server stoping...")
			defer signal.GetSignalContext().Info("grpc server stoped")
			s.GracefulStop()
		}
	}()

	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s.Serve(lis)
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
			httpCtx.Warn("Req:", req, "Method:", info.FullMethod, "Err:", err)
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
