package hfw

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/google/gops/agent"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/redis"
	"github.com/hsyan2008/hfw2/serve"
)

//ENVIRONMENT 环境
var ENVIRONMENT string

//VERSION 版本
var VERSION string

//APPPATH 项目路径
var APPPATH = common.GetAppPath()

//APPNAME 项目名称
var APPNAME = common.GetAppName()

var HOSTNAME, _ = os.Hostname()

var PID = os.Getpid()

var Config configs.AllConfig

var DefaultRedisIns redis.RedisInterface

func init() {
	parseFlag()
	loadConfig()
	initLog()
}

func parseFlag() {
	ENVIRONMENT = os.Getenv("ENVIRONMENT")
	if len(ENVIRONMENT) == 0 {
		flag.StringVar(&ENVIRONMENT, "e", "", "set env, e.g dev test prod")
	}

	VERSION = os.Getenv("VERSION")
	if len(VERSION) == 0 {
		flag.StringVar(&VERSION, "v", "0.1", "set version")
	}

	flag.Parse()
}

//setLog 初始化log写入文件
func initLog() {
	lc := Config.Logger
	logger.SetLogGoID(lc.LogGoID)

	if len(lc.LogFile) > 0 {
		logger.SetLevelStr(lc.LogLevel)
		logger.SetConsole(lc.IsConsole)
		if strings.ToLower(lc.LogType) == "daily" {
			logger.SetRollingDaily(lc.LogFile)
		} else if strings.ToLower(lc.LogType) == "roll" {
			logger.SetRollingFile(lc.LogFile, lc.LogMaxNum, lc.LogSize, lc.LogUnit)
		} else {
			panic("undefined logtype")
		}
	} else {
		logger.SetLevelStr("debug")
		if common.IsExist("/opt/log") {
			logger.SetRollingDaily(filepath.Join("/opt/log", APPNAME+".log"))
		} else {
			logger.SetRollingDaily(filepath.Join(APPPATH, APPNAME+".log"))
		}
	}

	if common.IsGoTest() {
		// if !testing.Verbose() {
		logger.SetConsole(false)
	}

	// logger.SetPrefix(fmt.Sprintf("Pid:%d", PID))
	logger.SetPrefix(HOSTNAME + "/" + VERSION)
}

func loadConfig() {
	if len(ENVIRONMENT) == 0 {
		if common.IsGoRun() {
			ENVIRONMENT = "dev"
		} else {
			panic("please specify env")
		}
	}
	configPath := filepath.Join(APPPATH, "config", ENVIRONMENT, "config.toml")
	if common.IsExist(configPath) {
		_, err := toml.DecodeFile(configPath, &Config)
		if err != nil {
			panic(err)
		}
	}

	initConfig()
}

//Run start
func Run() (err error) {

	logger.Info("Starting ...")
	defer logger.Info("Shutdowned!")

	logger.Infof("Running, VERSION=%s, ENVIRONMENT=%s, APPNAME=%s, APPPATH=%s", VERSION, ENVIRONMENT, APPNAME, APPPATH)

	if err = agent.Listen(agent.Options{}); err != nil {
		logger.Fatal(err)
		return
	}

	//监听信号
	go signalContext.listenSignal()

	//等待工作完成
	defer signalContext.Shutdowned()

	if Config.HotDeploy.Enable {
		go HotDeploy(Config.HotDeploy)
	}

	if len(Config.Server.Address) == 0 {
		logger.Warn("server address is nil")
		return
	}

	signalContext.IsHTTP = true

	if Config.Server.Concurrence > 0 {
		concurrenceChan = make(chan bool, Config.Server.Concurrence)
	}

	err = serve.Start(Config)

	//如果未启动服务，就触发退出
	if err != nil && err != http.ErrServerClosed {
		logger.Fatal(err)
	}

	return
}

func initConfig() {
	//设置默认路由
	if len(Config.Route.DefaultController) == 0 {
		Config.Route.DefaultController = "index"
	} else {
		Config.Route.DefaultController = strings.ToLower(Config.Route.DefaultController)
	}
	if len(Config.Route.DefaultAction) == 0 {
		Config.Route.DefaultAction = "index"
	} else {
		Config.Route.DefaultAction = strings.ToLower(Config.Route.DefaultAction)
	}

	//转为绝对路径
	if !filepath.IsAbs(Config.Template.HTMLPath) {
		Config.Template.HTMLPath = filepath.Join(APPPATH, Config.Template.HTMLPath)
	}
	if len(Config.Template.WidgetsPath) > 0 {
		if !filepath.IsAbs(Config.Template.WidgetsPath) {
			Config.Template.WidgetsPath = filepath.Join(APPPATH, Config.Template.WidgetsPath)
		}
		m, err := filepath.Glob(Config.Template.WidgetsPath)
		if err != nil || len(m) == 0 {
			panic("error WidgetsPath")
		}
	}

	if len(Config.Server.Port) > 0 && !strings.Contains(Config.Server.Port, ":") {
		Config.Server.Port = ":" + Config.Server.Port
	}
	//兼容
	if len(Config.Server.Address) == 0 && len(Config.Server.Port) > 0 {
		Config.Server.Address = Config.Server.Port
	}

	certFile := Config.Server.HTTPSCertFile
	keyFile := Config.Server.HTTPSKeyFile
	if len(certFile) > 0 && len(keyFile) > 0 {
		if !filepath.IsAbs(certFile) {
			certFile = filepath.Join(APPPATH, certFile)
		}

		if !filepath.IsAbs(keyFile) {
			keyFile = filepath.Join(APPPATH, keyFile)
		}
	}

	var err error
	if len(Config.Redis.Server) > 0 {
		DefaultRedisIns, err = redis.NewRedis(Config.Redis)
		if err != nil {
			panic("error redis config:" + err.Error())
		}
	}
}
