package common

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	appPath = getAppPath()
	appName = getAppName()
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
