package hfw

import (
	"flag"
	"os"

	"github.com/hsyan2008/hfw/common"
)

var (
	//VERSION 版本
	VERSION string
	//ENVIRONMENT 环境
	ENVIRONMENT string

	PID         = os.Getpid()
	HOSTNAME, _ = os.Hostname()
)

func GetVersion() string {
	return VERSION
}

func GetEnv() string {
	return ENVIRONMENT
}

func parseFlag() {
	ENVIRONMENT = os.Getenv("ENVIRONMENT")
	if len(ENVIRONMENT) == 0 {
		flag.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	}

	VERSION = os.Getenv("VERSION")
	if len(VERSION) == 0 {
		flag.StringVar(&VERSION, "v", "v0.1", "set version")
	}

	flag.Parse()
}

func GetPid() int {
	return PID
}

func GetHostName() string {
	return HOSTNAME
}

const (
	DEV  = "dev"
	TEST = "test"
	PROD = "prod"
)

func IsProdEnv() bool {
	return ENVIRONMENT == PROD
}

func IsTestEnv() bool {
	return ENVIRONMENT == TEST
}

func IsDevEnv() bool {
	return ENVIRONMENT == DEV
}

var (
	APPPATH = GetAppPath()
	APPNAME = GetAppName()
)

func GetAppPath() string {
	return common.GetAppPath()
}

func GetAppName() string {
	return common.GetAppName()
}
