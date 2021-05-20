package hfw

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery"
	"github.com/hsyan2008/hfw/grpc/server"
	"github.com/hsyan2008/hfw/prometheus"
	"github.com/hsyan2008/hfw/signal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

//如果是https+证书grpc，请配置好Server并使用NewGrpcServer+hfw.Run
//如果是grpc，请配置好Server和GrpcServer并使用NewGrpcServer+hfw.RunGrpc
//如果是grpc+http，请配置好Server和GrpcServer并使用NewGrpcServer+hfw.RunGrpc+hfw.Run

func NewGrpcServer(config configs.AllConfig) (s *grpc.Server, err error) {
	return server.NewServer(config.Server.ServerConfig, grpc.UnaryInterceptor(UnaryServerInterceptor),
		grpc.StreamInterceptor(StreamServerInterceptor))
}

var grpcListener net.Listener

func GetGrpcServerListener() net.Listener {
	return grpcListener
}

func RunGrpc(s *grpc.Server, config configs.GrpcServerConfig) (err error) {
	//监听信号
	signalContext := signal.GetSignalContext()
	go signalContext.Listen()

	signalContext.Mix("grpc server Starting ...")
	defer signalContext.Mix("grpc server Shutdowned!")

	//等待工作完成
	defer signalContext.Shutdowned()
	address, err := common.GetAddrForListen(config.Address)
	if err != nil {
		logger.Fatal("grpc StartServer:", err)
		return err
	}
	grpcListener, err = net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("grpc StartServer:", err)
		return err
	}

	//注册服务
	r, err := discovery.RegisterServer(config.ServerConfig, common.GetServerAddr(grpcListener.Addr().String(), config.Address))
	if err != nil {
		return err
	}
	if r != nil {
		defer r.UnRegister()
	}

	go func() {
		signalContext.WgAdd()
		defer signalContext.WgDone()
		select {
		case <-signalContext.Ctx.Done():
			signalContext.Info("grpc server stoping...")
			defer signalContext.Info("grpc server stoped")
			s.GracefulStop()
		}
	}()

	logger.Mix("Listen on grpc:", grpcListener.Addr().String())

	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s.Serve(grpcListener)
}

func UnaryServerInterceptor(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {

	httpCtx := NewHTTPContextWithGrpcIncomingCtx(ctx)
	defer httpCtx.Cancel()
	httpCtx.AppendPrefix("Path:" + info.FullMethod)

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

	startTime := time.Now()
	prometheus.RequestsTotal(info.FullMethod, "GRPC")
	defer func() {
		costTime := time.Since(startTime)
		httpCtx.Mixf("Method:%s CostTime:%s", "GRPC", costTime)
		prometheus.RequestsCosttime(info.FullMethod, "GRPC", costTime)
	}()

	onlineNum := atomic.AddUint32(&online, 1)
	httpCtx.Mixf("Online:%d", onlineNum)
	defer func() {
		atomic.AddUint32(&online, ^uint32(0))
		if e := recover(); e != nil {
			if e == ErrStopRun {
				return
			}
			err = errors.New("panic")
			httpCtx.Fatal(err, string(common.GetStack()))
		}
	}()

	err = checkConcurrence(onlineNum)
	if err != nil {
		return
	}

	return handler(httpCtx, req)
}

func StreamServerInterceptor(
	srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) (err error) {

	httpCtx := NewHTTPContextWithGrpcIncomingCtx(ss.Context())
	defer httpCtx.Cancel()
	httpCtx.AppendPrefix("Path:" + info.FullMethod)

	defer func() {
		if err != nil {
			httpCtx.Warn("Err:", err)
		}
	}()

	startTime := time.Now()
	prometheus.RequestsTotal(info.FullMethod, "Stream")
	defer func() {
		costTime := time.Since(startTime)
		httpCtx.Mixf("Method:%s CostTime:%s", "Stream", costTime)
		prometheus.RequestsCosttime(info.FullMethod, "Stream", costTime)
	}()

	onlineNum := atomic.AddUint32(&online, 1)
	httpCtx.Mixf("Online:%d", onlineNum)
	defer func() {
		atomic.AddUint32(&online, ^uint32(0))
		if e := recover(); e != nil {
			if e == ErrStopRun {
				return
			}
			err = errors.New("panic")
			httpCtx.Fatal(err, string(common.GetStack()))
		}
	}()

	err = checkConcurrence(onlineNum)
	if err != nil {
		return
	}

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
