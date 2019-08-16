package common

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ParseFlag() {
	ENVIRONMENT = os.Getenv("ENVIRONMENT")
	if len(ENVIRONMENT) == 0 {
		flag.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	}

	VERSION = os.Getenv("VERSION")
	if len(VERSION) == 0 {
		flag.StringVar(&VERSION, "v", "v0.1", "set version")
	}

	flag.Parse()

	if len(ENVIRONMENT) == 0 && (IsGoRun() || IsGoTest()) {
		ENVIRONMENT = DEV
	}
}

var (
	appPath = getAppPath()
	appName = getAppName()

	APPPATH = GetAppPath()
	APPNAME = GetAppName()
)

func GetAppPath() string {
	return appPath
}

func getAppPath() (appPath string) {
	var err error
	pwd, _ := filepath.Abs(os.Args[0])
	if strings.Contains(pwd, "go-build") {
		pwd = stripSuffix(pwd)
		if strings.HasSuffix(pwd, ".test") {
			isGoTest = true
		} else {
			isGoRun = true
		}
		appPath, err = os.Getwd()
		if err != nil {
			panic(err)
		}
		minLen := 1
		if runtime.GOOS == "windows" {
			minLen = 3
		}
		for {
			if IsExist(filepath.Join(appPath, "config")) ||
				IsExist(filepath.Join(appPath, "main.go")) ||
				IsExist(filepath.Join(appPath, "controllers")) {
				return
			} else {
				if len(appPath) <= minLen {
					return
				}
				appPath = filepath.Dir(appPath)
			}
		}
	} else {
		appPath = filepath.Dir(pwd)
	}

	return
}

func GetAppName() string {
	return appName
}

func getAppName() (appName string) {
	if IsGoRun() || IsGoTest() {
		appName = filepath.Base(getAppPath())
	} else {
		pwd, _ := filepath.Abs(os.Args[0])
		appName = strings.ToLower(filepath.Base(pwd))
		appName = stripSuffix(appName)
	}

	return
}

func stripSuffix(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.TrimSuffix(path, ".exe")
	}

	return path
}

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
	//是否go run运行
	isGoRun bool
	//是否go test运行
	isGoTest bool
)

func IsGoRun() bool {
	return isGoRun
}

func IsGoTest() bool {
	return isGoTest
}
