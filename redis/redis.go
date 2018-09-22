package redis

import "github.com/hsyan2008/hfw2/configs"

func NewRedis(redisConfig configs.RedisConfig) (i RedisInterface, err error) {
	if redisConfig.IsCluster {
		return NewRedisCluster(redisConfig)
	} else {
		return NewRedisSimple(redisConfig)
	}
}
