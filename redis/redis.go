package redis

import (
	"errors"

	"github.com/hsyan2008/hfw2/configs"
)

func NewRedis(redisConfig configs.RedisConfig) (i RedisInterface, err error) {
	if len(redisConfig.Server) == 0 {
		return nil, errors.New("err redis config")
	}

	if redisConfig.IsCluster {
		return NewRedisCluster(redisConfig)
	} else {
		return NewRedisSimple(redisConfig)
	}
}

var DefaultRedisIns RedisInterface

func IsExist(key string) (isExist bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.IsExist(key)
}

//args可以是以下任意组合
// NX
// XX
// EX seconds
// PX milliseconds
func Set(key string, value interface{}, args ...interface{}) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Set(key, value, args...)
}

func MSet(items ...interface{}) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.MSet(items...)
}

func Get(key string) (value interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Get(key)
}

func MGet(keys ...string) (values map[string]interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.MGet(keys...)
}

func Incr(key string) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Incr(key)
}

func Decr(key string) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Decr(key)
}

func IncrBy(key string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.IncrBy(key, delta)
}

func DecrBy(key string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.DecrBy(key, delta)
}

func Del(keys ...string) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Del(keys...)
}

func SetNx(key string, value interface{}) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.SetNx(key, value)
}

//set的复杂格式，支持过期时间
func SetEx(key string, value interface{}, expiration int) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.SetEx(key, value, expiration)
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func SetNxEx(key string, value interface{}, expiration int) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.SetNxEx(key, value, expiration)
}

//当key存在，但不是hash类型，会报错AppErr
func HExists(key, field string) (value bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.HExists(key, field)
}

func HSet(key, field string, value interface{}) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.HSet(key, field, value)
}

func HGet(key, field string) (value interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.HGet(key, field)
}

func HIncrBy(key, field string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.HIncrBy(key, field, delta)
}

func HDel(key string, fields ...string) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.HDel(key, fields...)
}

func ZIncrBy(key, member string, increment float64) (value string, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.ZIncrBy(key, member, increment)
}

func ZRange(key string, start, stop int64) (values map[string]string, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.ZRange(key, start, stop)
}

func ZRevRange(key string, start, stop int64) (values map[string]string, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.ZRevRange(key, start, stop)
}

func Rename(oldKey, newKey string) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Rename(oldKey, newKey)
}

func RenameNx(oldKey, newKey string) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.RenameNx(oldKey, newKey)
}

func Expire(key string, expiration int32) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.Expire(key, expiration)
}
