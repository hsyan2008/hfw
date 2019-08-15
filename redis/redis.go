package redis

import (
	"errors"

	"github.com/hsyan2008/hfw/configs"
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

func Del(keys ...string) (num int, err error) {
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

//不支持INCR，请用ZIncrBy代替
func ZAdd(key string, args ...interface{}) (num int, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.ZAdd(key, args...)
}

func ZRem(key string, members ...interface{}) (num int, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.ZRem(key, members...)
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

//geo
//GEOADD key longitude latitude member [longitude latitude member ...]
func GeoAdd(key string, members ...interface{}) (num int, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoAdd(key, members...)
}

//GEODIST key member1 member2 [unit]]
func GeoDist(key string, args ...interface{}) (distance float64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoDist(key, args...)
}

//GEOHASH key member [member ...]
func GeoHash(key string, members ...string) (values map[string]string, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoHash(key, members...)
}

//GEOPOS key member [member ...]
func GeoPos(key string, members ...string) (values map[string][2]float64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoPos(key, members...)
}

//GEORADIUS key longitude latitude radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
//为简单起便，三个WITH只且必须支持WITHDIST，返回距离
func GeoRadius(key string, args ...interface{}) (values []*Geo, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoRadius(key, args...)
}

//GEORADIUSBYMEMBER key member radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
//为简单起便，三个WITH只且必须支持WITHDIST，返回距离
func GeoRadiusByMember(key string, args ...interface{}) (values []*Geo, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}

	return DefaultRedisIns.GeoRadiusByMember(key, args...)
}
