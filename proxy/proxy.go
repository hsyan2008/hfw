package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"acln.ro/zerocopy"
	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
)

func Enable() {
	hfw.RegisterServeHTTPCook(IsProxy, ProxyServe)
}

func IsProxy(r *http.Request) bool {
	return r.Header.Get("Proxy-Connection") != "" ||
		r.URL.Host != ""
}

const retryTime = 3

func ProxyServe(w http.ResponseWriter, r *http.Request) {
	httpCtx := hfw.NewHTTPContext()
	defer httpCtx.Cancel()
	httpCtx.AppendPrefix("PROXY")
	httpCtx.AppendPrefix(r.Host)

	httpCtx.Mixf("From:%s", r.RemoteAddr)
	defer func(now time.Time) {
		httpCtx.Mixf("CostTime:%s", time.Since(now))
	}(time.Now())

	httpCtx.ResponseWriter = w
	httpCtx.Request = r

	httpCtx.Request.Header.Del("Proxy-Connection")
	//否则远程连接不会关闭，导致Copy卡住
	httpCtx.Request.Header.Set("Connection", "close")
	httpCtx.Request.Close = true

	//获取底层连接
	httpCtx.Debug("hijacker")
	conn, bufrw, err := httpCtx.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()

	defer func() {
		if err := recover(); err != nil {
			httpCtx.Error(err, string(common.GetStack()))
			httpError(httpCtx, bufrw, http.StatusInternalServerError, "panic")
		}
	}()

	//以下重试
	i := 0
	for i < retryTime {
		i++
		select {
		case <-httpCtx.Ctx.Done():
			return
		default:
			//发起连接
			httpCtx.Debug("connect service")
			serviceConn, err := dial(httpCtx, httpCtx.Request.Host)
			if err != nil {
				httpCtx.Warn(err)
				if i == retryTime-1 {
					httpError(httpCtx, bufrw, http.StatusBadGateway, "dial service faild")
					return
				}
				continue
			}
			defer serviceConn.Close()

			httpCtx.Debug("write data")
			if httpCtx.Request.Method == http.MethodConnect {
				_, err = io.WriteString(conn, "HTTP/1.0 200 Connection Established\r\n\r\n")
			} else {
				err = httpCtx.Request.Write(serviceConn)
			}
			if err != nil {
				httpCtx.Warn(err)
				httpError(httpCtx, bufrw, http.StatusBadGateway, "send data to service faild")
				return
			}

			httpCtx.Debug("multi copy data")
			go copy(httpCtx, conn, serviceConn)
			go copy(httpCtx, serviceConn, conn)
			select {
			case <-httpCtx.Ctx.Done():
				return
			}

			return
		}
	}
}

var dialer = new(net.Dialer)

func dial(httpCtx *hfw.HTTPContext, addr string) (con net.Conn, err error) {
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") ||
		!strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:80", addr)
	}
	con, err = dialer.DialContext(httpCtx.Ctx, "tcp", addr)

	return
}

func copy(httpCtx *hfw.HTTPContext, des, src net.Conn) {
	defer func() {
		httpCtx.Cancel()
	}()

	zerocopy.Transfer(des, src)
}

func httpError(httpCtx *hfw.HTTPContext, bufrw *bufio.ReadWriter, code int, msg string) {
	bufrw.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\nTrace-Id: %s\r\n\r\n%s\r\n", code, http.StatusText(code), httpCtx.GetTraceID(), msg))
	bufrw.Flush()
}
