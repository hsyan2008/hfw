package hfw

import (
	"flag"
	//pprof
	_ "net/http/pprof"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
)

func init() {
	loadConfig()
	setLog()
}

//Run start
func Run() {
	certFile := Config.Server.HTTPSCertFile
	keyFile := Config.Server.HTTPSKeyFile
	if IsExist(certFile) && IsExist(keyFile) {
		startHTTPSServe(certFile, keyFile)
	} else {
		startServe()
	}
}

//setLog 初始化log写入文件
func setLog() {
	lc := Config.Logger
	if lc.LogLevel == "" {
		return
	}
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

	switch ENVIRONMENT {
	case "dev":
		fallthrough
	case "test":
		fallthrough
	case "prod":
		configPath := filepath.Join("config", ENVIRONMENT, "config.toml")
		if !IsExist(configPath) {
			panic("config file not exist")
		}
		_, err := toml.DecodeFile(configPath, &Config)
		if err != nil {
			panic(err)
		}
	default:
		panic("error ENVIRONMENT")
	}

	// logger.Info(Config)

	//设置默认路由
	if Config.Route.DefaultController == "" {
		Config.Route.DefaultController = "index"
	}
	if Config.Route.DefaultAction == "" {
		Config.Route.DefaultAction = "index"
	}
}
