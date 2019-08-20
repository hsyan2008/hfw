package server

import (
	"context"
	"fmt"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
)

func unaryFilter(
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

func streamFilter(
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
