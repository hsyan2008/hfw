//kill -INT pid 终止
//kill -TERM pid 重启
package serve

import (
	"time"

	"github.com/hsyan2008/gracehttp"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

func Start(config configs.AllConfig) (err error) {

	addr := config.Server.Address
	readTimeout := config.Server.ReadTimeout * time.Second
	writeTimeout := config.Server.WriteTimeout * time.Second
	certFile := config.Server.HTTPSCertFile
	keyFile := config.Server.HTTPSKeyFile

	if common.IsExist(certFile) && common.IsExist(keyFile) {
		err = gracehttp.NewServer(addr, nil, readTimeout, writeTimeout).ListenAndServeTLS(certFile, keyFile)
	} else {
		err = gracehttp.NewServer(addr, nil, readTimeout, writeTimeout).ListenAndServe()
	}

	return
}
