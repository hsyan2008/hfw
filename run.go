package hfw

import (
	"flag"
	"os"
	"strings"
	//pprof
	_ "net/http/pprof"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
)

//APPPATH 项目路径
var APPPATH string

func init() {
	initAPPPATH()
	loadConfig()
	setLog()
}

//初始化项目路径
func initAPPPATH() {
	pwd, _ := filepath.Abs(os.Args[0])
	//处理go run的情况此判断linux下有效
	if strings.Contains(pwd, "/tmp/go-build") {
		APPPATH, _ = os.Getwd()
	} else {
		APPPATH = filepath.Dir(pwd)
	}
}

//setLog 初始化log写入文件
func setLog() {
	lc := Config.Logger
	logger.SetLevelStr(lc.LogLevel)
	logger.SetConsole(lc.IsConsole)
	logger.SetLogGoID(lc.LogGoID)

	if lc.LogFile != "" {
		if lc.LogType == "daily" {
			logger.SetRollingDaily(lc.LogFile)
		} else if lc.LogType == "roll" {
			logger.SetRollingFile(lc.LogFile, lc.LogMaxNum, lc.LogSize, lc.LogUnit)
		} else {
			logger.Warn("undefined LogType")
		}
	} else {
		logger.Warn("undefined LogFile")
	}
}

func loadConfig() {

	flag.StringVar(&ENVIRONMENT, "e", "dev", "set env, e.g dev test prod")
	flag.Parse()

	configPath := filepath.Join(APPPATH, "config", ENVIRONMENT, "config.toml")
	if !IsExist(configPath) {
		panic("config file not exist")
	}
	_, err := toml.DecodeFile(configPath, &Config)
	if err != nil {
		panic(err)
	}

	// logger.Info(Config)

	//设置默认路由
	if Config.Route.DefaultController == "" {
		Config.Route.DefaultController = "index"
	}
	if Config.Route.DefaultAction == "" {
		Config.Route.DefaultAction = "index"
	}
	//转为绝对路径
	if !filepath.IsAbs(Config.Template.HTMLPath) {
		Config.Template.HTMLPath = filepath.Join(APPPATH, Config.Template.HTMLPath)
	}

	if !strings.Contains(Config.Server.Port, ":") {
		Config.Server.Port = ":" + Config.Server.Port
	}
}

//Run start
func Run() {

	//等待工作完成
	defer waitShutdownDone()

	certFile := Config.Server.HTTPSCertFile
	keyFile := Config.Server.HTTPSKeyFile

	if certFile != "" && !filepath.IsAbs(certFile) {
		certFile = filepath.Join(APPPATH, certFile)
	}

	if keyFile != "" && filepath.IsAbs(keyFile) {
		keyFile = filepath.Join(APPPATH, keyFile)
	}

	if IsExist(certFile) && IsExist(keyFile) {
		startHTTPSServe(certFile, keyFile)
	} else {
		startServe()
	}
}
