//kill -USR2 pid 重启
//kill -INT pid 终止
//kill -TERM pid 终止
package hfw

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/hsyan2008/go-logger/logger"
)

func startServe() {

	s := &http.Server{
		Addr: Config.Server.Port,
		// Handler:      controllers,
		ReadTimeout:  time.Duration(Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.Server.WriteTimeout) * time.Second,
		// MaxHeaderBytes: 1 << 20,
	}

	err := gracehttp.Serve(s)
	// err:= s.ListenAndServe()

	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}

//支持https、grace
func startHTTPSServe(certFile, keyFile, phrase string) {

	var err error
	var cert tls.Certificate

	if phrase != "" {
		//通过密码解密证书，暂时两个文件都是加密过的，如果有个文件没有加密，去掉pem.Decode和x509.DecryptPEMBlock即可
		certByte, err := ioutil.ReadFile(certFile)
		if err != nil {
			logger.Fatal("load cert file error:", err)
		}
		certBlock, _ := pem.Decode(certByte)
		certDeBlock, err := x509.DecryptPEMBlock(certBlock, []byte(phrase))
		if err != nil {
			logger.Fatal("Decrypt cert file error:", err)
		}

		keyByte, err := ioutil.ReadFile(keyFile)
		if err != nil {
			logger.Fatal("load key file error:", err)
		}
		keyBlock, _ := pem.Decode(keyByte)
		keyDeBlock, err := x509.DecryptPEMBlock(keyBlock, []byte(phrase))
		if err != nil {
			logger.Fatal("Decrypt key file error:", err)
		}

		cert, err = tls.X509KeyPair(certDeBlock, keyDeBlock)
	} else {
		cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	}

	if err != nil {
		logger.Fatal("X509KeyPair cert file error:", err)
	}

	s := &http.Server{
		Addr: Config.Server.Port,
		// Handler:      controllers,
		ReadTimeout:  time.Duration(Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.Server.WriteTimeout+Config.Server.ReadTimeout) * time.Second,
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

	if err != nil {
		logger.Fatal("ListenAndServeTls: ", err)
	}
}
