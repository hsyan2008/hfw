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

var insMap = make(map[string]*Client)
var l = new(sync.Mutex)

func New(redisConfig configs.RedisConfig) (c *Client, err error) {
	if len(redisConfig.Addresses) == 0 {
		return c, errors.New("err redis config")
	}

	key, err := encoding.JSON.MarshalToString(redisConfig)
	if err != nil {
		return
	}
	l.Lock()
	defer l.Unlock()
	if c, ok := insMap[key]; ok {
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

	c = &Client{
		config: redisConfig,
		prefix: redisConfig.Prefix,

		Marshal:   encoding.JSON.Marshal,
		Unmarshal: encoding.JSON.Unmarshal,
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
		return
	}

	insMap[key] = c

	return
}

func isOk(s string) bool {
	if s == "OK" {
		return true
	}
	return false
}
