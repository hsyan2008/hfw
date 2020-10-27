package client

import (
	"context"
	"time"

	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/signal"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	retry = 3
)

//如果有特殊需求，请自行修改
//如GetConn里的authValue，这里是空
//如GetConn里的grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(52428800))
//           和grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(52428800))
func Do(httpCtx *hfw.HTTPContext, c configs.GrpcConfig,
	call func(ctx context.Context, conn *grpc.ClientConn) (interface{}, error),
	timeout time.Duration,
) (resp interface{}, err error) {

	if httpCtx == nil {
		return nil, common.NewRespErr(500, "nil httpCtx")
	}

	var conn *grpc.ClientConn
	var retryNum int
	if len(c.Addresses) > 0 {
		retryNum = common.Min(retry, len(c.Addresses)+1)
	} else {
		//服务发现下，len是0
		retryNum = retry
	}

	ctx, cancel := context.WithTimeout(httpCtx.Ctx, timeout)
	defer cancel()

FOR:
	for i := 0; i < retryNum; i++ {
		select {
		case <-ctx.Done():
			return nil, common.NewRespErr(500, ctx.Err())
		default:
			newCtx := metadata.NewOutgoingContext(ctx, metadata.MD{
				common.GrpcTraceIDKey: []string{common.GetPureUUID(httpCtx.GetTraceID())},
			})
			if c.IsAuth {
				conn, err = GetConnWithAuth(signal.GetSignalContext().Ctx, c, "",
					grpc.WithUnaryInterceptor(UnaryClientInterceptor),
					grpc.WithStreamInterceptor(StreamClientInterceptor))
			} else {
				conn, err = GetConnWithDefaultInterceptor(signal.GetSignalContext().Ctx, c)
			}
			if err != nil {
				continue FOR
			}
			func() {
				defer func(t time.Time) {
					httpCtx.Infof("Call Grpc ServerName: %s CostTime: %s",
						c.ServerName, time.Since(t))
				}(time.Now())
				resp, err = call(newCtx, conn)
			}()
			if err == nil {
				return
			}
			httpCtx.Warnf("Call Grpc ServerName: %s %v", c.ServerName, err)
			// removeClientConn(c, err)
			if err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			if _, ok := err.(*common.RespErr); ok {
				return
			}
		}
	}

	return
}
