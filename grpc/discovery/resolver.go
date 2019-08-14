package discovery

import (
	"fmt"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/grpc/discovery/resolver"
)

const StaticResolver = "static"
const ConsulResolver = "consul"
const EtcdResolver = "etcd"

func GetResolver(cc configs.GrpcConfig) (scheme string, err error) {

	if cc.ResolverType == "" && len(cc.Addresses) > 0 {
		cc.ResolverType = StaticResolver
	}

	switch cc.ResolverType {
	case StaticResolver:
		return resolver.GenerateAndRegisterStaticResolver(cc)
	case ConsulResolver:
		return resolver.GenerateAndRegisterConsulResolver(cc)
		// case EtcdResolver:
	}

	return "", fmt.Errorf("err resolver type")
}
