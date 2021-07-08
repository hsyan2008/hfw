package proxy

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/hsyan2008/hfw"
)

var ErrAuth = errors.New("proxy auth faild")
var proxyAuthType = "Basic"

var authFunc func(*hfw.HTTPContext, *http.Request, string, string) bool

func SetAuthFunc(f func(*hfw.HTTPContext, *http.Request, string, string) bool) {
	authFunc = f
}

func auth(httpCtx *hfw.HTTPContext, r *http.Request, auth string) (err error) {
	//如果不需要验证
	if authFunc == nil {
		return
	}

	var username, password string

	c := strings.Fields(auth)
	if len(c) == 2 && strings.EqualFold(c[0], proxyAuthType) {
		b, err := base64.StdEncoding.DecodeString(c[1])
		if err != nil {
			return ErrAuth
		}
		f := strings.Split(string(b), ":")
		if len(f) == 2 {
			username = f[0]
			password = f[1]
		}
	}

	if authFunc(httpCtx, r, username, password) {
		return nil
	}

	return ErrAuth
}
