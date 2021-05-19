package common

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
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

//取listend的端口，加上配置的host(可能为空)
func GetServerAddr(listendAddr, addr string) string {
	_, port, _ := net.SplitHostPort(listendAddr)
	host, _, _ := net.SplitHostPort(addr)

	return net.JoinHostPort(host, port)
}

//如果address是完整的ip:port，则返回
//否则取本机的ip地址，加上address的端口
func GetRegisterAddress(interfaceName, address string) (host string, port int, err error) {
	var p string
	host, p, err = net.SplitHostPort(address)
	if err != nil {
		return
	}
	port, err = strconv.Atoi(p)
	if err != nil {
		return
	}

	//如果是个合格的ip地址
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		return
	}

	host, err = GetHostIP(interfaceName)

	return
}

//获取本机ip，可以指定网卡名字
func GetHostIP(interfaceName string) (host string, err error) {
	//根据网卡名字查找ip
	if interfaceName != "" {
		host = getIpByInterface(interfaceName)
		if host != "" {
			return
		}
	}

	//获取默认网卡的ip
	host = getIpByInterface(getDefaultInerfaceByRoute())
	if host != "" {
		return
	}

	//根据hostname查找ip
	ips, _ := net.LookupIP(GetHostName())
	for _, ip := range ips {
		if !ip.IsLoopback() && ip.To4() != nil {
			host = ip.String()
			return
		}
	}

	//没有host
	err = errors.New("not found host for register")
	return
}

//根据ip route获取默认的网卡名称
func getDefaultInerfaceByRoute() string {
	cmd := exec.Command("sh", "-c", "ip route show | grep default")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	fields := bytes.Fields(b)
	if len(fields) < 5 {
		return ""
	}

	return string(fields[4])
}

func getIpByInterface(ifName string) (host string) {

	if ifName == "" {
		return
	}

	iface, err := net.InterfaceByName(ifName)
	if err != nil {
		return
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			host = ipnet.IP.String()
			return
		}
	}

	return
}
