package hfw

import (
	"fmt"
	"time"

	//mysql
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/cachestore"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/hsyan2008/go-logger/logger"
)

var engine *xorm.Engine

//InitDb init db
func initDb(dbConfig DbConfig, cacheServers []string) {

	var err error
	driver := dbConfig.Driver
	dbDsn := fmt.Sprintf("%s:%s@%s(%s)/%s%s",
		dbConfig.Username, dbConfig.Password, dbConfig.Protocol,
		dbConfig.Address, dbConfig.Dbname, dbConfig.Params)

	engine, err = xorm.NewEngine(driver, dbDsn)
	if err != nil {
		logger.Warn(err)
		panic(err)
	}

	if ENVIRONMENT != "prod" {
		engine.ShowSQL(true)
		engine.Logger().SetLevel(core.LOG_DEBUG)
	} else {
		engine.Logger().SetLevel(core.LOG_WARNING)
	}

	//连接池的空闲数大小
	engine.SetMaxIdleConns(dbConfig.MaxIdleConns)
	//最大打开连接数
	engine.SetMaxOpenConns(dbConfig.MaxOpenConns)

	go keepalive(time.Duration(dbConfig.KeepAlive))

	openCache(dbConfig.CacheType, cacheServers)
}

func openCache(cacheType string, cacheServers []string) {
	var cacher *xorm.LRUCacher

	//开启缓存
	switch cacheType {
	case "memory":
		cacher = xorm.NewLRUCacher(xorm.NewMemoryStore(), 1000)
	case "memcache":
		cacher = xorm.NewLRUCacher(cachestore.NewMemCache(cacheServers), 999999999)
	}
	if cacher != nil {
		//可以指定缓存有效时间，如下
		cacher.Expired = 86400 * time.Second
		//所有表开启缓存
		engine.SetDefaultCacher(cacher)
	}
}

//保持mysql连接活跃
func keepalive(long time.Duration) {
	for {
		time.Sleep(long * time.Second)
		_ = engine.Ping()
	}
}
