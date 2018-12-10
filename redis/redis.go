package redis

import (
	"errors"
	"math"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/encoding"
	"github.com/mediocregopher/radix.v2/redis"
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
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("EXISTS", key)
	if resp.Err != nil {
		return isExist, resp.Err
	}
	i, err := resp.Int()

	return i == 1, nil
}

func Set(key string, value interface{}) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return err
	}
	//OK
	_, err = DefaultRedisIns.Cmd("SET", key, v).Str()

	return
}

func MSet(items ...interface{}) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	for key, val := range items {
		if int(math.Mod(float64(key), 2)) == 0 {
			items[key] = DefaultRedisIns.getKey(val.(string))
		} else {
			v, err := encoding.Gob.Marshal(&val)
			if err != nil {
				return err
			}
			items[key] = v
		}
	}

	//OK
	_, err = DefaultRedisIns.Cmd("MSET", items).Str()

	return
}

func Get(key string) (value interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("GET", key)
	if resp.Err != nil {
		return value, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return value, nil
	}

	v, err := resp.Bytes()
	if err != nil {
		return
	}

	err = encoding.Gob.Unmarshal(v, &value)

	return
}

func MGet(keys ...string) (values map[string]interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	newKeys := make([]string, len(keys))
	for k, v := range keys {
		newKeys[k] = DefaultRedisIns.getKey(v)
	}

	resp := DefaultRedisIns.Cmd("MGET", newKeys)
	if resp.Err != nil {
		return values, resp.Err
	}

	if resp.IsType(redis.Array) {
		values = make(map[string]interface{})
		var value interface{}
		resps, err := resp.Array()
		if err != nil {
			return values, err
		}
		for k, resp1 := range resps {
			if resp1.IsType(redis.Nil) {
				continue
			}

			v, err := resp1.Bytes()
			if err != nil {
				return values, err
			}

			err = encoding.Gob.Unmarshal(v, &value)
			if err != nil {
				return values, err
			}
			values[keys[k]] = value
		}
	} else {
		return values, errors.New("mget error: not array")
	}

	return
}

func Incr(key string) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("INCR", key)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func Decr(key string) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("DECR", key)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func IncrBy(key string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("INCRBY", key, delta)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func DecrBy(key string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("DECRBY", key, delta)

	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func Del(keys ...string) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	for k, v := range keys {
		keys[k] = DefaultRedisIns.getKey(v)
	}

	resp := DefaultRedisIns.Cmd("DEL", keys)
	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()
	if err != nil {
		return
	}

	return i > 0, err
}

func SetNx(key string, value interface{}) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := DefaultRedisIns.Cmd("SET", key, v, "NX")
	if resp.Err != nil {
		return isOk, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return isOk, nil
	}

	//OK
	return true, nil
}

//set的复杂格式，支持过期时间
func SetEx(key string, value interface{}, expiration int) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := DefaultRedisIns.Cmd("SET", key, v, "EX", expiration)
	if resp.Err != nil {
		return resp.Err
	}

	//OK
	return
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func SetNxEx(key string, value interface{}, expiration int) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := DefaultRedisIns.Cmd("SET", key, v, "NX")
	if resp.Err != nil {
		return isOk, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return isOk, nil
	}

	//OK
	return true, nil
}

//当key存在，但不是hash类型，会报错AppErr
func HExists(key, field string) (value bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("HEXISTS", key, field)
	if resp.Err != nil {
		return value, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}

func HSet(key, field string, value interface{}) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := DefaultRedisIns.Cmd("HSET", key, field, v)
	if resp.Err != nil {
		return resp.Err
	}

	_, err = resp.Int()

	return
}

func HGet(key, field string) (value interface{}, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("HGET", key, field)
	if resp.Err != nil {
		return value, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return value, nil
	}

	v, err := resp.Bytes()
	if err != nil {
		return value, err
	}

	err = encoding.Gob.Unmarshal(v, &value)

	return
}

func HIncrBy(key, field string, delta int64) (value int64, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("HINCRBY", key, field, delta)

	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func HDel(key string, fields ...string) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	resp := DefaultRedisIns.Cmd("HDEL", key, fields)
	if resp.Err != nil {
		return resp.Err
	}

	_, err = resp.Int()

	return
}

func Rename(oldKey, newKey string) (err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	oldKey = DefaultRedisIns.getKey(oldKey)
	newKey = DefaultRedisIns.getKey(newKey)

	resp := DefaultRedisIns.Cmd("RENAME", oldKey, newKey)
	if resp.Err != nil {
		return resp.Err
	}

	return
}

func RenameNx(oldKey, newKey string) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	oldKey = DefaultRedisIns.getKey(oldKey)
	newKey = DefaultRedisIns.getKey(newKey)

	resp := DefaultRedisIns.Cmd("RENAMENX", oldKey, newKey)
	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}

func Expire(key string, expiration int32) (isOk bool, err error) {
	if DefaultRedisIns == nil {
		err = errors.New("redis instance need init")
		return
	}
	key = DefaultRedisIns.getKey(key)

	var resp *redis.Resp
	if expiration > 0 {
		resp = DefaultRedisIns.Cmd("EXPIRE", key, expiration)
	} else {
		resp = DefaultRedisIns.Cmd("PERSIST", key)
	}

	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}
