package hfw

import (
	"errors"
	"sync"

	"github.com/pborman/uuid"
)

type sessionStoreInterface interface {
	Put(string, string, interface{}) error
	Get(string, string) (interface{}, error)
	IsExist(string, string) (bool, error)
	Del(string, string) error
	Destroy(string) error
	Rename(string, string) error
}

type Session struct {
	id    string
	newid string
	store sessionStoreInterface
}

var sessPool = sync.Pool{
	New: func() interface{} {
		return new(Session)
	},
}

func NewSession(sessId string) (s *Session, err error) {
	// s := sessPool.Get().(*Session)
	s = new(Session)

	s.newid = uuid.New()

	s.id = sessId

	switch Config.Session.CacheType {
	case "redis":
		s.store, err = NewSessRedisStore()
	default:
		err = errors.New("session config error")
	}

	return
}

func (s *Session) IsExist(k string) bool {
	v, _ := s.store.IsExist(s.id, k)
	return v
}

func (s *Session) Set(k string, v interface{}) {
	_ = s.store.Put(s.id, k, v)
}

func (s *Session) Get(k string) interface{} {
	v, _ := s.store.Get(s.id, k)
	return v
}

func (s *Session) Del(k string) {
	_ = s.store.Del(s.id, k)
}

func (s *Session) Destroy() {
	_ = s.store.Destroy(s.id)
}

func (s *Session) Rename() {
	_ = s.store.Rename(s.id, s.newid)
}
