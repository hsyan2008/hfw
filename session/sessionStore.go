package session

import (
	"errors"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/redis"
)

type sessRedisStore struct {
	redisIns   redis.RedisInterface
	prefix     string
	expiration int32
}

var _ sessionStoreInterface = &sessRedisStore{}

var sessRedisStoreIns *sessRedisStore

func NewSessRedisStore(redisIns redis.RedisInterface, config configs.AllConfig) (*sessRedisStore, error) {
	if sessRedisStoreIns == nil {
		redisConfig := config.Redis
		sessConfig := config.Session
		if sessConfig.CookieName != "" && redisConfig.Server == "" {
			return nil, errors.New("session config error")
		}
		sessRedisStoreIns = &sessRedisStore{
			redisIns:   redisIns,
			prefix:     "sess_",
			expiration: redisConfig.Expiration,
		}
	}

	return sessRedisStoreIns, nil
}

func (s *sessRedisStore) SetExpiration(expiration int32) {
	s.expiration = expiration
}

func (s *sessRedisStore) IsExist(sessid, key string) (value bool, err error) {

	return s.redisIns.Hexists(s.prefix+sessid, key)
}

func (s *sessRedisStore) Put(sessid, key string, value interface{}) (err error) {

	return s.redisIns.Hset(s.prefix+sessid, key, value)
}

func (s *sessRedisStore) Get(sessid, key string) (value interface{}, err error) {

	return s.redisIns.Hget(s.prefix+sessid, key)
}

func (s *sessRedisStore) Del(sessid, key string) (err error) {

	return s.redisIns.Hdel(s.prefix+sessid, key)
}

func (s *sessRedisStore) Destroy(sessid string) (err error) {

	_, err = s.redisIns.Del(s.prefix + sessid)

	return
}

func (s *sessRedisStore) Rename(sessid, newid string) (err error) {

	b, err := s.redisIns.RenameNx(s.prefix+sessid, s.prefix+newid)
	if err != nil {
		return
	}

	if !b {
		return errors.New(newid + " is exist")
	}

	if s.expiration > 0 {
		s.redisIns.Expire(s.prefix+newid, s.expiration)
	}

	return
}
