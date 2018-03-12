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

//APPNAME 项目名称
var APPNAME string

func init() {
	initAPPPATH()
	initAPPNAME()
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

func initAPPNAME() {
	pwd, _ := filepath.Abs(os.Args[0])
	if filepath.Base(pwd) == "main" {
		APPNAME = filepath.Base(APPPATH)
	} else {
		APPNAME = filepath.Base(pwd)
	}
}

//setLog 初始化log写入文件
func setLog() {
	lc := Config.Logger
	logger.SetLogGoID(lc.LogGoID)

	if lc.LogFile != "" {
		logger.SetLevelStr(lc.LogLevel)
		logger.SetConsole(lc.IsConsole)
		if lc.LogType == "daily" {
			logger.SetRollingDaily(lc.LogFile)
		} else if lc.LogType == "roll" {
			logger.SetRollingFile(lc.LogFile, lc.LogMaxNum, lc.LogSize, lc.LogUnit)
		} else {
			panic("undefined logtype")
		}
	} else {
		logger.SetLevelStr("debug")
		logger.SetRollingDaily(filepath.Join(APPPATH, APPNAME+".log"))
		logger.Info("undefined logfile, set debug level, log to console and default file")
	}
}

func loadConfig() {

	flag.StringVar(&ENVIRONMENT, "e", "dev", "set env, e.g dev test prod")
	flag.Parse()

	configPath := filepath.Join(APPPATH, "config", ENVIRONMENT, "config.toml")
	if !IsExist(configPath) {
		// panic("config file not exist")
		//如果文件不存在，直接返回，不进行初始化
		return
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

	if Config.Server.Port != "" && !strings.Contains(Config.Server.Port, ":") {
		Config.Server.Port = ":" + Config.Server.Port
	}
	//兼容
	if Config.Server.Address == "" && Config.Server.Port != "" {
		Config.Server.Address = Config.Server.Port
	}

	if Config.Server.ReadTimeout == 0 {
		Config.Server.ReadTimeout = 60
	}
	if Config.Server.WriteTimeout == 0 {
		Config.Server.WriteTimeout = 60
	}
}

//Run start
func Run() {

	logger.Debug("Pid:", os.Getpid(), "Starting ...")
	defer logger.Debug("Pid:", os.Getpid(), "Shutdown complete!")

	logger.Debug("Start to run, Config ENVIRONMENT is", ENVIRONMENT, "APPNAME is", APPNAME, "APPPATH is", APPPATH)

	//等待工作完成
	defer Shutdowned()

	if Config.Server.Address == "" {
		return
	}

	isHttp = true

	certFile := Config.Server.HTTPSCertFile
	keyFile := Config.Server.HTTPSKeyFile

	logger.Info("started server listen to ", Config.Server.Address)

	if certFile != "" && keyFile != "" {
		if !filepath.IsAbs(certFile) {
			certFile = filepath.Join(APPPATH, certFile)
		}

		if !filepath.IsAbs(keyFile) {
			keyFile = filepath.Join(APPPATH, keyFile)
		}

		logger.Info("https key is:", certFile, keyFile)

		if IsExist(certFile) && IsExist(keyFile) {
			startHTTPSServe(certFile, keyFile, Config.Server.HTTPSPhrase)
		} else {
			logger.Error("HTTPSCertFile and HTTPSKeyFile is not exist")
		}
	} else {
		startServe()
	}
}
