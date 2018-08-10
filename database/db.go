package database

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
	"github.com/hsyan2008/hfw2/configs"
)

type engineCache struct {
	engines map[string]*xorm.Engine
	mtx     *sync.Mutex
}

var ec = &engineCache{
	engines: make(map[string]*xorm.Engine),
	mtx:     new(sync.Mutex),
}

func InitDb(config configs.AllConfig, dbConfigs ...configs.DbConfig) *xorm.Engine {

	var dbConfig configs.DbConfig

	if len(dbConfigs) > 0 {
		dbConfig = dbConfigs[0]
	} else {
		dbConfig = config.Db
	}

	if dbConfig.Driver == "" {
		panic("dbConfig Driver is nil")
	}

	driver := dbConfig.Driver
	dbDsn := getDbDsn(dbConfig)

	ec.mtx.Lock()
	if e, ok := ec.engines[dbDsn]; ok {
		ec.mtx.Unlock()
		return e
	}

	logger.Info("dbDsn:", dbDsn)

	engine, err := xorm.NewEngine(driver, dbDsn)
	if err != nil {
		logger.Warn("NewEngine:", dbConfig, err)
		panic(err)
	}

	engine.SetLogger(&mysqlLog{})
	engine.ShowSQL(true)

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

	openCache(engine, config)

	ec.engines[dbDsn] = engine
	ec.mtx.Unlock()

	return engine
}

func getDbDsn(dbConfig configs.DbConfig) string {
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
	var cacher *xorm.LRUCacher

	//开启缓存
	switch config.Db.CacheType {
	case "memory":
		cacher = xorm.NewLRUCacher(xorm.NewMemoryStore(), 1000)
	case "memcache":
		if len(config.Cache.Servers) == 0 {
			return
		}
		cacher = xorm.NewLRUCacher(cachestore.NewMemCache(config.Cache.Servers), 999999999)
	case "redis":
		if config.Redis.Server == "" {
			return
		}
		cf := make(map[string]string)
		cf["key"] = config.Redis.Prefix + "mysqlCache"
		cf["password"] = config.Redis.Password
		cf["conn"] = config.Redis.Server
		cacher = xorm.NewLRUCacher(cachestore.NewRedisCache(cf), 999999999)
		// case "leveldb":
		// 	cacher = xorm.NewLRUCacher(cachestore.NewLevelDBStore(cacheServers), 999999999)
	}
	if cacher != nil {
		//可以指定缓存有效时间，如下
		cacher.Expired = 86400 * time.Second
		//所有表开启缓存
		engine.SetDefaultCacher(cacher)
	}
}

func keepalive(engine *xorm.Engine, long time.Duration) {
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
