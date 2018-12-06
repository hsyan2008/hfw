package common

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/axgle/mahonia"
	"github.com/google/uuid"
)

var (
	appPath string
	appName string
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
	if appPath == "" {
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
			for len(appPath) > 0 {
				if IsExist(filepath.Join(appPath, "config")) ||
					IsExist(filepath.Join(appPath, "main.go")) ||
					IsExist(filepath.Join(appPath, "controllers")) {
					return appPath
				} else {
					appPath = filepath.Dir(appPath)
				}
			}
		} else {
			appPath = filepath.Dir(pwd)
		}
	}

	return appPath
}

func GetAppName() string {
	if appName == "" {
		GetAppPath()
		if isGoRun {
			appName = filepath.Base(appPath)
		} else {
			pwd, _ := filepath.Abs(os.Args[0])
			appName = strings.ToLower(filepath.Base(pwd))
			appName = stripSuffix(appName)
		}
	}
	return appName
}

func stripSuffix(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.TrimSuffix(path, ".exe")
	}

	return path
}

//Response ..
type Response struct {
	ErrNo   int64       `json:"err_no"`
	ErrMsg  string      `json:"err_msg"`
	Results interface{} `json:"results"`
}

//Max ..
func Max(i int, j ...int) int {
	for _, v := range j {
		if v > i {
			i = v
		}
	}
	return i
}

//Min ..
func Min(i int, j ...int) int {
	for _, v := range j {
		if v < i {
			i = v
		}
	}
	return i
}

//Md5 ..
func Md5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

//IsExist ...
func IsExist(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

//IsDir ...
func IsDir(filepath string) bool {
	f, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	return f.IsDir()
}

//转换为当前操作系统支持的编码
//linux和mac为utf8
//win为GBK
func ToOsCode(text string) string {
	if runtime.GOOS == "windows" {
		enc := mahonia.NewEncoder(("gbk"))
		return enc.ConvertString(text)
	}

	return text
}

func Uuid() string {
	if id, err := uuid.NewRandom(); err == nil {
		return id.String()
	}

	return ""
}
