package db

import (
	"fmt"
	"strings"
	"sync"
	"time"

	//mysql
	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/cachestore"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/hsyan2008/go-logger/logger"
	hfw "github.com/hsyan2008/hfw2"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/encoding"
)

var engineMap = new(sync.Map)

func InitDb(config configs.AllConfig, dbConfigs ...configs.DbConfig) (engine xorm.EngineInterface) {

	var dbConfig configs.DbConfig
	var err error
	var isNew bool

	if len(dbConfigs) > 0 {
		dbConfig = dbConfigs[0]
	} else {
		dbConfig = config.Db
	}

	engine, isNew = getEngine(dbConfig.DbStdConfig)

	var isnew bool
	var slaveEngine *xorm.Engine
	if len(dbConfig.Slaves) > 0 {
		var slaves []*xorm.Engine
		for _, val := range dbConfig.Slaves {
			slaveEngine, isnew = getEngine(val)
			isNew = isNew || isnew
			slaves = append(slaves, slaveEngine)
		}
		engine, err = xorm.NewEngineGroup(engine, slaves)
		if err != nil {
			logger.Warn("NewEngineGroup:", dbConfig, err)
			panic(err)
		}
	}

	engine.SetLogger(&mysqlLog{})
	engine.ShowSQL(true)
	engine.ShowExecTime(true)

	if isNew {
		err = engine.Ping()
		if err != nil {
			logger.Warn("Ping:", dbConfig, err)
			panic(err)
		}

		//连接池的空闲数大小
		if dbConfig.MaxIdleConns > 0 {
			engine.SetMaxIdleConns(dbConfig.MaxIdleConns)
		}
		//最大打开连接数
		if dbConfig.MaxOpenConns > 0 {
			engine.SetMaxOpenConns(dbConfig.MaxOpenConns)
		}

		go keepalive(engine, dbConfig.KeepAlive)
	}

	// openCache(engine, config)

	return engine
}
func getEngine(config configs.DbStdConfig) (engine *xorm.Engine, isNew bool) {

	if config.Driver == "" {
		panic("dbConfig Driver is nil")
	}

	driver := config.Driver
	dbDsn := getDbDsn(config)

	if e, ok := engineMap.Load(common.Md5(dbDsn)); ok {
		return e.(*xorm.Engine), isNew
	}

	logger.Info("dbDsn:", dbDsn)
	var err error

	engine, err = xorm.NewEngine(driver, dbDsn)
	if err != nil {
		logger.Warn("NewEngine:", config, err)
		panic(err)
	}

	engineMap.Store(common.Md5(dbDsn), engine)
	isNew = true

	return
}

func getDbDsn(dbConfig configs.DbStdConfig) string {
	switch strings.ToLower(dbConfig.Driver) {
	case "mysql":
		if dbConfig.Port != "" {
			dbConfig.Address = fmt.Sprintf("%s:%s", dbConfig.Address, dbConfig.Port)
		}
		return fmt.Sprintf("%s:%s@%s(%s)/%s%s",
			dbConfig.Username, dbConfig.Password, dbConfig.Protocol,
			dbConfig.Address, dbConfig.Dbname, dbConfig.Params)
	case "mssql", "sqlserver":
		return fmt.Sprintf("odbc:user id=%s;password=%s;server=%s;port=%s;database=%s;%s",
			dbConfig.Username, dbConfig.Password, dbConfig.Address, dbConfig.Port,
			dbConfig.Dbname, dbConfig.Params)
	default:
		panic("error db driver")
	}

}

func openCache(engine *xorm.Engine, config configs.AllConfig) {
	cacher := GetCacher(config)
	if cacher != nil {
		//所有表开启缓存
		engine.SetDefaultCacher(cacher)
	}
}

var cacherMap = new(sync.Map)

func GetCacher(config configs.AllConfig) (cacher *xorm.LRUCacher) {
	if config.Db.CacheMaxSize == 0 {
		config.Db.CacheMaxSize = 999999999
	}

	var key string
	switch config.Db.CacheType {
	case "memory":
		key = common.Md5(config.Db.CacheType)
	case "memcache":
		j, err := encoding.JSON.Marshal(config.Cache.Servers)
		if err != nil {
			panic(err)
		}
		key = common.Md5(config.Db.CacheType + string(j))
	case "redis":
		j, err := encoding.JSON.Marshal(config.Redis)
		if err != nil {
			panic(err)
		}
		key = common.Md5(config.Db.CacheType + string(j))
		// case "leveldb":
	}

	if len(key) == 0 {
		return
	}

	if c, ok := cacherMap.Load(key); ok {
		return c.(*xorm.LRUCacher)
	}

	//开启缓存
	switch config.Db.CacheType {
	case "memory":
		cacher = xorm.NewLRUCacher(xorm.NewMemoryStore(), config.Db.CacheMaxSize)
	case "memcache":
		if len(config.Cache.Servers) == 0 {
			return
		}
		cacher = xorm.NewLRUCacher(cachestore.NewMemCache(config.Cache.Servers), config.Db.CacheMaxSize)
	case "redis":
		if config.Redis.Server == "" {
			return
		}
		cf := map[string]string{
			"key":      config.Redis.Prefix + "mysqlCache",
			"password": config.Redis.Password,
			"conn":     config.Redis.Server,
		}
		cacher = xorm.NewLRUCacher(cachestore.NewRedisCache(cf), config.Db.CacheMaxSize)
		// case "leveldb":
		// 	cacher = xorm.NewLRUCacher(cachestore.NewLevelDBStore(cacheServers), config.Db.CacheMaxSize)
	}
	if cacher != nil {
		cacherMap.Store(key, cacher)
		//可以指定缓存有效时间，如下
		cacher.Expired = config.Db.CacheTimeout * time.Second
	}

	return
}

func keepalive(engine xorm.EngineInterface, long time.Duration) {
	if long <= 0 {
		return
	}
	t := time.Tick(long * time.Second)
	ctx := hfw.GetSignalContext()
FOR:
	for {
		select {
		case <-t:
			_ = engine.Ping()
		case <-ctx.Ctx.Done():
			break FOR
		}
	}
}

type mysqlLog struct {
	isShowSQL bool
}

func (mlog *mysqlLog) Debug(v ...interface{}) {
	logger.Output(4, "DEBUG", v...)
}
func (mlog *mysqlLog) Debugf(format string, v ...interface{}) {
	logger.Output(4, "DEBUG", fmt.Sprintf(format, v...))
}
func (mlog *mysqlLog) Info(v ...interface{}) {
	logger.Output(4, "INFO", v...)
}
func (mlog *mysqlLog) Infof(format string, v ...interface{}) {
	logger.Output(4, "INFO", fmt.Sprintf(format, v...))
}
func (mlog *mysqlLog) Warn(v ...interface{}) {
	logger.Output(4, "WARN", v...)
}
func (mlog *mysqlLog) Warnf(format string, v ...interface{}) {
	logger.Output(4, "WARN", fmt.Sprintf(format, v...))
}
func (mlog *mysqlLog) Error(v ...interface{}) {
	logger.Output(4, "ERROR", v...)
}
func (mlog *mysqlLog) Errorf(format string, v ...interface{}) {
	logger.Output(4, "ERROR", fmt.Sprintf(format, v...))
}

func (mlog *mysqlLog) Level() core.LogLevel {
	return core.LogLevel(logger.Level())
}

func (mlog *mysqlLog) SetLevel(l core.LogLevel) {
	logger.SetLevel(logger.LEVEL(l))
}

func (mlog *mysqlLog) ShowSQL(show ...bool) {
	mlog.isShowSQL = show[0]
}
func (mlog *mysqlLog) IsShowSQL() bool {
	return mlog.isShowSQL
}
