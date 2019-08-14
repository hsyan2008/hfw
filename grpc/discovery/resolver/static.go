package resolver

import (
	"github.com/hsyan2008/hfw2/configs"
	"google.golang.org/grpc/resolver"
)

const StaticResolver = "static"

type staticBuilder struct {
	//服务名字，一般取域名
	serviceName string
	addrs       []string
}

func NewStaticBuilder(serviceName string, addrs []string) *staticBuilder {
	return &staticBuilder{
		serviceName, addrs,
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
	return StaticResolver
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
	builder := NewStaticBuilder(cc.ServerName, cc.Addresses)
	resolver.Register(builder)
	schema = builder.Scheme()
	return
}
