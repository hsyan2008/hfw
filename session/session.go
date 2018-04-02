package session

import (
	"errors"
	"net/http"
	"sync"

	"github.com/hsyan2008/hfw2/configs"
	"github.com/hsyan2008/hfw2/redis"
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
	id         string
	newid      string
	isNew      bool
	store      sessionStoreInterface
	cookieName string
	reName     bool
}

var sessPool = sync.Pool{
	New: func() interface{} {
		return new(Session)
	},
}

func NewSession(request *http.Request, redisIns *redis.Redis, config configs.AllConfig) (s *Session, err error) {
	// s := sessPool.Get().(*Session)
	if config.Session.CookieName == "" {
		return
	}

	s = new(Session)
	s.newid = uuid.New()
	s.cookieName = config.Session.CookieName
	s.reName = config.Session.ReName

	var sessId string
	cookie, err := request.Cookie(s.cookieName)
	if err == nil {
		sessId = cookie.Value
	}
	if sessId == "" {
		s.id = s.newid
		s.isNew = true
	} else {
		s.id = sessId
	}

	switch config.Session.CacheType {
	case "redis":
		s.store, err = NewSessRedisStore(redisIns, config)
	default:
		err = errors.New("session config error")
	}

	return
}

func (s *Session) Close(request *http.Request, response http.ResponseWriter) {
	if s.cookieName != "" {
		if !s.isNew && s.reName {
			s.Rename()
			s.id = s.newid
			s.isNew = true
		}
		if s.isNew {
			cookie := &http.Cookie{
				Name:     s.cookieName,
				Value:    s.id,
				Path:     "/",
				HttpOnly: true,
				Secure:   request.URL.Scheme == "https",
			}
			http.SetCookie(response, cookie)
		}
	}
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
	if s.id != s.newid {
		_ = s.store.Rename(s.id, s.newid)
	}
}
