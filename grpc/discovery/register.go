package discovery

import (
	"errors"
	"net"
	"strconv"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/grpc/discovery/register"
	"github.com/hsyan2008/hfw/grpc/discovery/resolver"
)

func RegisterServer(cc configs.ServerConfig, address string) (r register.Register, err error) {
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
	switch cc.ResolverType {
	case resolver.ConsulResolver:
		r = register.NewConsulRegister(cc.ResolverAddresses[0], int(cc.UpdateInterval)*2)
	case resolver.EtcdResolver:
		r = register.NewEtcdRegister(cc.ResolverAddresses, int(cc.UpdateInterval)*2)
	default:
		return nil, errors.New("unsupport ResolverType")
	}
	err = r.Register(register.RegisterInfo{
		Host:           host,
		Port:           port,
		ServiceName:    cc.ServerName,
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

	//如果是个合格的ip地址
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() && !ip.IsUnspecified() {
		return
	}

	//根据网卡名字查找ip
	if cc.Interface != "" {
		var iface *net.Interface
		iface, err = net.InterfaceByName(cc.Interface)
		if err != nil {
			return
		}
		var addrs []net.Addr
		addrs, err = iface.Addrs()
		if err != nil {
			return
		}

		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				host = ipnet.IP.String()
				return
			}
		}
	}

	//根据hostname查找ip
	var ips []net.IP
	ips, err = net.LookupIP(common.GetHostName())
	if err != nil {
		return
	}
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
