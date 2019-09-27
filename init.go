package hfw

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/db"
	"github.com/hsyan2008/hfw/redis"
)

var (
	//废弃，请用configs.Config
	Config   configs.AllConfig
	isInited bool
)

func init() {
	Init()
}

func Init() (err error) {
	if isInited {
		return
	}
	isInited = true

	err = common.ParseFlag()
	if err != nil {
		logger.Warn(err)
	}

	err = configs.LoadDefaultConfig()
	if err != nil {
		logger.Warn(err)
		return err
	}
	Config = configs.Config

	err = initLog()
	if err != nil {
		logger.Warn(err)
		return err
	}

	//初始化redis
	if len(Config.Redis.Server) > 0 && redis.DefaultRedisIns == nil {
		logger.Info("begin to connect default REDIS server")
		redis.DefaultRedisIns, err = redis.NewRedis(Config.Redis)
		if err != nil {
			logger.Warn("connect to default REDIS faild:", err)
			return fmt.Errorf("connect to default redis faild: %s", err.Error())
		}
		logger.Info("connect to default REDIS server success")
	}

	//初始化mysql
	if Config.Db.Driver != "" {
		logger.Info("begin connect to default MYSQL server")
		db.DefaultDao, err = db.NewXormDao(Config, Config.Db)
		if err != nil {
			logger.Warn("connect to default MYSQL faild:", err)
			return fmt.Errorf("connect to default mysql faild: %s", err.Error())
		}
		logger.Info("connect to default MYSQL server success")
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
