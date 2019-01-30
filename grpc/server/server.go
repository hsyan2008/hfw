//只支持https+grpc的共享端口版，不需要ca证书
//Usage:
//s, err := server.InitGrpcServer(hfw.Config.Server)
//RegisterHelloServiceServer(s, &HelloServiceImpl{auth: &X{Value: "ab", Key:"x"}})
package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type ServerCreds struct {
	CaFile   string
	CertFile string
	KeyFile  string
}

var grpcServer *grpc.Server

func GetGrpcServer() *grpc.Server {
	return grpcServer
}

func InitGrpcServer(serverConfig configs.ServerConfig, opt ...grpc.ServerOption) (*grpc.Server, error) {
	if common.IsExist(serverConfig.HTTPSCertFile) && common.IsExist(serverConfig.HTTPSKeyFile) {
		logger.Debug("init grpc server with certFile and keyFile")
		t := &ServerCreds{
			CertFile: serverConfig.HTTPSCertFile,
			KeyFile:  serverConfig.HTTPSKeyFile,
		}

		creds, err := t.GetCredentials()
		if err != nil {
			return nil, err
		}

		opt = append(opt, grpc.Creds(creds))
	}

	opt = append(opt, grpc.UnaryInterceptor(unaryFilter), grpc.StreamInterceptor(streamFilter))

	grpcServer = grpc.NewServer(
		opt...,
	)

	return grpcServer, nil
}

func (t *ServerCreds) GetCredentials() (credentials.TransportCredentials, error) {
	if common.IsExist(t.CaFile) {
		return t.GetCredentialsByCA()
	}

	return t.GetTLSCredentials()
}

func (t *ServerCreds) GetCredentialsByCA() (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(t.CaFile)
	if err != nil {
		return nil, err
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("certPool.AppendCertsFromPEM err")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})

	return c, err
}

func (t *ServerCreds) GetTLSCredentials() (credentials.TransportCredentials, error) {
	c, err := credentials.NewServerTLSFromFile(t.CertFile, t.KeyFile)
	if err != nil {
		return nil, err
	}

	return c, err
}
