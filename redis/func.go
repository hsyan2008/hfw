package redis

import (
	"github.com/hsyan2008/hfw/encoding"
	radix "github.com/mediocregopher/radix/v3"
)

var DefaultIns = &Client{
	Marshal:   encoding.JSON.Marshal,
	Unmarshal: encoding.JSON.Unmarshal,
}

func Marshal(v interface{}) ([]byte, error) {
	return DefaultIns.Marshal(v)
}
func Unmarshal(data []byte, v interface{}) error {
	return DefaultIns.Unmarshal(data, v)
}

func Do(a radix.Action) error {
	return DefaultIns.Do(a)
}

func Close() error {
	return DefaultIns.Close()
}

//args可以是以下任意组合
// NX
// XX
// EX seconds
// PX milliseconds
func Set(key string, args ...interface{}) (b bool, err error) {
	return DefaultIns.Set(key, args...)
}

func Get(recv interface{}, key string) (b bool, err error) {
	return DefaultIns.Get(recv, key)
}

func MSet(items ...interface{}) (err error) {
	return DefaultIns.MSet(items...)
}

func MGet(keys ...string) (recv [][]byte, err error) {
	return DefaultIns.MGet(keys...)
}

func IsExist(key string) (isExist bool, err error) {
	return DefaultIns.IsExist(key)
}

func Incr(key string) (value int64, err error) {
	return DefaultIns.Incr(key)
}

func Decr(key string) (value int64, err error) {
	return DefaultIns.Decr(key)
}

func IncrBy(key string, delta int64) (value int64, err error) {
	return DefaultIns.IncrBy(key, delta)
}

func DecrBy(key string, delta int64) (value int64, err error) {
	return DefaultIns.DecrBy(key, delta)
}

func Del(keys ...string) (num int64, err error) {
	return DefaultIns.Del(keys...)
}

func SetNx(key string, value interface{}) (b bool, err error) {
	return DefaultIns.SetNx(key, value)
}

//set的复杂格式，支持过期时间
func SetEx(key string, value interface{}, expiration int64) (err error) {
	return DefaultIns.SetEx(key, value, expiration)
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func SetNxEx(key string, value interface{}, expiration int64) (b bool, err error) {
	return DefaultIns.SetNxEx(key, value, expiration)
}

func HSet(key, field string, value interface{}) (err error) {
	return DefaultIns.HSet(key, field, value)
}

func HGet(recv interface{}, key, field string) (b bool, err error) {
	return DefaultIns.HGet(recv, key, field)
}

func HExists(key, field string) (b bool, err error) {
	return DefaultIns.HExists(key, field)
}

func HIncrBy(key, field string, delta int64) (value int64, err error) {
	return DefaultIns.HIncrBy(key, field, delta)
}

func HDel(key string, fields ...string) (num int64, err error) {
	return DefaultIns.HDel(key, fields...)
}

func ZAdd(key string, args ...interface{}) (num int64, err error) {
	return DefaultIns.ZAdd(key, args...)
}

func ZRem(key string, members ...interface{}) (num int64, err error) {
	return DefaultIns.ZRem(key, members...)
}

func ZIncrBy(key, member string, increment int64) (num int64, err error) {
	return DefaultIns.ZIncrBy(key, member, increment)
}

//注意，返回的map不是有序的
func ZRange(key string, start, stop int64) (values map[string]int64, err error) {
	return DefaultIns.ZRange(key, start, stop)
}

//注意，返回的map不是有序的，所以其实和ZRange一样
func ZRevRange(key string, start, stop int64) (values map[string]int64, err error) {
	return DefaultIns.ZRevRange(key, start, stop)
}

//集群不支持RENAME
func Rename(oldKey, newKey string) (err error) {
	return DefaultIns.Rename(oldKey, newKey)
}

//集群不支持RENAMENX
func RenameNx(oldKey, newKey string) (b bool, err error) {
	return DefaultIns.RenameNx(oldKey, newKey)
}

func Expire(key string, expiration int64) (b bool, err error) {
	return DefaultIns.Expire(key, expiration)
}

func Ttl(key string) (num int64, err error) {
	return DefaultIns.Ttl(key)
}

//geo
//GEOADD key longitude latitude member [longitude latitude member ...]
func GeoAdd(key string, members ...interface{}) (num int64, err error) {
	return DefaultIns.GeoAdd(key, members...)
}

//GEODIST key member1 member2 [unit]]
func GeoDist(key string, args ...interface{}) (distance float64, err error) {
	return DefaultIns.GeoDist(key, args...)
}

//GEOHASH key member [member ...]
func GeoHash(key string, members ...string) (values []string, err error) {
	return DefaultIns.GeoHash(key, members...)
}

//GEOPOS key member [member ...]
func GeoPos(key string, members ...string) (values [][]float64, err error) {
	return DefaultIns.GeoPos(key, members...)
}

//GEORADIUS key longitude latitude radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
func GeoRadius(key string, args ...interface{}) (values [][]interface{}, err error) {
	return DefaultIns.GeoRadius(key, args...)
}

//GEORADIUSBYMEMBER key member radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
func GeoRadiusByMember(key string, args ...interface{}) (values [][]interface{}, err error) {
	return DefaultIns.GeoRadiusByMember(key, args...)
}
