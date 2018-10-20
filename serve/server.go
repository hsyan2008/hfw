//kill -INT pid 终止
//kill -TERM pid 重启
package serve

import (
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/gracehttp"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

func Start(config configs.AllConfig) (err error) {

	addr := config.Server.Address
	readTimeout := config.Server.ReadTimeout * time.Second
	writeTimeout := config.Server.WriteTimeout * time.Second
	s := gracehttp.NewServer(addr, nil, readTimeout, writeTimeout)

	if common.IsExist(config.Server.HTTPSCertFile) && common.IsExist(config.Server.HTTPSKeyFile) {
		logger.Info("Listen on https", config.Server.Address)
		err = s.ListenAndServeTLS(config.Server.HTTPSCertFile, config.Server.HTTPSKeyFile)
	} else {
		logger.Info("Listen on http", config.Server.Address)
		err = s.ListenAndServe()
	}

	return
}
