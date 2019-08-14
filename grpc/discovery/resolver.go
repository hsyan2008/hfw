package discovery

import (
	"fmt"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/grpc/discovery/resolver"
)

func GetResolver(cc configs.GrpcConfig) (scheme string, err error) {

	if cc.ResolverType == "" && len(cc.Addresses) > 0 {
		cc.ResolverType = resolver.StaticResolver
	}

	switch cc.ResolverType {
	case resolver.StaticResolver:
		return resolver.GenerateAndRegisterStaticResolver(cc)
	case resolver.ConsulResolver:
		return resolver.GenerateAndRegisterConsulResolver(cc)
		// case EtcdResolver:
	}

	return "", fmt.Errorf("err resolver type")
}
