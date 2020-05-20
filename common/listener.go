package common

import (
	"fmt"
	"net"
	"strings"
)

//监听127.0.0.1用于限定内部访问，完整的返回
//其他情况用于注册服务，只返回端口部分
func GetAddrForListen(addr string) (string, error) {
	if strings.HasPrefix(addr, "127.0.0.1:") {
		return addr, nil
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(":%s", port), nil
}

//取listend的端口，加上配置的host，用于注册
func GetListendAddrForRegister(listendAddr, addr string) string {
	_, port, _ := net.SplitHostPort(listendAddr)
	host, _, _ := net.SplitHostPort(addr)

	return net.JoinHostPort(host, port)
}
