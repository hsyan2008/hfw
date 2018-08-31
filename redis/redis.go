package redis

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/encoding"
)

func redisPool(redisConfig configs.RedisConfig) *redis.Pool {
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

func NewRedis(redisConfig configs.RedisConfig) *Redis {

	return &Redis{
		p:      redisPool(redisConfig),
		prefix: redisConfig.Prefix,
	}
}

type Redis struct {
	p      *redis.Pool
	prefix string
}

func (this *Redis) pool() redis.Conn {

	return this.p.Get()
}

func (this *Redis) IsExist(key string) (isExist bool, err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("IsExist cache key:", sessid, key)

	isExist, err = redis.Bool(this.pool().Do("EXISTS", key))
	if err != nil {
		logger.Debug("IsExist cache key:", key, err)
	}

	return
}

func (this *Redis) Set(key string, value interface{}) (err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("Set cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(this.pool().Do("SET", key, v))
		if ok != "OK" {
			err = errors.New("set return not ok")
		}
		if err != nil {
			logger.Debug("Set cache key:", key, v, ok, err)
		}
	}

	return
}

func (this *Redis) Get(key string) (value interface{}, err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Get cache key:", sessid, key)

	v, err := this.pool().Do("GET", key)
	if err != nil {
		logger.Debug("Get cache key:", key, err)
	} else {
		err = encoding.Gob.Unmarshal(v.([]byte), &value)
		if err != nil {
			logger.Debug("Get cache Gob Unmarshal key:", key, err)
		}
	}

	return
}

func (this *Redis) Del(key string) (isExist bool, err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Del cache key:", sessid, key)

	isExist, err = redis.Bool(this.pool().Do("DEL", key))
	if err != nil {
		logger.Debug("Del cache key:", key, err)
	}

	return
}

func (this *Redis) SetNx(key string, value interface{}) (isExist bool, err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("Setnx cache Gob Marshal key:", key, value, err)
	} else {
		isExist, err = redis.Bool(this.pool().Do("SETNX", key, v))
		if err != nil {
			logger.Debug("Setnx cache key:", key, v, err)
		}
	}

	return
}

//set的复杂格式，支持过期时间
func (this *Redis) SetEx(key string, value interface{}, expiration int) (err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("SetEx cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(this.pool().Do("SET", key, v, "EX", expiration))
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
func (this *Redis) SetNxEx(key string, value interface{}, expiration int) (err error) {
	key = this.prefix + key
	// key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	// Debug("Put cache key:", sessid, key, value)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		logger.Debug("SetNxEx cache Gob Marshal key:", key, value, err)
	} else {
		var ok string
		ok, err = redis.String(this.pool().Do("SET", key, v, "NX", "EX", expiration))
		if ok != "OK" {
			err = errors.New("SetNxEx return not ok")
		}
		if err != nil {
			logger.Debug("SetNxEx cache key:", key, v, ok, err)
		}
	}

	return
}

func (this *Redis) Hexists(key, field string) (value bool, err error) {
	key = this.prefix + key

	value, err = redis.Bool(this.pool().Do("HEXISTS", key, field))

	return
}
func (this *Redis) Hset(key, field string, value interface{}) (err error) {
	key = this.prefix + key

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		logger.Warn("Hset cache Gob Marshal key:", key, field, value, err)
	} else {
		_, err = this.pool().Do("HSET", key, field, v)
		if err != nil {
			logger.Warn("Hset cache key:", key, field, v, err)
		}
	}

	return
}

func (this *Redis) Hget(key, field string) (value interface{}, err error) {
	key = this.prefix + key

	v, err := this.pool().Do("HGET", key, field)
	if err != nil {
		logger.Warn("HGet cache key:", key, field, err)
	} else {
		err = encoding.Gob.Unmarshal(v.([]byte), &value)
		if err != nil {
			logger.Warn("HGet cache Gob Unmarshal key:", key, field, err)
		}
	}

	return
}
func (this *Redis) Hdel(key, field string) (err error) {
	key = this.prefix + key

	_, err = this.pool().Do("HDEL", key, field)
	if err != nil {
		logger.Warn("HDel cache key:", key, field, err)
	}

	return
}

func (this *Redis) Rename(oldKey, newKey string) (err error) {
	oldKey = this.prefix + oldKey
	newKey = this.prefix + newKey

	_, err = this.pool().Do("RENAME", oldKey, newKey)
	if err != nil {
		logger.Warn("Rename cache key:", oldKey, "to key:", newKey, err)
	}

	return
}

func (this *Redis) Expire(key string, expiration int32) {
	key = this.prefix + key

	if expiration > 0 {
		_, _ = this.pool().Do("EXPIRE", key, expiration)
	} else {
		_, _ = this.pool().Do("PERSIST", key)
	}

}
