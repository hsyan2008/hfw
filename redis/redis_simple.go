package redis

import (
	"errors"
	"math"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/encoding"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

type RedisSimple struct {
	p      *pool.Pool
	prefix string
}

var _ RedisInterface = &RedisSimple{}

func NewRedisSimple(redisConfig configs.RedisConfig) (rs *RedisSimple, err error) {
	df := func(network, addr string) (*redis.Client, error) {
		client, err := redis.Dial(network, addr)
		if err != nil {
			return nil, err
		}
		if len(redisConfig.Password) > 0 {
			if err = client.Cmd("AUTH", redisConfig.Password).Err; err != nil {
				client.Close()
				return nil, err
			}
		}
		if redisConfig.Db > 0 {
			if err = client.Cmd("SELECT", redisConfig.Db).Err; err != nil {
				client.Close()
				return nil, err
			}
		}
		return client, nil
	}
	p, err := pool.NewCustom("tcp", redisConfig.Server, 10, df)

	if err != nil {
		return
	}

	return &RedisSimple{
		p:      p,
		prefix: redisConfig.Prefix,
	}, nil
}

func (this *RedisSimple) getKey(key string) string {
	return this.prefix + key
}

func (this *RedisSimple) Cmd(cmd string, args ...interface{}) (resp *redis.Resp) {
	c, err := this.p.Get()
	if err != nil {
		resp.Err = err
	}
	defer this.p.Put(c)

	return c.Cmd(cmd, args)
}

func (this *RedisSimple) IsExist(key string) (isExist bool, err error) {
	key = this.getKey(key)

	resp := this.Cmd("EXISTS", key)
	if resp.Err != nil {
		return isExist, resp.Err
	}
	i, err := resp.Int()

	return i == 1, nil
}

func (this *RedisSimple) Set(key string, value interface{}) (err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return err
	}
	//OK
	_, err = this.Cmd("SET", key, v).Str()

	return
}

func (this *RedisSimple) MSet(items ...interface{}) (err error) {
	for key, val := range items {
		if int(math.Mod(float64(key), 2)) == 0 {
			items[key] = this.getKey(val.(string))
		} else {
			v, err := encoding.Gob.Marshal(&val)
			if err != nil {
				return err
			}
			items[key] = v
		}
	}

	//OK
	_, err = this.Cmd("MSET", items).Str()

	return
}

func (this *RedisSimple) Get(key string) (value interface{}, err error) {
	key = this.getKey(key)

	resp := this.Cmd("GET", key)
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

func (this *RedisSimple) MGet(keys ...string) (values map[string]interface{}, err error) {
	newKeys := make([]string, len(keys))
	for k, v := range keys {
		newKeys[k] = this.getKey(v)
	}

	resp := this.Cmd("MGET", newKeys)
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

func (this *RedisSimple) Incr(key string) (value int64, err error) {
	key = this.getKey(key)

	resp := this.Cmd("INCR", key)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func (this *RedisSimple) Decr(key string) (value int64, err error) {
	key = this.getKey(key)

	resp := this.Cmd("DECR", key)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func (this *RedisSimple) IncrBy(key string, delta int64) (value int64, err error) {
	key = this.getKey(key)

	resp := this.Cmd("INCRBY", key, delta)
	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func (this *RedisSimple) DecrBy(key string, delta int64) (value int64, err error) {
	key = this.getKey(key)

	resp := this.Cmd("DECRBY", key, delta)

	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func (this *RedisSimple) Del(keys ...string) (isOk bool, err error) {
	for k, v := range keys {
		keys[k] = this.getKey(v)
	}

	resp := this.Cmd("DEL", keys)
	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()
	if err != nil {
		return
	}

	return i > 0, err
}

func (this *RedisSimple) SetNx(key string, value interface{}) (isOk bool, err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("SET", key, v, "NX")
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
func (this *RedisSimple) SetEx(key string, value interface{}, expiration int) (err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("SET", key, v, "EX", expiration)
	if resp.Err != nil {
		return resp.Err
	}

	//OK
	return
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func (this *RedisSimple) SetNxEx(key string, value interface{}, expiration int) (isOk bool, err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("SET", key, v, "NX")
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
func (this *RedisSimple) HExists(key, field string) (value bool, err error) {
	key = this.getKey(key)

	resp := this.Cmd("HEXISTS", key, field)
	if resp.Err != nil {
		return value, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}

func (this *RedisSimple) HSet(key, field string, value interface{}) (err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("HSET", key, field, v)
	if resp.Err != nil {
		return resp.Err
	}

	_, err = resp.Int()

	return
}

func (this *RedisSimple) HGet(key, field string) (value interface{}, err error) {
	key = this.getKey(key)

	resp := this.Cmd("HGET", key, field)
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

func (this *RedisSimple) HIncrBy(key, field string, delta int64) (value int64, err error) {
	key = this.getKey(key)

	resp := this.Cmd("HINCRBY", key, field, delta)

	if resp.Err != nil {
		return value, resp.Err
	}

	return resp.Int64()
}

func (this *RedisSimple) HDel(key string, fields ...string) (err error) {
	key = this.getKey(key)

	resp := this.Cmd("HDEL", key, fields)
	if resp.Err != nil {
		return resp.Err
	}

	_, err = resp.Int()

	return
}

func (this *RedisSimple) Rename(oldKey, newKey string) (err error) {
	oldKey = this.getKey(oldKey)
	newKey = this.getKey(newKey)

	resp := this.Cmd("RENAME", oldKey, newKey)
	if resp.Err != nil {
		return resp.Err
	}

	return
}

func (this *RedisSimple) RenameNx(oldKey, newKey string) (isOk bool, err error) {
	oldKey = this.getKey(oldKey)
	newKey = this.getKey(newKey)

	resp := this.Cmd("RENAMENX", oldKey, newKey)
	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}

func (this *RedisSimple) Expire(key string, expiration int32) (isOk bool, err error) {
	key = this.getKey(key)

	var resp *redis.Resp
	if expiration > 0 {
		resp = this.Cmd("EXPIRE", key, expiration)
	} else {
		resp = this.Cmd("PERSIST", key)
	}

	if resp.Err != nil {
		return isOk, resp.Err
	}

	i, err := resp.Int()

	return i == 1, err
}
