package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/grpc/auth"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

type connInstance struct {
	//每个地址的连接实例
	c *grpc.ClientConn
	l *sync.Mutex
}

var connInstanceMap = make(map[string]*connInstance)
var lock = new(sync.Mutex)

func GetConn(ctx context.Context, c configs.GrpcConfig, opt ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return GetConnWithAuth(ctx, c, "", opt...)
}

func GetConnWithAuth(ctx context.Context, c configs.GrpcConfig, authValue string, opt ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	if len(c.ServerName) == 0 {
		return nil, errors.New("please specify grpc ServerName")
	}
	var ok bool
	var p *connInstance
	// 在main.go里自己手动init，放这里会出现map的panic
	// scheme, err := discovery.GetResolver(c)
	// if err != nil {
	// 	return nil, err
	// }
	address := fmt.Sprintf("%s:///%s", c.ResolverType, c.ServerName)
	lock.Lock()
	if p, ok = connInstanceMap[c.ServerName]; !ok {
		p = &connInstance{
			l: new(sync.Mutex),
		}
		connInstanceMap[c.ServerName] = p
		lock.Unlock()
	} else {
		lock.Unlock()
		if p.c != nil {
			return p.c, nil
		}
	}

	p.l.Lock()
	defer p.l.Unlock()

	conn, err = newClientConn(ctx, address, c, authValue, opt...)
	if err != nil {
		return
	}

	p.c = conn

	return
}

func newClientConn(ctx context.Context, address string, c configs.GrpcConfig, authValue string, opt ...grpc.DialOption) (*grpc.ClientConn, error) {
	logger.Warn(address, c)
	if strings.Contains(address, ":///") {
		// opt = append(opt, grpc.WithBalancerName("round_robin")) //grpc里默认是grpc.WithBalancerName("pick_first")
		if c.BalancerName == "" {
			// 现在默认是round_robin
			c.BalancerName = roundrobin.Name
		}
		opt = append(opt, grpc.WithBalancerName(c.BalancerName))
	}
	if len(c.ServerName) > 0 && common.IsExist(c.CertFile) {
		if c.IsAuth {
			opt = append(opt, grpc.WithPerRPCCredentials(auth.NewAuthWithHTTPS(authValue)))
		}
		return NewClientConnWithSecurity(
			ctx,
			address,
			c.CertFile,
			c.ServerName,
			opt...)
	}

	if c.IsAuth {
		opt = append(opt, grpc.WithPerRPCCredentials(auth.NewAuth(authValue)))
	}

	return NewClientConn(ctx, address, opt...)
}
