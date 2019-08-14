//kill -INT pid 终止
//kill -TERM pid 重启
package serve

import (
	"fmt"
	"net"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/gracehttp"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

var listener net.Listener

func GetAddr() (string, error) {
	if listener == nil {
		return "", fmt.Errorf("nil listener")
	}

	return listener.Addr().String(), nil
}

func Start(config configs.ServerConfig) (err error) {

	addr := config.Address
	readTimeout := config.ReadTimeout * time.Second
	writeTimeout := config.WriteTimeout * time.Second
	s := gracehttp.NewServer(addr, nil, readTimeout, writeTimeout)

	listener, err = s.InitListener()

	if common.IsExist(config.CertFile) && common.IsExist(config.KeyFile) {
		logger.Info("Listen on https", config.Address)
		err = s.ListenAndServeTLS(config.CertFile, config.KeyFile)
	} else {
		logger.Info("Listen on http", config.Address)
		err = s.ListenAndServe()
	}

	return
}
