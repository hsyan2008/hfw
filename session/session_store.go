package session

import (
	"errors"
	"sync"

	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/redis"
)

type sessRedisStore struct {
	redisIns   *redis.Client
	prefix     string
	expiration int64
}

var _ sessionStoreInterface = &sessRedisStore{}

var sessRedisStoreIns *sessRedisStore
var once = new(sync.Once)

func NewSessRedisStore(redisIns *redis.Client, config configs.RedisConfig) *sessRedisStore {
	once.Do(func() {
		sessRedisStoreIns = &sessRedisStore{
			redisIns:   redisIns,
			prefix:     "sess_",
			expiration: config.Expiration,
		}
	})

	return sessRedisStoreIns
}

func (s *sessRedisStore) SetExpiration(expiration int64) {
	if expiration > 0 {
		s.expiration = expiration
	}
}

func (s *sessRedisStore) IsExist(sessid, key string) (value bool, err error) {

	return s.redisIns.HExists(s.prefix+sessid, key)
}

func (s *sessRedisStore) Put(sessid, key string, value interface{}) (err error) {

	return s.redisIns.HSet(s.prefix+sessid, key, value)
}

func (s *sessRedisStore) Get(value interface{}, sessid, key string) (err error) {

	err = s.redisIns.HGet(&value, s.prefix+sessid, key)

	return
}

func (s *sessRedisStore) Del(sessid, key string) (err error) {

	_, err = s.redisIns.HDel(s.prefix+sessid, key)

	return
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
