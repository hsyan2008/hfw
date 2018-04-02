//kill -INT pid 终止
//kill -TERM pid 重启
package serve

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hsyan2008/grace/gracehttp"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

func Start(config configs.AllConfig) (err error) {

	if common.IsExist(config.Server.HTTPSCertFile) && common.IsExist(config.Server.HTTPSKeyFile) {
		err = startHTTPS(config)
	} else {
		err = startHTTP(config)
	}

	return
}

func startHTTP(config configs.AllConfig) (err error) {

	s := &http.Server{
		Addr: config.Server.Address,
		// Handler:      controllers,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout+config.Server.ReadTimeout) * time.Second,
		// MaxHeaderBytes: 1 << 20,
	}

	err = gracehttp.Serve(s)
	// err = s.ListenAndServe()

	return
}

//支持https、grace
func startHTTPS(config configs.AllConfig) (err error) {
	certFile := config.Server.HTTPSCertFile
	keyFile := config.Server.HTTPSKeyFile
	phrase := config.Server.HTTPSPhrase

	var cert tls.Certificate

	if phrase != "" {
		//通过密码解密证书，暂时两个文件都是加密过的，如果有个文件没有加密，去掉pem.Decode和x509.DecryptPEMBlock即可
		certByte, err := ioutil.ReadFile(certFile)
		if err != nil {
			return err
		}
		certBlock, _ := pem.Decode(certByte)
		certDeBlock, err := x509.DecryptPEMBlock(certBlock, []byte(phrase))
		if err != nil {
			return err
		}

		keyByte, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}
		keyBlock, _ := pem.Decode(keyByte)
		keyDeBlock, err := x509.DecryptPEMBlock(keyBlock, []byte(phrase))
		if err != nil {
			return err
		}

		cert, err = tls.X509KeyPair(certDeBlock, keyDeBlock)
	} else {
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	}

	if err != nil {
		return
	}

	s := &http.Server{
		Addr: config.Server.Address,
		// Handler:      controllers,
		ReadTimeout:  time.Duration(config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Server.WriteTimeout+config.Server.ReadTimeout) * time.Second,
		// MaxHeaderBytes: 1 << 20,
		TLSConfig: &tls.Config{
			// NextProtos: []string{"http/1.1", "h2"}, //去掉1.1才支持h2
			NextProtos: []string{"h2"},
			Certificates: []tls.Certificate{
				cert,
			},
		},
	}

	err = gracehttp.Serve(s)

	return
}
