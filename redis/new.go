package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/encoding"
	radix "github.com/mediocregopher/radix/v3"
)

var ErrNotSupportCluster = errors.New("not support cluster")
var ErrParmasNotEnough = errors.New("params not enough")

var OK = "OK"

var insMap = make(map[string]radix.Client)
var l = new(sync.Mutex)

func newClient(redisConfig configs.RedisConfig) (c *Client, err error) {
	if len(redisConfig.Addresses) == 0 {
		return c, errors.New("err redis config")
	}

	key, err := encoding.JSON.MarshalToString(redisConfig)
	if err != nil {
		return
	}

	c = &Client{
		config: redisConfig,
		prefix: redisConfig.Prefix,

		Marshal:   encoding.JSON.Marshal,
		Unmarshal: encoding.JSON.Unmarshal,
	}

	l.Lock()
	defer l.Unlock()
	if i, ok := insMap[key]; ok {
		c.client = i
		return c, nil
	}

	if redisConfig.PoolSize <= 0 {
		redisConfig.PoolSize = 10
	}

	customConnFunc := func(network, addr string) (radix.Conn, error) {
		return radix.Dial(network, addr,
			radix.DialTimeout(3*time.Second),
			radix.DialAuthPass(redisConfig.Password),
			radix.DialSelectDB(redisConfig.Db),
		)
	}

	if redisConfig.IsCluster {
		clusterFunc := func(network, addr string) (radix.Client, error) {
			return radix.NewPool(network, addr, redisConfig.PoolSize, radix.PoolConnFunc(customConnFunc))
		}
		c.client, err = radix.NewCluster(redisConfig.Addresses, radix.ClusterPoolFunc(clusterFunc))
	} else {
		c.client, err = radix.NewPool("tcp", redisConfig.Addresses[0], redisConfig.PoolSize, radix.PoolConnFunc(customConnFunc))
	}
	if err != nil {
		c.client = nil
		return
	}

	insMap[key] = c.client

	return
}

//不能Close，会影响之前的连接
func Clone(src ...*Client) (dst *Client, err error) {
	c := DefaultIns
	if len(src) > 0 {
		c = src[0]
	}
	dst.client = c.client
	dst.prefix = c.prefix
	dst.Marshal = c.Marshal
	dst.Unmarshal = c.Unmarshal
	dst.config = c.config

	return
}

func closeClient(ins *Client) (err error) {
	if ins == nil || ins.client == nil {
		return nil
	}

	key, err := encoding.JSON.MarshalToString(ins.config)
	if err != nil {
		return
	}

	_ = ins.client.Close()
	ins.client = nil
	ins = nil

	l.Lock()
	defer l.Unlock()
	delete(insMap, key)

	return
}
