package session

import (
	"net/http"
	"sync"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
)

type sessionStoreInterface interface {
	SetExpiration(int64)
	Put(string, string, interface{}) error
	Get(interface{}, string, string) error
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
	expiration int64
}

var sessPool = sync.Pool{
	New: func() interface{} {
		return new(Session)
	},
}

func NewSession(request *http.Request, store sessionStoreInterface, config configs.SessionConfig) (s *Session) {
	// s := sessPool.Get().(*Session)
	if config.CookieName == "" {
		config.CookieName = "sess_name"
	}

	s = new(Session)
	s.newid = common.Uuid()
	s.cookieName = config.CookieName
	s.reName = config.ReName
	s.expiration = config.Expiration

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

	s.store = store
	s.store.SetExpiration(s.expiration)

	return
}

func (s *Session) Close(request *http.Request, response http.ResponseWriter) {
	if !s.isNew && s.reName {
		err := s.Rename()
		if err == nil {
			s.id = s.newid
			s.isNew = true
		}
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

func (s *Session) IsExist(k string) bool {
	v, _ := s.store.IsExist(s.id, k)
	return v
}

func (s *Session) Set(k string, v interface{}) {
	_ = s.store.Put(s.id, k, v)
}

func (s *Session) Get(value interface{}, k string) {
	s.store.Get(value, s.id, k)
}

func (s *Session) Del(k string) {
	_ = s.store.Del(s.id, k)
}

func (s *Session) Destroy() {
	_ = s.store.Destroy(s.id)
}

func (s *Session) Rename() (err error) {
	if s.id != s.newid {
		return s.store.Rename(s.id, s.newid)
	}

	return
}
