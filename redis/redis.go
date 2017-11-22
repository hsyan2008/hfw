package redis

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/hsyan2008/go-logger/logger"
	hfw "github.com/hsyan2008/hfw2"
)

var redisConfig = hfw.Config.Redis
var redisPool *redis.Pool

func init() {
	if redisConfig.Server != "" {
		redisPool = NewPool()
	}
}

func NewPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Other pool configuration not shown in this example.
		Dial: func() (redis.Conn, error) {

			dialOption := []redis.DialOption{
				redis.DialConnectTimeout(1 * time.Second),
				redis.DialReadTimeout(1 * time.Second),
				redis.DialWriteTimeout(1 * time.Second),
				redis.DialDatabase(redisConfig.Db),
			}
			if redisConfig.Password != "" {
				dialOption = append(dialOption, redis.DialPassword(redisConfig.Password))
			}

			c, err := redis.Dial("tcp", redisConfig.Server, dialOption...)

			if err != nil {
				return nil, err
			}

			return c, nil
		},
		// Other pool configuration not shown in this example.
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func IsExist(key string) (isExist bool, err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("IsExist cache key:", sessid, key)

	isExist, err = redis.Bool(redisPool.Get().Do("EXISTS", key))
	if err != nil {
		logger.Debug("IsExist cache key:", key, err)
	}

	return
}

func Set(key string, value interface{}) (err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := hfw.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("Set cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(redisPool.Get().Do("SET", key, v))
		if ok != "OK" {
			err = errors.New("set return not ok")
		}
		if err != nil {
			logger.Debug("Set cache key:", key, v, ok, err)
		}
	}

	return
}

func Get(key string) (value interface{}, err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Get cache key:", sessid, key)

	v, err := redisPool.Get().Do("GET", key)
	if err != nil {
		logger.Debug("Get cache key:", key, err)
	} else {
		err = hfw.Gob.Unmarshal(v.([]byte), &value)
		if err != nil {
			logger.Debug("Get cache Gob Unmarshal key:", key, err)
		}
	}

	return
}

func Del(key string) (isExist bool, err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Del cache key:", sessid, key)

	isExist, err = redis.Bool(redisPool.Get().Do("DEL", key))
	if err != nil {
		logger.Debug("Del cache key:", key, err)
	}

	return
}

func Setnx(key string, value interface{}) (isExist bool, err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := hfw.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("Setnx cache Gob Marshal key:", key, value, err)
	} else {
		isExist, err = redis.Bool(redisPool.Get().Do("SETNX", key, v))
		if err != nil {
			logger.Debug("Setnx cache key:", key, v, err)
		}
	}

	return
}

//set的复杂格式，支持过期时间
func SetEx(key string, value interface{}, expiration int) (err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := hfw.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("SetEx cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(redisPool.Get().Do("SET", key, v, "EX", expiration))
		if ok != "OK" {
			err = errors.New("SetEx return not ok")
		}
		if err != nil {
			logger.Debug("SetEx cache key:", key, v, ok, err)
		}
	}

	return
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func SetNxEx(key string, value interface{}, expiration int) (err error) {
	key = redisConfig.Prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := hfw.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("SetNxEx cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(redisPool.Get().Do("SET", key, v, "NX", "EX", expiration))
		if ok != "OK" {
			err = errors.New("SetNxEx return not ok")
		}
		if err != nil {
			logger.Debug("SetNxEx cache key:", key, v, ok, err)
		}
	}

	return
}
