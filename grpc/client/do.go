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

	ctx, cancel := context.WithTimeout(httpCtx.Ctx, timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, metadata.MD{
		"trace_id": []string{httpCtx.GetTraceID()},
	})

	var conn *grpc.ClientConn

FOR:
	for i := 0; i < common.Min(retry, len(c.Addresses)+1); i++ {
		select {
		case <-ctx.Done():
			return nil, common.NewRespErr(500, ctx.Err())
		default:
			if c.IsAuth {
				conn, err = GetConnWithAuth(signal.GetSignalContext().Ctx, c, "")
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
				resp, err = call(ctx, conn)
			}()
			if err == nil {
				return
			}
			httpCtx.Warnf("Call Grpc ServerName: %s %v", c.ServerName, err)
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
