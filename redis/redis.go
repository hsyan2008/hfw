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
