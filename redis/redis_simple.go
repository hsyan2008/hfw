package redis

import (
	"errors"
	"time"

	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/encoding"
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
	if redisConfig.PoolSize <= 0 {
		redisConfig.PoolSize = 10
	}
	p, err := pool.NewCustom("tcp", redisConfig.Server, redisConfig.PoolSize, df, pool.PingInterval(time.Hour))

	if err != nil {
		return
	}

	return &RedisSimple{
		p:      p,
		prefix: redisConfig.Prefix,
	}, nil
}

func (this *RedisSimple) Close() {
	this.p.Empty()
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

//args可以是以下任意组合
// NX
// XX
// EX seconds
// PX milliseconds
func (this *RedisSimple) Set(key string, value interface{}, args ...interface{}) (isOk bool, err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return false, err
	}
	var resp *redis.Resp
	if len(args) > 0 {
		resp = this.Cmd("SET", append([]interface{}{key, v}, args...)...)
	} else {
		resp = this.Cmd("SET", key, v)
	}
	if resp.Err != nil {
		return false, resp.Err
	}
	if resp.IsType(redis.Nil) {
		return false, nil
	}

	return true, nil
}

func (this *RedisSimple) MSet(items ...interface{}) (err error) {
	for key, val := range items {
		if key%2 == 0 {
			items[key] = this.getKey(val.(string))
		} else {
			v, err := encoding.Gob.Marshal(&val)
			if err != nil {
				return err
			}
			items[key] = v
		}
	}
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

func (this *RedisSimple) Del(keys ...string) (num int, err error) {
	for k, v := range keys {
		keys[k] = this.getKey(v)
	}

	resp := this.Cmd("DEL", keys)
	if resp.Err != nil {
		return 0, resp.Err
	}

	return resp.Int()
}

func (this *RedisSimple) SetNx(key string, value interface{}) (isOk bool, err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("SET", key, v, "NX")
	if resp.Err != nil {
		return false, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return false, nil
	}

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

	return
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func (this *RedisSimple) SetNxEx(key string, value interface{}, expiration int) (isOk bool, err error) {
	key = this.getKey(key)

	v, err := encoding.Gob.Marshal(&value)
	if err != nil {
		return
	}

	resp := this.Cmd("SET", key, v, "NX", "EX", expiration)
	if resp.Err != nil {
		return false, resp.Err
	}

	if resp.IsType(redis.Nil) {
		return false, nil
	}

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

//不支持INCR，请用ZIncrBy代替
func (this *RedisSimple) ZAdd(key string, args ...interface{}) (num int, err error) {
	key = this.getKey(key)

	resp := this.Cmd("ZADD", append([]interface{}{key}, args...)...)
	if resp.Err != nil {
		return 0, resp.Err
	}

	return resp.Int()
}

func (this *RedisSimple) ZRem(key string, members ...interface{}) (num int, err error) {
	key = this.getKey(key)

	resp := this.Cmd("ZREM", append([]interface{}{key}, members...)...)
	if resp.Err != nil {
		return 0, resp.Err
	}

	return resp.Int()
}

func (this *RedisSimple) ZIncrBy(key, member string, increment float64) (value string, err error) {
	key = this.getKey(key)

	resp := this.Cmd("ZINCRBY", key, increment, member)
	if resp.Err != nil {
		return "", resp.Err
	}

	return resp.Str()
}

func (this *RedisSimple) ZRange(key string, start, stop int64) (values map[string]string, err error) {
	key = this.getKey(key)

	resp := this.Cmd("ZRANGE", key, start, stop, "WITHSCORES")
	if resp.Err != nil {
		return nil, resp.Err
	}

	if resp.IsType(redis.Array) {
		values = make(map[string]string)
		resps, err := resp.Array()
		if err != nil {
			return nil, err
		}
		arrLen := len(resps)
		if arrLen%2 != 0 {
			return nil, errors.New("err resp num")
		}
		for i := 0; i < arrLen; i += 2 {
			if resps[i].IsType(redis.Nil) || resps[i+1].IsType(redis.Nil) {
				continue
			}
			k, err := resps[i].Str()
			if err != nil {
				return nil, err
			}
			v, err := resps[i+1].Str()
			if err != nil {
				return nil, err
			}

			values[k] = v
		}
	}

	return
}

func (this *RedisSimple) ZRevRange(key string, start, stop int64) (values map[string]string, err error) {
	key = this.getKey(key)

	resp := this.Cmd("ZREVRANGE", key, start, stop, "WITHSCORES")
	if resp.Err != nil {
		return nil, resp.Err
	}

	if resp.IsType(redis.Array) {
		values = make(map[string]string)
		resps, err := resp.Array()
		if err != nil {
			return nil, err
		}
		arrLen := len(resps)
		if arrLen%2 != 0 {
			return nil, errors.New("err resp num")
		}
		for i := 0; i < arrLen; i += 2 {
			if resps[i].IsType(redis.Nil) || resps[i+1].IsType(redis.Nil) {
				continue
			}
			k, err := resps[i].Str()
			if err != nil {
				return nil, err
			}
			v, err := resps[i+1].Str()
			if err != nil {
				return nil, err
			}

			values[k] = v
		}
	}

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

//geo
//GEOADD key longitude latitude member [longitude latitude member ...]
func (this *RedisSimple) GeoAdd(key string, members ...interface{}) (num int, err error) {
	if len(members)%3 != 0 {
		return 0, errors.New("error params number")
	}
	key = this.getKey(key)

	resp := this.Cmd("GEOADD", append([]interface{}{key}, members...)...)
	if resp.Err != nil {
		return 0, resp.Err
	}

	return resp.Int()
}

//GEODIST key member1 member2 [unit]]
func (this *RedisSimple) GeoDist(key string, args ...interface{}) (distance float64, err error) {
	if len(args) < 2 || len(args) > 3 {
		return distance, errors.New("error params number")
	}
	key = this.getKey(key)

	resp := this.Cmd("GEODIST", append([]interface{}{key}, args...)...)
	if resp.Err != nil {
		return distance, resp.Err
	}

	return resp.Float64()
}

//GEOHASH key member [member ...]
func (this *RedisSimple) GeoHash(key string, members ...string) (values map[string]string, err error) {
	if len(members) < 1 {
		return nil, errors.New("error params number")
	}
	key = this.getKey(key)

	args := make([]interface{}, len(members)+1)
	args[0] = key
	for k, v := range members {
		args[k+1] = v
	}
	resp := this.Cmd("GEOHASH", args...)
	if resp.Err != nil {
		return nil, resp.Err
	}
	if resp.IsType(redis.Array) {
		values = make(map[string]string)
		resps, err := resp.Array()
		if err != nil {
			return values, err
		}
		for k, resp1 := range resps {
			if resp1.IsType(redis.Nil) {
				continue
			}

			v, err := resp1.Str()
			if err != nil {
				return values, err
			}
			values[members[k]] = v
		}
	}

	return
}

//GEOPOS key member [member ...]
func (this *RedisSimple) GeoPos(key string, members ...string) (values map[string][2]float64, err error) {
	if len(members) < 1 {
		return nil, errors.New("error params number")
	}
	key = this.getKey(key)

	args := make([]interface{}, len(members)+1)
	args[0] = key
	for k, v := range members {
		args[k+1] = v
	}
	resp := this.Cmd("GEOPOS", args...)
	if resp.Err != nil {
		return nil, resp.Err
	}
	if resp.IsType(redis.Array) {
		values = make(map[string][2]float64)
		resps, err := resp.Array()
		if err != nil {
			return values, err
		}
		for k, resp1 := range resps {
			if resp1.IsType(redis.Nil) {
				continue
			}

			v, err := resp1.Array()
			if err != nil {
				return values, err
			}
			if len(v) != 2 {
				return values, errors.New("error number response")
			}
			longitude, err := v[0].Float64()
			if err != nil {
				return values, err
			}
			latitude, err := v[1].Float64()
			if err != nil {
				return values, err
			}

			values[members[k]] = [2]float64{
				longitude, latitude,
			}
		}
	}

	return
}

//GEORADIUS key longitude latitude radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
//为简单起便，三个WITH只且必须支持WITHDIST，返回距离
func (this *RedisSimple) GeoRadius(key string, args ...interface{}) (values []*Geo, err error) {
	if len(args) < 4 {
		return nil, errors.New("error params number")
	}
	key = this.getKey(key)

	resp := this.Cmd("GEORADIUS", append([]interface{}{key}, args...)...)
	if resp.Err != nil {
		return nil, resp.Err
	}
	if resp.IsType(redis.Array) {
		resps, err := resp.Array()
		if err != nil {
			return values, err
		}
		for _, resp1 := range resps {
			if resp1.IsType(redis.Nil) {
				continue
			}

			v, err := resp1.Array()
			if err != nil {
				return values, err
			}
			if len(v) != 2 {
				return values, errors.New("error number response")
			}
			member, err := v[0].Str()
			if err != nil {
				return values, err
			}
			radius, err := v[1].Float64()
			if err != nil {
				return values, err
			}

			values = append(values, &Geo{Member: member, Dist: radius})
		}
	}

	return
}

//GEORADIUSBYMEMBER key member radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
//为简单起便，三个WITH只且必须支持WITHDIST，返回距离
func (this *RedisSimple) GeoRadiusByMember(key string, args ...interface{}) (values []*Geo, err error) {
	if len(args) < 4 {
		return nil, errors.New("error params number")
	}
	key = this.getKey(key)

	resp := this.Cmd("GEORADIUSBYMEMBER", append([]interface{}{key}, args...)...)
	if resp.Err != nil {
		return nil, resp.Err
	}
	if resp.IsType(redis.Array) {
		resps, err := resp.Array()
		if err != nil {
			return values, err
		}
		for _, resp1 := range resps {
			if resp1.IsType(redis.Nil) {
				continue
			}

			v, err := resp1.Array()
			if err != nil {
				return values, err
			}
			if len(v) != 2 {
				return values, errors.New("error number response")
			}
			member, err := v[0].Str()
			if err != nil {
				return values, err
			}
			radius, err := v[1].Float64()
			if err != nil {
				return values, err
			}

			values = append(values, &Geo{Member: member, Dist: radius})
		}
	}

	return
}
