package hfw

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/google/gops/agent"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/redis"
	"github.com/hsyan2008/hfw2/serve"
	"github.com/hsyan2008/hfw2/signal"
)

var (
	Config configs.AllConfig
)

func init() {
	parseFlag()
	loadConfig()
	initLog()
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
		var path string
		if common.IsExist("/opt/log") {
			path = filepath.Join("/opt/log", GetAppName()+".log")
		} else {
			path = filepath.Join(APPPATH, GetAppName()+".log")
		}
		logger.SetRollingFile(path, 2, 1, "GB")
	}

	if common.IsGoTest() {
		if testing.Verbose() {
			logger.SetConsole(true)
		} else {
			logger.SetConsole(false)
		}
	} else if common.IsGoRun() {
		logger.SetConsole(true)
	}

	// logger.SetPrefix(fmt.Sprintf("Pid:%d", GetPid()))
	logger.SetPrefix(filepath.Join(GetAppName(), GetEnv(), GetHostName(), GetVersion()))
}

func loadConfig() {
	if len(ENVIRONMENT) == 0 {
		//config目录存在的时候，就必须要指定环境变量来加载config.toml
		if common.IsExist(filepath.Join(APPPATH, "config")) {
			if common.IsGoRun() || common.IsGoTest() {
				ENVIRONMENT = DEV
			} else {
				panic("please specify env")
			}
		} else {
			logger.Warn("config dir not exist")
			return
		}
	}
	configPath := filepath.Join(APPPATH, "config", ENVIRONMENT, "config.toml")
	logger.Info("load config file: ", configPath)
	if common.IsExist(configPath) {
		_, err := toml.DecodeFile(configPath, &Config)
		if err != nil {
			panic(err)
		}
	} else {
		logger.Warnf("config file: %s not exist", configPath)
	}

	initConfig()
}

//Run start
func Run() (err error) {

	logger.Info("Starting ...")
	defer logger.Info("Shutdowned!")

	logger.Infof("Running, VERSION=%s, ENVIRONMENT=%s, APPNAME=%s, APPPATH=%s",
		GetVersion(), GetEnv(), GetAppName(), GetAppPath())

	if err = agent.Listen(agent.Options{}); err != nil {
		logger.Fatal(err)
		return
	}

	signalContext := signal.GetSignalContext()

	//监听信号
	go signalContext.Listen()

	//等待工作完成
	defer signalContext.Shutdowned()

	if Config.HotDeploy.Enable {
		go HotDeploy(Config.HotDeploy)
	}

	if len(Config.Server.Address) == 0 {
		logger.Warn("server address is nil")
		return
	}

	//启动http
	signalContext.IsHTTP = true

	err = serve.Start(Config.Server)

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

	certFile := Config.Server.CertFile
	keyFile := Config.Server.KeyFile
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
		redis.DefaultRedisIns, err = redis.NewRedis(Config.Redis)
		if err != nil {
			panic("error redis config:" + err.Error())
		}
	}
}
