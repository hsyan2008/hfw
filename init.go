package hfw

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/redis"
)

var (
	Config configs.AllConfig
)

func Init() (err error) {
	common.ParseFlag()

	err = loadConfig()
	if err != nil {
		return err
	}

	err = initLog()
	if err != nil {
		return err
	}

	return
}

//setLog 初始化log写入文件
func initLog() error {
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
			return errors.New("undefined logtype")
		}
	} else {
		logger.SetLevelStr("debug")
		var path string
		if common.IsExist("/opt/log") {
			path = filepath.Join("/opt/log", common.GetAppName()+".log")
		} else {
			path = filepath.Join(common.GetAppPath(), common.GetAppName()+".log")
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
	logger.SetPrefix(filepath.Join(common.GetAppName(), common.GetEnv(), common.GetHostName(), common.GetVersion()))

	return nil
}

func loadConfig() (err error) {
	var configList []string
	defer func() {
		logger.Info("load config list:", configList)
	}()
	//加载当前目录下的配置
	configList, err = loadConfigFromFile(common.GetAppPath())
	if err != nil {
		return
	}

	//加载通用配置
	list, err := loadConfigFromFile(filepath.Join(common.GetAppPath(), "config"))
	if err != nil {
		return
	}
	configList = append(configList, list...)

	//加载环境配置
	if len(common.GetEnv()) > 0 {
		list, err = loadConfigFromFile(filepath.Join(common.GetAppPath(), "config", common.GetEnv()))
		if err != nil {
			return
		}
		configList = append(configList, list...)
	}

	return initConfig()
}

var ErrConfigPathNotExist = errors.New("config path not exist")

func loadConfigFromFile(configPath string) ([]string, error) {
	if !common.IsExist(configPath) {
		logger.Warnf("filepath: %s is not exist", configPath)
		return nil, ErrConfigPathNotExist
	}
	files, _ := filepath.Glob(filepath.Join(configPath, "*.toml"))
	for _, file := range files {
		// logger.Info("load config from file: ", file)
		_, err := toml.DecodeFile(file, &Config)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func initConfig() error {
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
		Config.Template.HTMLPath = filepath.Join(common.GetAppPath(), Config.Template.HTMLPath)
	}
	if len(Config.Template.WidgetsPath) > 0 {
		if !filepath.IsAbs(Config.Template.WidgetsPath) {
			Config.Template.WidgetsPath = filepath.Join(common.GetAppPath(), Config.Template.WidgetsPath)
		}
		m, err := filepath.Glob(Config.Template.WidgetsPath)
		if err != nil || len(m) == 0 {
			return errors.New("error WidgetsPath")
		}
	}

	certFile := Config.Server.CertFile
	keyFile := Config.Server.KeyFile
	if len(certFile) > 0 && len(keyFile) > 0 {
		if !filepath.IsAbs(certFile) {
			certFile = filepath.Join(common.GetAppPath(), certFile)
		}

		if !filepath.IsAbs(keyFile) {
			keyFile = filepath.Join(common.GetAppPath(), keyFile)
		}
	}

	var err error
	if len(Config.Redis.Server) > 0 {
		redis.DefaultRedisIns, err = redis.NewRedis(Config.Redis)
		if err != nil {
			return errors.New("error redis config:" + err.Error())
		}
	}

	return nil
}
