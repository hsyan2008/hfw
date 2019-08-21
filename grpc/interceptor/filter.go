package interceptor

import (
	"context"
	"fmt"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc server panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	if logger.Level() == logger.DEBUG {
		startTime := time.Now()
		resp, err = handler(ctx, req)
		logger.Debugf("Grpc server: %s CostTime: %s", info.FullMethod, time.Since(startTime))
		return resp, err
	}

	return handler(ctx, req)
}

func StreamServerInterceptor(
	srv interface{}, ss grpc.ServerStream,
	info *grpc.StreamServerInfo, handler grpc.StreamHandler,
) (err error) {
	logger.Debug("filter: ", info)

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc server panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	if logger.Level() == logger.DEBUG {
		startTime := time.Now()
		err = handler(srv, ss)
		logger.Debugf("Grpc server: %s CostTime: %s", info.FullMethod, time.Since(startTime))
		return err
	}

	return handler(srv, ss)
}

func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc client panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	if logger.Level() == logger.DEBUG {
		startTime := time.Now()
		err = invoker(ctx, method, req, reply, cc, opts...)
		logger.Debugf("Grpc client: %s CostTime: %s", method, time.Since(startTime))
		return err
	}

	return invoker(ctx, method, req, reply, cc, opts...)
}

func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
	streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc client panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return streamer(ctx, desc, cc, method, opts...)
}
