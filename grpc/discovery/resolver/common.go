package resolver

import (
	"fmt"
	"sync"

	"github.com/hsyan2008/hfw/configs"
	dc "github.com/hsyan2008/hfw/grpc/discovery/common"
)

var lock = new(sync.RWMutex)

func CompleteResolverScheme(c configs.GrpcConfig) (configs.GrpcConfig, error) {
	if c.ResolverScheme == "" {
		//static下，有可能服务名一样而地址不一样，做特殊处理
		if c.ResolverType == dc.StaticResolver {
			if len(c.Addresses) == 0 {
				return c, fmt.Errorf("please specify grpc %s Addresses", c.ServerName)
			}
			c.ResolverScheme = fmt.Sprintf("%s_%s_%s_%s", c.ResolverType, c.ServerName, c.Tag, c.Addresses[0])
		} else {
			if len(c.ResolverAddresses) == 0 {
				return c, fmt.Errorf("please specify grpc %s Addresses", c.ServerName)
			}
			c.ResolverScheme = fmt.Sprintf("%s_%s_%s_%s", c.ResolverType, c.ServerName, c.Tag, c.ResolverAddresses[0])
		}
	}

	return c, nil
}
