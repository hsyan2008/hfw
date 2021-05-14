package redis

import (
	"errors"
	"fmt"

	"github.com/hsyan2008/hfw/configs"
	radix "github.com/mediocregopher/radix/v3"
)

type Client struct {
	client radix.Client
	prefix string
	config configs.RedisConfig

	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}

func (c *Client) Do(a radix.Action) error {
	if c == nil || c.client == nil {
		return errors.New("redis instance not init")
	}

	return c.client.Do(a)
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) AddPrefix(s string) string {
	return c.prefix + s
}

//args可以是以下任意组合
// NX
// XX
// EX seconds
// PX milliseconds
func (c *Client) Set(key string, args ...interface{}) (b bool, err error) {
	if len(args) == 0 {
		return b, ErrParmasNotEnough
	}
	args[0], err = c.Marshal(args[0])
	if err != nil {
		return
	}
	var s string
	err = c.Do(radix.FlatCmd(&s, "SET", c.AddPrefix(key), args...))
	return isOk(s), err
}

func (c *Client) Get(recv interface{}, key string) (b bool, err error) {
	var data []byte
	mn := radix.MaybeNil{Rcv: &data}
	err = c.Do(radix.Cmd(&mn, "GET", c.AddPrefix(key)))
	if err != nil {
		return
	}
	if mn.Nil {
		return
	}
	if len(data) > 0 {
		err = c.Unmarshal(data, &recv)
	}

	return true, err
}

func (c *Client) MSet(items ...interface{}) (err error) {
	if len(items)%2 != 0 {
		return ErrParmasNotEnough
	}

	if c.config.IsCluster {
		for i := 0; i < len(items); i += 2 {
			_, err = Set(items[i].(string), items[i+1])
			if err != nil {
				return
			}
		}
	} else {
		for i := 0; i < len(items); i += 2 {
			items[i] = c.AddPrefix(items[i].(string))
			items[i+1], err = c.Marshal(items[i+1])
			if err != nil {
				return
			}
		}

		err = c.Do(radix.FlatCmd(nil, "MSET", items[0].(string), items[1:]...))
	}

	return
}

func (c *Client) MGet(keys ...string) (recv [][]byte, err error) {
	if c.config.IsCluster {
		recv = make([][]byte, len(keys))
		var data []byte
		for k, v := range keys {
			err = c.Do(radix.Cmd(&data, "GET", c.AddPrefix(v)))
			if err != nil {
				return
			}
			recv[k] = data
		}
	} else {
		newKeys := make([]string, len(keys))
		for k, v := range keys {
			newKeys[k] = c.AddPrefix(v)
		}

		err = c.Do(radix.Cmd(&recv, "MGET", newKeys...))
	}

	return
}

func (c *Client) IsExist(key string) (isExist bool, err error) {
	err = c.Do(radix.Cmd(&isExist, "EXISTS", c.AddPrefix(key)))
	return
}

func (c *Client) Incr(key string) (value int64, err error) {
	err = c.Do(radix.Cmd(&value, "INCR", c.AddPrefix(key)))

	return
}

func (c *Client) Decr(key string) (value int64, err error) {
	err = c.Do(radix.Cmd(&value, "DECR", c.AddPrefix(key)))

	return
}

func (c *Client) IncrBy(key string, delta int64) (value int64, err error) {
	err = c.Do(radix.FlatCmd(&value, "INCRBY", c.AddPrefix(key), delta))

	return
}

func (c *Client) DecrBy(key string, delta int64) (value int64, err error) {
	err = c.Do(radix.FlatCmd(&value, "DECRBY", c.AddPrefix(key), delta))

	return
}

func (c *Client) Del(keys ...string) (num int64, err error) {
	if c.config.IsCluster {
		var i int64
		for _, v := range keys {
			err = c.Do(radix.Cmd(&i, "DEL", c.AddPrefix(v)))
			if err != nil {
				return
			}
			num += i
		}
	} else {
		for k, v := range keys {
			keys[k] = c.AddPrefix(v)
		}

		err = c.Do(radix.Cmd(&num, "DEL", keys...))
	}

	return
}

func (c *Client) SetNx(key string, value interface{}) (b bool, err error) {
	return Set(key, value, "NX")
}

//set的复杂格式，支持过期时间
func (c *Client) SetEx(key string, value interface{}, expiration int64) (err error) {
	_, err = Set(key, value, "EX", expiration)
	return
}

//set的复杂格式，支持过期时间，当key存在的时候不保存
func (c *Client) SetNxEx(key string, value interface{}, expiration int64) (b bool, err error) {
	return Set(key, value, "NX", "EX", expiration)
}

func (c *Client) HSet(key, field string, value interface{}) (err error) {
	v, err := c.Marshal(&value)
	if err != nil {
		return
	}

	err = c.Do(radix.FlatCmd(nil, "HSET", c.AddPrefix(key), field, v))

	return
}

func (c *Client) HGet(recv interface{}, key, field string) (b bool, err error) {
	var data []byte
	mn := radix.MaybeNil{Rcv: &data}
	err = c.Do(radix.Cmd(&mn, "HGET", c.AddPrefix(key), field))
	if err != nil {
		return
	}
	if mn.Nil {
		return
	}

	if len(data) > 0 {
		err = c.Unmarshal(data, &recv)
	}

	return true, err
}

func (c *Client) HExists(key, field string) (b bool, err error) {
	err = c.Do(radix.Cmd(&b, "HEXISTS", c.AddPrefix(key), field))

	return
}

func (c *Client) HIncrBy(key, field string, delta int64) (value int64, err error) {
	err = c.Do(radix.FlatCmd(&value, "HINCRBY", c.AddPrefix(key), field, delta))

	return
}

func (c *Client) HDel(key string, fields ...string) (num int64, err error) {
	err = c.Do(radix.FlatCmd(&num, "HDEL", c.AddPrefix(key), fields))

	return
}

func (c *Client) ZAdd(key string, args ...interface{}) (num int64, err error) {
	err = c.Do(radix.FlatCmd(&num, "ZADD", c.AddPrefix(key), args...))

	return
}

func (c *Client) ZRem(key string, members ...interface{}) (num int64, err error) {
	err = c.Do(radix.FlatCmd(&num, "ZREM", c.AddPrefix(key), members...))

	return
}

func (c *Client) ZIncrBy(key, member string, increment int64) (num int64, err error) {
	err = c.Do(radix.FlatCmd(&num, "ZINCRBY", c.AddPrefix(key), increment, member))

	return
}

//注意，返回的map不是有序的
func (c *Client) ZRange(key string, start, stop int64) (values map[string]int64, err error) {
	err = c.Do(radix.FlatCmd(&values, "ZRANGE", c.AddPrefix(key), start, stop, "WITHSCORES"))
	return
}

//注意，返回的map不是有序的，所以其实和ZRange一样
func (c *Client) ZRevRange(key string, start, stop int64) (values map[string]int64, err error) {
	err = c.Do(radix.FlatCmd(&values, "ZREVRANGE", c.AddPrefix(key), start, stop, "WITHSCORES"))
	return
}

//集群不支持RENAME
func (c *Client) Rename(oldKey, newKey string) (err error) {
	if oldKey == newKey {
		return
	}
	if c.config.IsCluster {
		var b bool
		b, err = IsExist(oldKey)
		if err != nil {
			return
		}
		if b == false {
			return fmt.Errorf("key: %s not exist", oldKey)
		}

		var data []byte
		err = c.Do(radix.Cmd(&data, "GET", c.AddPrefix(oldKey)))
		if err != nil {
			return
		}
		err = c.Do(radix.FlatCmd(nil, "SET", c.AddPrefix(newKey), data))
	} else {
		err = c.Do(radix.Cmd(nil, "RENAME", c.AddPrefix(oldKey), c.AddPrefix(newKey)))
	}

	return
}

//集群不支持RENAMENX
func (c *Client) RenameNx(oldKey, newKey string) (b bool, err error) {
	if oldKey == newKey {
		return
	}
	if c.config.IsCluster {
		b, err = IsExist(oldKey)
		if err != nil {
			return
		}
		if b == false {
			return b, fmt.Errorf("key: %s not exist", oldKey)
		}
		var data []byte
		err = c.Do(radix.Cmd(&data, "GET", c.AddPrefix(oldKey)))
		if err != nil {
			return
		}
		var s string
		err = c.Do(radix.FlatCmd(&s, "SET", c.AddPrefix(newKey), data, "NX"))
		return isOk(s), err
	} else {
		err = c.Do(radix.Cmd(&b, "RENAMENX", c.AddPrefix(oldKey), c.AddPrefix(newKey)))
	}

	return
}

func (c *Client) Expire(key string, expiration int64) (b bool, err error) {
	if expiration > 0 {
		err = c.Do(radix.FlatCmd(&b, "EXPIRE", c.AddPrefix(key), expiration))
	} else {
		err = c.Do(radix.Cmd(&b, "PERSIST", c.AddPrefix(key)))
	}

	return
}

func (c *Client) Ttl(key string) (num int64, err error) {
	err = c.Do(radix.Cmd(&num, "TTL", c.AddPrefix(key)))

	return
}

//geo
//GEOADD key longitude latitude member [longitude latitude member ...]
func (c *Client) GeoAdd(key string, members ...interface{}) (num int64, err error) {
	if len(members) == 0 || len(members)%3 != 0 {
		return 0, ErrParmasNotEnough
	}

	err = c.Do(radix.FlatCmd(&num, "GEOADD", c.AddPrefix(key), members...))

	return
}

//GEODIST key member1 member2 [unit]]
func (c *Client) GeoDist(key string, args ...interface{}) (distance float64, err error) {
	if len(args) < 2 || len(args) > 3 {
		return distance, ErrParmasNotEnough
	}

	err = c.Do(radix.FlatCmd(&distance, "GEODIST", c.AddPrefix(key), args...))

	return
}

//GEOHASH key member [member ...]
func (c *Client) GeoHash(key string, members ...string) (values []string, err error) {
	if len(members) < 1 {
		return nil, ErrParmasNotEnough
	}

	err = c.Do(radix.Cmd(&values, "GEOHASH", append([]string{c.AddPrefix(key)}, members...)...))

	return
}

//GEOPOS key member [member ...]
func (c *Client) GeoPos(key string, members ...string) (values [][]float64, err error) {
	if len(members) < 1 {
		return nil, ErrParmasNotEnough
	}

	err = c.Do(radix.Cmd(&values, "GEOPOS", append([]string{c.AddPrefix(key)}, members...)...))

	return
}

//GEORADIUS key longitude latitude radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
func (c *Client) GeoRadius(key string, args ...interface{}) (values [][]interface{}, err error) {
	if len(args) < 4 {
		return nil, ErrParmasNotEnough
	}

	err = c.Do(radix.FlatCmd(&values, "GEORADIUS", c.AddPrefix(key), args...))

	return
}

//GEORADIUSBYMEMBER key member radius m|km|ft|mi [WITHCOORD] [WITHDIST] [WITHHASH] [COUNT count]
func (c *Client) GeoRadiusByMember(key string, args ...interface{}) (values [][]interface{}, err error) {
	if len(args) < 3 {
		return nil, ErrParmasNotEnough
	}

	err = c.Do(radix.FlatCmd(&values, "GEORADIUSBYMEMBER", c.AddPrefix(key), args...))

	return
}

func (c *Client) LPush(key string, values ...interface{}) (num int64, err error) {
	if len(values) == 0 {
		return 0, ErrParmasNotEnough
	}
	for k, v := range values {
		values[k], err = c.Marshal(v)
		if err != nil {
			return
		}
	}
	err = c.Do(radix.FlatCmd(&num, "LPUSH", c.AddPrefix(key), values...))
	return
}

func (c *Client) LPop(recv interface{}, key string) (b bool, err error) {
	var data []byte
	mn := radix.MaybeNil{Rcv: &data}
	err = c.Do(radix.Cmd(&mn, "LPOP", c.AddPrefix(key)))
	if mn.Nil {
		return
	}
	if len(data) > 0 {
		err = c.Unmarshal(data, &recv)
	}
	return true, err
}

func (c *Client) RPush(key string, values ...interface{}) (num int64, err error) {
	if len(values) == 0 {
		return 0, ErrParmasNotEnough
	}
	for k, v := range values {
		values[k], err = c.Marshal(v)
		if err != nil {
			return
		}
	}
	err = c.Do(radix.FlatCmd(&num, "RPUSH", c.AddPrefix(key), values...))
	return
}

func (c *Client) RPop(recv interface{}, key string) (b bool, err error) {
	var data []byte
	mn := radix.MaybeNil{Rcv: &data}
	err = c.Do(radix.Cmd(&mn, "RPOP", c.AddPrefix(key)))
	if mn.Nil {
		return
	}
	if len(data) > 0 {
		err = c.Unmarshal(data, &recv)
	}
	return true, err
}
