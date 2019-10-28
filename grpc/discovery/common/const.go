package common

import (
	"github.com/hsyan2008/hfw/configs"
)

const (
	StaticResolver = "static"
	ConsulResolver = "consul"
	EtcdResolver   = "etcd"
)

var ResolverFuncMap = make(map[string]func(configs.GrpcConfig) (string, error))
var RegisterFuncMap = make(map[string]func([]string, int) Register)
