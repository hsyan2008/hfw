package common

import (
	"bytes"
	"errors"
	"net"
	"os/exec"
	"strconv"
)

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
	cmd := exec.Command("ip", "route")
	b, err := cmd.Output()
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
