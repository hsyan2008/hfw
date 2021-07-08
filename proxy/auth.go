package proxy

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/hsyan2008/hfw"
)

var ErrAuth = errors.New("proxy auth faild")

var authFunc func(string, string) bool

func SetAuthFunc(f func(string, string) bool) {
	authFunc = f
}

func auth(httpCtx *hfw.HTTPContext, auth string) (err error) {
	//如果不需要验证
	if authFunc == nil {
		return
	}

	if auth == "" {
		return ErrAuth
	}
	c := strings.Fields(auth)
	if len(c) == 2 && strings.EqualFold(c[0], "Basic") {
		b, err := base64.StdEncoding.DecodeString(c[1])
		if err != nil {
			return ErrAuth
		}

		f := strings.Split(string(b), ":")
		if len(f) == 2 && authFunc(f[0], f[1]) {
			return nil
		}
	}

	return ErrAuth
}
