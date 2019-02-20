package server

import (
	"context"
	"runtime"

	logger "github.com/hsyan2008/go-logger"
	"google.golang.org/grpc"
)

func unaryFilter(
	ctx context.Context, req interface{},
	info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	logger.Debug("filter: ", info)

	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<20)
			num := runtime.Stack(buf, false)
			logger.Fatal(err, num, string(buf))
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
		if err := recover(); err != nil {
			buf := make([]byte, 1<<20)
			num := runtime.Stack(buf, false)
			logger.Fatal(err, num, string(buf))
		}
	}()

	return handler(srv, ss)
}
