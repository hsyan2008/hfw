package discovery

import (
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/common"
	"github.com/hsyan2008/hfw/grpc/discovery/resolver"
)

func GetAndRegisterResolver(cc configs.GrpcConfig) (scheme string, err error) {
	if r, ok := common.ResolverFuncMap[cc.ResolverType]; ok {
		return r(cc)
	} else {
		cc.ResolverType = common.StaticResolver
		return resolver.GenerateAndRegisterStaticResolver(cc)
	}
}
