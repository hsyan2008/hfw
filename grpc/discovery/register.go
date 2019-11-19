package discovery

import (
	"bytes"
	"errors"
	"net"
	"os/exec"
	"strconv"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	dc "github.com/hsyan2008/hfw/grpc/discovery/common"
	_ "github.com/hsyan2008/hfw/grpc/discovery/register"
)

func RegisterServer(cc configs.ServerConfig, address string) (r dc.Register, err error) {
	if cc.ResolverType == "" || len(cc.ResolverAddresses) == 0 || cc.ServerName == "" {
		logger.Mix("ResolverType or ResolverAddresses or ServerName is empty, so do not Registered")
		return nil, nil
	}
	if cc.UpdateInterval == 0 {
		cc.UpdateInterval = 10
	}
	host, port, err := getHostPort(cc, address)
	if err != nil {
		return nil, err
	}
	logger.Infof("Start register service: %s host: %s port: %d to %s", cc.ServerName, host, port, cc.ResolverType)
	if rf, ok := dc.RegisterFuncMap[cc.ResolverType]; ok {
		r = rf(cc.ResolverAddresses, int(cc.UpdateInterval)*2)
	} else {
		return nil, errors.New("unsupport ResolverType")
	}
	err = r.Register(dc.RegisterInfo{
		Host:           host,
		Port:           port,
		ServerName:     cc.ServerName,
		UpdateInterval: cc.UpdateInterval,
	})
	return r, err
}

func getHostPort(cc configs.ServerConfig, address string) (host string, port int, err error) {
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
	// if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() {
		return
	}

	//根据网卡名字查找ip
	if cc.Interface != "" {
		host = getIpByInterface(cc.Interface)
		if host != "" {
			return
		}
	}

	//根据hostname查找ip
	ips, _ := net.LookupIP(common.GetHostName())
	for _, ip := range ips {
		if !ip.IsLoopback() && ip.To4() != nil {
			host = ip.String()
			return
		}
	}

	//获取默认网卡的ip
	host = getIpByInterface(getDefaultInerfaceByRoute())
	if host != "" {
		return
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
