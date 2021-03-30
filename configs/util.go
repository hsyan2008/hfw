package configs

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
)

func Load(config interface{}) (err error) {
	var configList []string
	defer func() {
		logger.Info("load config list:", configList)
		if err == ErrConfigPathNotExist {
			err = nil
		}
	}()
	//加载当前目录下的配置
	configList, err = loadFromFile(common.GetAppPath(), config)
	if err != nil {
		return
	}

	//加载通用配置
	configPath := filepath.Join(common.GetAppPath(), "config")
	list, err := loadFromFile(configPath, config)
	if err != nil {
		return
	}
	configList = append(configList, list...)

	//加载环境配置
	if len(common.GetEnv()) > 0 {
		envConfigPath := filepath.Join(configPath, common.GetEnv())
		if !(common.IsGoRun() || common.IsGoTest()) && !common.IsExist(envConfigPath) {
			//非go run和go test下，就必须存在对应目录加载配置
			return fmt.Errorf("config path: %s not exist", envConfigPath)
		}
		list, err = loadFromFile(envConfigPath, config)
		if err != nil {
			return
		}
		configList = append(configList, list...)
	}

	return nil
}

var ErrConfigPathNotExist = errors.New("config path not exist")

func loadFromFile(configPath string, config interface{}) ([]string, error) {
	if !common.IsExist(configPath) {
		logger.Infof("configPath: %s is not exist", configPath)
		return nil, ErrConfigPathNotExist
	}
	files, _ := filepath.Glob(filepath.Join(configPath, "*.toml"))
	for _, file := range files {
		_, err := toml.DecodeFile(file, config)
		if err != nil {
			return nil, err
		}
		logger.Info("load config from file success:", file)
	}

	return files, nil
}

func LoadDefaultConfig() (err error) {
	err = Load(&Config)
	if err != nil {
		return
	}
	return initDefaultConfig()
}

func initDefaultConfig() error {

	//错误码基数，如果小于10就认为是位数
	if Config.ErrorBase == 0 {
		Config.ErrorBase = 6
	}
	if Config.ErrorBase < 10 {
		Config.ErrorBase = int64(math.Pow10(int(Config.ErrorBase)))
	}

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
	Config.Server.CertFile = certFile
	Config.Server.KeyFile = keyFile

	//session
	if Config.EnableSession {
		if Config.Session.CookieName == "" {
			Config.Session.CookieName = "sessionid"
		}
		if Config.Session.CacheType == "" {
			Config.Session.CacheType = "redis"
		}
	}

	//redis
	if len(Config.Redis.Addresses) == 0 && Config.Redis.Server != "" {
		Config.Redis.Addresses = []string{Config.Redis.Server}
	}

	//prometheus
	if Config.Prometheus.IsEnable {
		if Config.Prometheus.RequestsTotal == "" {
			Config.Prometheus.RequestsTotal = "requests_total"
		}
		if Config.Prometheus.RequestsCosttime == "" {
			Config.Prometheus.RequestsCosttime = "requests_costtime"
		}
		if Config.Prometheus.RoutePath == "" {
			Config.Prometheus.RoutePath = "/metrics"
		}
		if len(Config.Prometheus.Tags) == 0 {
			Config.Prometheus.Tags = append(Config.Prometheus.Tags, "prometheus")
		}
	FOR:
		for _, val := range Config.Prometheus.Tags {
			for _, v := range Config.Server.Tags {
				if val == v {
					continue FOR
				}
			}
			Config.Server.Tags = append(Config.Server.Tags, val)
		}
	}

	return nil
}
