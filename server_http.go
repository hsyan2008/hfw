//kill -INT pid 终止
//kill -TERM pid 重启
package hfw

import (
	"net"
	"sync"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/gracehttp"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery"
)

var listener net.Listener
var s *gracehttp.Server
var lock = new(sync.Mutex)

func GetHTTPListener() net.Listener {
	return listener
}

func GetHTTPAddr(config configs.HTTPServerConfig) (string, error) {

	err := newHTTPServer(config)
	if err != nil {
		return "", err
	}

	return listener.Addr().String(), nil
}

func newHTTPServer(config configs.HTTPServerConfig) (err error) {
	if s == nil || listener == nil {
		lock.Lock()
		defer lock.Unlock()
		if s == nil {
			addr, err := common.GetAddrForListen(config.Address)
			if err != nil {
				return err
			}
			readTimeout := config.ReadTimeout * time.Second
			writeTimeout := config.WriteTimeout * time.Second
			s = gracehttp.NewServer(addr, nil, readTimeout, writeTimeout)
		}
		if listener == nil {
			listener, err = s.InitListener()
			if err != nil {
				return
			}
		}
	}

	return
}

func StartHTTP(config configs.HTTPServerConfig) (err error) {

	err = newHTTPServer(config)
	if err != nil {
		return
	}

	//注册服务
	r, err := discovery.RegisterServer(config.ServerConfig, common.GetListendAddrForRegister(listener.Addr().String(), config.Address))
	if err != nil {
		return err
	}
	if r != nil {
		defer r.UnRegister()
	}

	if common.IsExist(config.CertFile) && common.IsExist(config.KeyFile) {
		logger.Mix("Listen on https:", listener.Addr().String())
		err = s.ListenAndServeTLS(config.CertFile, config.KeyFile)
	} else {
		logger.Mix("Listen on http:", listener.Addr().String())
		err = s.ListenAndServe()
	}

	return
}
