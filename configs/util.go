package configs

import (
	"errors"
	"fmt"
	"path/filepath"

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
