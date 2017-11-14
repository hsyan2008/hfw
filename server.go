package hfw

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/hsyan2008/go-logger/logger"
)

func startServe() {

	s := &http.Server{
		Addr: ":" + Config.Server.Port,
		// Handler:      controllers,
		ReadTimeout:  time.Duration(Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(Config.Server.WriteTimeout) * time.Second,
		// MaxHeaderBytes: 1 << 20,
	}
	//kill -USR2 pid 来重启
	err := gracehttp.Serve(s)
	// err:= s.ListenAndServe()

	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}

//支持https、grace
func startHTTPSServe(certFile, keyFile string) {

	var err error
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logger.Fatal("load cert file error:", err)
	}

	s := &http.Server{
		Addr: ":" + Config.Server.Port,
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

	//kill -USR2 pid 来重启
	err = gracehttp.Serve(s)

	if err != nil {
		logger.Fatal("ListenAndServeTls: ", err)
	}
}
