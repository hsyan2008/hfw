package resolver

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/common"
	"google.golang.org/grpc/resolver"
)

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

func (builder *staticBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
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
	addrStrs := r.addrsStore[r.target.Endpoint()]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*staticResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*staticResolver) Close()                                  {}

func init() {
	common.ResolverFuncMap[common.StaticResolver] = GenerateAndRegisterStaticResolver
}

func GenerateAndRegisterStaticResolver(cc configs.GrpcConfig) (schema string, err error) {
	if len(cc.Addresses) == 0 {
		return "", fmt.Errorf("GrpcConfig has nil Addresses")
	}
	cc, err = CompleteResolverScheme(cc)
	if err != nil {
		return
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
