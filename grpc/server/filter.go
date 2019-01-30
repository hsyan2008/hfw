package server

import (
	"context"

	logger "github.com/hsyan2008/go-logger"
	"google.golang.org/grpc"
)

func unaryFilter(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	logger.Debug("filter: ", info)

	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("panic: %v", r)
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
		if r := recover(); r != nil {
			logger.Errorf("panic: %v", r)
		}
	}()

	return handler(srv, ss)
}
