package client

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/auth"
	"github.com/hsyan2008/hfw/grpc/balancer/p2c"
	"github.com/hsyan2008/hfw/grpc/discovery"
	"github.com/hsyan2008/hfw/grpc/discovery/resolver"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type connInstance struct {
	//每个地址的连接实例
	c *grpc.ClientConn
	l *sync.Mutex
}

var connInstanceMap = make(map[string]*connInstance)
var lock = new(sync.Mutex)

func GetConn(ctx context.Context, c configs.GrpcConfig, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	return GetConnWithDefaultInterceptor(ctx, c, opts...)
}

func GetConnWithDefaultInterceptor(ctx context.Context, c configs.GrpcConfig, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	opts = append([]grpc.DialOption{
		grpc.WithUnaryInterceptor(UnaryClientInterceptor),
		grpc.WithStreamInterceptor(StreamClientInterceptor),
	}, opts...)
	return GetConnWithAuth(ctx, c, "", opts...)
}

func GetConnWithAuth(ctx context.Context, c configs.GrpcConfig, authValue string,
	opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	if len(c.ServerName) == 0 {
		return nil, errors.New("please specify grpc ServerName")
	}
	c, err = resolver.CompleteResolverScheme(c)
	if err != nil {
		return
	}
	var ok bool
	var p *connInstance
	lock.Lock()
	if p, ok = connInstanceMap[c.ResolverScheme]; !ok {
		p = &connInstance{
			l: new(sync.Mutex),
		}
		connInstanceMap[c.ResolverScheme] = p
		lock.Unlock()
	} else {
		lock.Unlock()
		if p.c != nil {
			return p.c, nil
		}
	}

	p.l.Lock()
	defer p.l.Unlock()

	if p.c != nil {
		return p.c, nil
	}

	scheme, err := discovery.GetAndRegisterResolver(c)
	if err != nil {
		return nil, err
	}
	address := fmt.Sprintf("%s:///%s", scheme, c.ServerName)

	conn, err = newClientConn(ctx, address, c, authValue, opts...)
	if err != nil {
		return
	}

	p.c = conn

	return
}

func removeClientConn(c configs.GrpcConfig, err error) {
	code := status.Code(err)
	if code != codes.Unavailable {
		return
	}
	c, err = resolver.CompleteResolverScheme(c)
	if err != nil {
		return
	}
	lock.Lock()
	if p, ok := connInstanceMap[c.ResolverScheme]; ok {
		p.c.Close()
		delete(connInstanceMap, c.ResolverScheme)
	}
	lock.Unlock()
}

func newClientConn(ctx context.Context, address string, c configs.GrpcConfig, authValue string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if strings.Contains(address, ":///") {
		switch c.BalancerName {
		case "round_robin":
			c.BalancerName = roundrobin.Name
		case "pick_first": //grpc默认
		default:
			// 现在默认是p2c
			c.BalancerName = p2c.Name
		}
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, c.BalancerName)))
	}
	if len(c.CertFile) > 0 && !filepath.IsAbs(c.CertFile) {
		c.CertFile = filepath.Join(common.GetAppPath(), c.CertFile)
		if !common.IsExist(c.CertFile) {
			return nil, fmt.Errorf("cert file: %s not exist", c.CertFile)
		}
	}
	if len(c.ServerName) > 0 && len(c.CertFile) > 0 && common.IsExist(c.CertFile) {
		if c.IsAuth {
			opts = append(opts, grpc.WithPerRPCCredentials(auth.NewAuthWithHTTPS(authValue)))
		}
		return NewClientConnWithSecurity(
			ctx,
			address,
			c.CertFile,
			c.ServerName,
			opts...)
	}

	if c.IsAuth {
		opts = append(opts, grpc.WithPerRPCCredentials(auth.NewAuth(authValue)))
	}

	return NewClientConn(ctx, address, opts...)
}
