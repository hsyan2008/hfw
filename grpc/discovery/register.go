package discovery

import (
	"errors"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	dc "github.com/hsyan2008/hfw/grpc/discovery/common"
	_ "github.com/hsyan2008/hfw/grpc/discovery/register"
)

func RegisterServer(cc configs.ServerConfig, address string) (r dc.Register, err error) {
	if cc.ResolverType == "" || len(cc.ResolverAddresses) == 0 || cc.ServerName == "" {
		logger.Mix("ResolverType or ResolverAddresses or ServerName is empty, so do not Registered")
		return nil, nil
	}
	if cc.UpdateInterval == 0 {
		cc.UpdateInterval = 10
	}
	host, port, err := common.GetRegisterAddress(cc.Interface, address)
	if err != nil {
		return nil, err
	}
	logger.Infof("Start register service: %s host: %s port: %d to %s", cc.ServerName, host, port, cc.ResolverType)
	if rf, ok := dc.RegisterFuncMap[cc.ResolverType]; ok {
		r = rf(cc.ResolverAddresses, int(cc.UpdateInterval)*2)
	} else {
		return nil, errors.New("unsupport ResolverType")
	}
	err = r.Register(dc.RegisterInfo{
		Host:           host,
		Port:           port,
		ServerName:     cc.ServerName,
		UpdateInterval: cc.UpdateInterval,
	})
	return r, err
}
