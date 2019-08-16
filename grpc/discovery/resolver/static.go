package resolver

import (
	"fmt"
	"strings"

	"github.com/hsyan2008/hfw/configs"
	"google.golang.org/grpc/resolver"
)

const StaticResolver = "static"

type staticBuilder struct {
	scheme string
	//服务名字，一般取域名
	serviceName string
	addrs       []string
}

func NewStaticBuilder(scheme, serviceName string, addrs []string) *staticBuilder {
	return &staticBuilder{
		scheme, serviceName, addrs,
	}
}

func (builder *staticBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	r := &staticResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			builder.serviceName: builder.addrs,
		},
	}
	r.start()
	return r, nil
}
func (builder *staticBuilder) Scheme() string {
	return builder.scheme
}

type staticResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *staticResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*staticResolver) ResolveNow(o resolver.ResolveNowOption) {}
func (*staticResolver) Close()                                 {}

func GenerateAndRegisterStaticResolver(cc configs.GrpcConfig) (schema string, err error) {
	if len(cc.Addresses) < 1 {
		return "", fmt.Errorf("GrpcConfig has nil Addresses")
	}
	if cc.ResolverScheme == "" {
		//每个服务调用地址不一样，所以必须区分
		cc.ResolverScheme = fmt.Sprintf("%s_%s", StaticResolver, strings.SplitN(cc.ServerName, ".", 2)[0])
	}
	lock.RLock()
	if resolver.Get(cc.ResolverScheme) != nil {
		lock.RUnlock()
		return cc.ResolverScheme, nil
	}
	lock.RUnlock()

	lock.Lock()
	defer lock.Unlock()

	if resolver.Get(cc.ResolverScheme) != nil {
		return cc.ResolverScheme, nil
	}
	builder := NewStaticBuilder(cc.ResolverScheme, cc.ServerName, cc.Addresses)
	resolver.Register(builder)
	schema = builder.Scheme()
	return
}
