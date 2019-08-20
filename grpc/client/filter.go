package client

import (
	"context"
	"fmt"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"google.golang.org/grpc"
)

func unaryFilter(ctx context.Context, method string, req, reply interface{},
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

func streamFilter(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string,
	streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Grpc client panic: %#v", e)
			logger.Fatal(err, string(common.GetStack()))
		}
	}()

	return streamer(ctx, desc, cc, method, opts...)
}
