//kill -INT pid 终止
//kill -TERM pid 重启
package serve

import (
	"fmt"
	"net"
	"strings"
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
			addr, err := getListenAddr(config.Address)
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

//可能监听127.0.0.1用于限定内部访问，完整的返回
//可能其他情况用于注册服务，只返回端口部分
func getListenAddr(addr string) (string, error) {
	if strings.HasPrefix(addr, "127.0.0.1:") {
		return addr, nil
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(":%s", port), nil
}

func Start(config configs.ServerConfig) (err error) {

	err = newServer(config)
	if err != nil {
		return
	}

	if common.IsExist(config.CertFile) && common.IsExist(config.KeyFile) {
		logger.Mix("Listen on https:", listener.Addr().String())
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
		logger.Mix("Listen on http:", listener.Addr().String())
		err = s.ListenAndServe()
	}

	return
}
