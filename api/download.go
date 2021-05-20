package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/curl"
	"github.com/hsyan2008/hfw/encoding"
)

func Download(httpCtx *hfw.HTTPContext, url string, p interface{}) (content []byte, err error) {
	if httpCtx == nil || httpCtx.Ctx == nil {
		return nil, common.NewRespErr(500, "nil httpCtx")
	}
	var c *curl.Curl
	if p != nil {
		c = curl.NewPost(httpCtx.Ctx, url)
		c.PostBytes, err = encoding.JSON.Marshal(p)
		if err != nil {
			return nil, common.NewRespErr(500, err)
		}
	} else {
		c = curl.NewGet(httpCtx.Ctx, url)
	}
	c.Headers.Set("Content-Type", "application/json")
	c.SetTimeout(5)

	if logger.Level() == logger.DEBUG {
		httpCtx.Debugf("Call: %s %#v start", url, string(c.PostBytes))
		defer func(t time.Time) {
			httpCtx.Debugf("Call: %s CostTime: %v", url, time.Since(t))
		}(time.Now())
	}

	var rs *curl.Response
FOR:
	for i := 0; i < 3; i++ {
		select {
		case <-httpCtx.Ctx.Done():
			return nil, common.NewRespErr(500, "context cancel")
		default:
			c.SetContext(httpCtx.Ctx)
			rs, err = c.Request()
			if err != nil {
				httpCtx.Warnf("Url: [%s] %s", c.Url, err.Error())
				// if err == curl.ErrRequestTimeout {
				// 	break FOR
				// }
				continue FOR
			}
			defer rs.Close()

			break FOR
		}
	}

	if err != nil {
		return nil, common.NewRespErr(500, err)
	}

	if rs.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Call %s get http status code: %d status: %s",
			c.Url, rs.StatusCode, rs.Status)
	}

	return ioutil.ReadAll(rs.Body)
}
