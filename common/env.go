package common

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	appPath = getAppPath()
	appName = getAppName()

	APPPATH = GetAppPath()
	APPNAME = GetAppName()
)

func ParseFlag() (err error) {
	// restore := flag.CommandLine
	// defer func() {
	// 	flag.CommandLine = restore
	// }()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// flag.CommandLine.Usage = func() {}
	var buf = new(bytes.Buffer)
	flag.CommandLine.SetOutput(buf)
	flag.CommandLine.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	var p bool
	flag.CommandLine.BoolVar(&p, "p", false, "print version")

	loadFlag()

	err = flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		err = errors.New(buf.String())
	}

	if os.Getenv("VERSION") != "" {
		VERSION = os.Getenv("VERSION")
	}

	if len(ENVIRONMENT) == 0 {
		ENVIRONMENT = os.Getenv("ENVIRONMENT")
	}
	if len(ENVIRONMENT) == 0 && (IsGoRun() || IsGoTest()) {
		ENVIRONMENT = DEV
	}

	if p {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	return
}

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
	//通过go build -ldflags "-X github.com/hsyan2008/hfw/common.VERSION=v0.2"赋值
	//也可以通过环境变量赋值，此方式优先
	VERSION string = "v0.1"
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
