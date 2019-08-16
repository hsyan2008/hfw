package discovery

import (
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/resolver"
)

func GetAndRegisterResolver(cc configs.GrpcConfig) (scheme string, err error) {
	switch cc.ResolverType {
	case resolver.StaticResolver:
		return resolver.GenerateAndRegisterStaticResolver(cc)
	case resolver.ConsulResolver:
		return resolver.GenerateAndRegisterConsulResolver(cc)
		// case EtcdResolver:
	default:
		cc.ResolverType = resolver.StaticResolver
		return resolver.GenerateAndRegisterStaticResolver(cc)
	}

	// return "", fmt.Errorf("err resolver type")
}
