package client

import "google.golang.org/grpc/resolver"

// resolver.Register(&ResolverBuilder{})

type ResolverBuilder struct {
	//协议名，一般取项目名字
	scheme string
	//服务名字，一般取域名
	serviceName string
	addrs       []string
}

func NewResolverBuilder(scheme, serviceName string, addrs []string) *ResolverBuilder {
	return &ResolverBuilder{
		scheme, serviceName, addrs,
	}
}

func (builder *ResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	r := &resolverImpl{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			builder.serviceName: builder.addrs,
		},
	}
	r.start()
	return r, nil
}
func (builder *ResolverBuilder) Scheme() string { return builder.scheme }

type resolverImpl struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *resolverImpl) start() {
	addrStrs := r.addrsStore[r.target.Endpoint]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*resolverImpl) ResolveNow(o resolver.ResolveNowOption) {}
func (*resolverImpl) Close()                                 {}
