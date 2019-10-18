//kill -INT pid 终止
//kill -TERM pid 重启
package serve

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

func GetAddr(config configs.ServerConfig) (string, error) {

	err := newServer(config)
	if err != nil {
		return "", err
	}

	return listener.Addr().String(), nil
}

func newServer(config configs.ServerConfig) (err error) {
	if s == nil || listener == nil {
		lock.Lock()
		defer lock.Unlock()
		if s == nil {
			addr := config.Address
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

func Start(config configs.ServerConfig) (err error) {

	err = newServer(config)
	if err != nil {
		return
	}

	if common.IsExist(config.CertFile) && common.IsExist(config.KeyFile) {
		logger.Info("Listen on https:", listener.Addr().String())
		//注册服务
		r, err := discovery.RegisterServer(config, listener.Addr().String())
		if err != nil {
			return err
		}
		if r != nil {
			defer r.UnRegister()
		}

		err = s.ListenAndServeTLS(config.CertFile, config.KeyFile)
	} else {
		logger.Info("Listen on http:", listener.Addr().String())
		err = s.ListenAndServe()
	}

	return
}
