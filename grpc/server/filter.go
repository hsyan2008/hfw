package server

import (
	"context"
	"fmt"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
)

func unaryFilter(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	logger.Debug("filter: ", info)

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc server panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

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

	return handler(srv, ss)
}
