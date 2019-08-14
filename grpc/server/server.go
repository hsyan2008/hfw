//支持https+grpc的共享端口版，不需要ca证书
//获取http+grpc,两个服务必须独立端口
//Usage:
//如果是grpc+https
//s, err := server.NewServer(hfw.Config.Server, opt...)
//RegisterHelloServiceServer(s, &HelloServiceImpl{auth: auth.NewAuthWithHTTPS("value")})
//如果是http+grpc
//s, err := server.NewServer(hfw.Config.Server, opt...)
//RegisterHelloServiceServer(s, &HelloServiceImpl{auth: auth.NewAuth("value")})
//go server.StartServer(s, ":1234")
package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type ServerCreds struct {
	CaFile   string
	CertFile string
	KeyFile  string
}

var grpcServer *grpc.Server

func GetServer() *grpc.Server {
	return grpcServer
}

func NewServer(serverConfig configs.ServerConfig, opt ...grpc.ServerOption) (*grpc.Server, error) {
	if common.IsExist(serverConfig.CertFile) && common.IsExist(serverConfig.KeyFile) {
		logger.Debug("init grpc server with certFile and keyFile")
		t := &ServerCreds{
			CertFile: serverConfig.CertFile,
			KeyFile:  serverConfig.KeyFile,
		}

		creds, err := t.GetCredentials()
		if err != nil {
			return nil, err
		}

		opt = append(opt, grpc.Creds(creds))
	}

	if serverConfig.MaxRecvMsgSize > 0 {
		opt = append(opt, grpc.MaxRecvMsgSize(serverConfig.MaxRecvMsgSize))
	}

	if serverConfig.MaxSendMsgSize > 0 {
		opt = append(opt, grpc.MaxSendMsgSize(serverConfig.MaxSendMsgSize))
	}

	opt = append(opt, grpc.UnaryInterceptor(unaryFilter), grpc.StreamInterceptor(streamFilter))

	grpcServer = grpc.NewServer(
		opt...,
	)

	return grpcServer, nil
}

func StartServer(s *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("grpc StartServer:", err)
		return err
	}
	// Register reflection service on gRPC server.
	reflection.Register(s)
	return s.Serve(lis)
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
