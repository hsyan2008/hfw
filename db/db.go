package db

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/cachestore"
	"github.com/go-xorm/xorm"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/db/cache"
	"github.com/hsyan2008/hfw/encoding"
	"github.com/hsyan2008/hfw/signal"
)

var engineMap = new(sync.Map)

func InitDb(config configs.AllConfig, dbConfig configs.DbConfig) (engine xorm.EngineInterface, err error) {

	var isNew bool

	engine, isNew, err = getEngine(dbConfig.DbStdConfig)
	if err != nil {
		return engine, err
	}

	var isnew bool
	var slaveEngine *xorm.Engine
	if len(dbConfig.Slaves) > 0 {
		var slaves []*xorm.Engine
		for _, val := range dbConfig.Slaves {
			slaveEngine, isnew, err = getEngine(val)
			if err != nil {
				return engine, err
			}
			isNew = isNew || isnew
			slaves = append(slaves, slaveEngine)
		}
		engine, err = xorm.NewEngineGroup(engine, slaves)
		if err != nil {
			return nil, fmt.Errorf("NewEngineGroup dbConfig: %v failed: %v", dbConfig, err)
		}
	}

	engine.SetLogger(newXormLog())
	engine.ShowSQL(true)
	engine.ShowExecTime(true)

	if isNew {
		err = engine.Ping()
		if err != nil {
			return nil, fmt.Errorf("Ping dbConfig: %v failed: %v", dbConfig, err)
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

	return engine, nil
}

func getEngine(config configs.DbStdConfig) (engine *xorm.Engine, isNew bool, err error) {

	if config.Driver == "" {
		err = errors.New("dbConfig Driver is nil")
		return
	}

	driver := config.Driver
	dbDsn := getDbDsn(config)

	if e, ok := engineMap.Load(common.Md5(dbDsn)); ok {
		return e.(*xorm.Engine), isNew, nil
	}

	logger.Info("dbDsn:", dbDsn)

	engine, err = xorm.NewEngine(driver, dbDsn)
	if err != nil {
		return engine, isNew, fmt.Errorf("NewEngine dbConfig: %v failed: %v", config, err)
	}

	engineMap.Store(common.Md5(dbDsn), engine)
	isNew = true

	return
}

var dnsFuncMap = make(map[string]func(configs.DbStdConfig) string)

func getDbDsn(dbConfig configs.DbStdConfig) string {
	dbConfig.Driver = strings.ToLower(dbConfig.Driver)
	if f, ok := dnsFuncMap[dbConfig.Driver]; ok {
		return f(dbConfig)
	} else {
		panic("error db driver")
	}
}

var cacherMap = new(sync.Map)

func GetCacher(config configs.AllConfig, dbConfig configs.DbConfig) (cacher *xorm.LRUCacher, err error) {
	if dbConfig.CacheMaxSize == 0 {
		dbConfig.CacheMaxSize = 999999999
	}

	var key string
	switch dbConfig.CacheType {
	case "memory":
		key = common.Md5(dbConfig.CacheType)
	case "memcache":
		j, err := encoding.JSON.Marshal(config.Cache.Servers)
		if err != nil {
			panic(err)
		}
		key = common.Md5(dbConfig.CacheType + string(j))
	case "redis":
		j, err := encoding.JSON.Marshal(config.Redis)
		if err != nil {
			panic(err)
		}
		key = common.Md5(dbConfig.CacheType + string(j))
		// case "leveldb":
	default:
		return nil, errors.New("nil err cacheType")
	}

	if c, ok := cacherMap.Load(key); ok {
		return c.(*xorm.LRUCacher), nil
	}

	//开启缓存
	switch dbConfig.CacheType {
	case "memory":
		cacher = xorm.NewLRUCacher(xorm.NewMemoryStore(), dbConfig.CacheMaxSize)
	case "memcache":
		if len(config.Cache.Servers) == 0 {
			return nil, fmt.Errorf("nil memcache servers")
		}
		cacher = xorm.NewLRUCacher(cachestore.NewMemCache(config.Cache.Servers), dbConfig.CacheMaxSize)
	case "redis":
		cacheStore, err := cache.NewRedisCache(config.Redis)
		if err != nil {
			return nil, fmt.Errorf("NewRedisCache redisConfig: %v failed: %v", config.Redis, err)
		}
		cacher = xorm.NewLRUCacher(cacheStore, dbConfig.CacheMaxSize)
	}
	if cacher != nil {
		cacherMap.Store(key, cacher)
		cacher.Expired = dbConfig.CacheTimeout * time.Second
	}

	return
}

func keepalive(engine xorm.EngineInterface, long time.Duration) {
	if long <= 0 {
		return
	}
	t := time.Tick(long * time.Second)
	ctx := signal.GetSignalContext()
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
