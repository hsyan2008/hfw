package redis

import "github.com/mediocregopher/radix.v2/redis"

type RedisInterface interface {
	getKey(string) string
	Cmd(string, ...interface{}) *redis.Resp

	IsExist(string) (bool, error)
	Set(string, interface{}, ...interface{}) (bool, error)
	MSet(...interface{}) error
	Get(string) (interface{}, error)
	MGet(...string) (map[string]interface{}, error)

	Incr(string) (int64, error)
	Decr(string) (int64, error)
	IncrBy(string, int64) (int64, error)
	DecrBy(string, int64) (int64, error)

	Del(...string) (int, error)

	SetNx(string, interface{}) (bool, error)
	SetEx(string, interface{}, int) error
	SetNxEx(string, interface{}, int) (bool, error)

	HExists(string, string) (bool, error)
	HSet(string, string, interface{}) error
	HGet(string, string) (interface{}, error)
	HIncrBy(string, string, int64) (int64, error)
	HDel(string, ...string) error

	ZIncrBy(string, string, float64) (string, error)
	ZRange(string, int64, int64) (map[string]string, error)
	ZRevRange(string, int64, int64) (map[string]string, error)

	Rename(string, string) error
	RenameNx(string, string) (bool, error)
	Expire(string, int32) (bool, error)

	GeoAdd(string, ...interface{}) (int, error)
	GeoDist(string, ...interface{}) (float64, error)
	GeoHash(string, ...string) (map[string]string, error)
	GeoPos(string, ...string) (map[string][2]float64, error)
	GeoRadius(string, ...interface{}) (map[string]float64, error)
	GeoRadiusByMember(string, ...interface{}) (map[string]float64, error)
}
