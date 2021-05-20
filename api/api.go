package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
	"github.com/hsyan2008/hfw/curl"
	"github.com/hsyan2008/hfw/encoding"
	serviceDiscovery "github.com/hsyan2008/hfw/service_discovery"
)

//内部第三方接口返回
type Response struct {
	ErrNo   int64       `json:"err_no"`
	ErrMsg  string      `json:"err_msg"`
	Results interface{} `json:"results"`
}

//RequestUnmarshal 从httpCtx里解析数据到params
func RequestUnmarshal(httpCtx *hfw.HTTPContext, params interface{}) (err error) {
	if httpCtx == nil || httpCtx.Request == nil {
		return common.NewRespErr(500, "nil httpCtx")
	}
	if httpCtx.Request.Method != "POST" {
		return common.NewRespErr(400, "request method must be POST")
	}

	if logger.Level() == logger.DEBUG {
		data, e := ioutil.ReadAll(httpCtx.Request.Body)
		if e != nil {
			return common.NewRespErr(400, e)
		}
		httpCtx.Debugf("Request Raw Params is: %s", string(data))
		err = encoding.JSON.Unmarshal(data, &params)
	} else {
		err = encoding.JSONIO.Unmarshal(httpCtx.Request.Body, &params)
	}

	httpCtx.Debugf("Request cmp Params is: %#v", params)
	if err != nil {
		return common.NewRespErr(400, err)
	}

	return nil
}

//用于调用内部的其他标准http服务
//标准的http服务是指response里包含err_no、err_msg和results
func StdCallByConsul(httpCtx *hfw.HTTPContext, serviceName, uri string, p interface{}, results interface{}, opts ...CallOption) (err error) {
	resolverAddresses := configs.Config.Server.ResolverAddresses
	if len(resolverAddresses) == 0 {
		return errors.New("nil resolverAddresses")
	}
	cr, err := serviceDiscovery.NewConsulResolver(serviceName, resolverAddresses[0], serviceDiscovery.RobinPolicy, "")
	if err != nil {
		return
	}

	return StdCall(httpCtx, cr, uri, p, results, opts...)
}

//用于调用内部的其他标准http服务
//标准的http服务是指response里包含err_no、err_msg和results
func StdCall(httpCtx *hfw.HTTPContext, addressParams interface{}, uri string, p interface{}, results interface{}, opts ...CallOption) (err error) {

	resp := &Response{}
	resp.Results = results

	err = Call(httpCtx, map[string]string{}, addressParams, uri, p, resp, opts...)
	if err != nil {
		return err
	}

	if resp.ErrNo != 0 {
		return common.NewRespErr(resp.ErrNo, resp.ErrMsg)
	}

	return
}

//用于任意返回json数据的http服务
func Call(httpCtxIn *hfw.HTTPContext, header map[string]string, addressParams interface{}, uri string, p interface{}, resp interface{}, opts ...CallOption) (err error) {
	if httpCtxIn == nil || httpCtxIn.Ctx == nil {
		return common.NewRespErr(500, "nil httpCtx")
	}
	httpCtx := hfw.NewHTTPContextWithCtx(httpCtxIn)
	defer httpCtx.Cancel()

	var (
		cr        *serviceDiscovery.ConsulResolver
		addresses []string
	)
	switch i := addressParams.(type) {
	case *serviceDiscovery.ConsulResolver:
		cr = i
	case []string:
		addresses = i
	default:
		return errors.New("error addressParams type")
	}

	c := curl.NewPost(httpCtx.Ctx, "")
	c.Headers.Set("Content-Type", "application/json")
	if appId := httpCtx.Ctx.Value("app_id"); appId != nil {
		c.Headers.Set("AppID", appId.(string))
	}
	for k, v := range header {
		c.Headers.Set(k, v)
	}
	c.SetTimeout(30)
	c.PostBytes, err = encoding.JSON.Marshal(p)
	if err != nil {
		return common.NewRespErr(500, err)
	}
	for _, f := range opts {
		err = f.Do(c)
		if err != nil {
			return common.NewRespErr(500, err)
		}
	}

	if cr == nil {
		httpCtx.Debugf("Call:%v %s %s start", addresses, uri, string(c.PostBytes))
	} else {
		httpCtx.Debugf("Call:%v %s %s start", cr.Addresses(), uri, string(c.PostBytes))
	}
	defer func(t time.Time) {
		if err == nil {
			httpCtx.Infof("Call:%s %#v CostTime:%v", uri, resp, time.Since(t))
		} else {
			httpCtx.Warnf("Call:%s Req:%s Err:%s CostTime:%v", string(c.PostBytes), err, uri, time.Since(t))
		}
	}(time.Now())

	var rs *curl.Response
FOR:
	for i := 0; i < 3; i++ {
		select {
		case <-httpCtx.Ctx.Done():
			return httpCtx.Ctx.Err()
		default:
			if cr != nil {
				addr, err := cr.GetAddress()
				if err != nil {
					return common.NewRespErr(500, err)
				}
				addresses = []string{addr}
			}
			c.Url, err = getApiUrl(addresses, uri)
			if err != nil {
				return common.NewRespErr(500, err)
			}
			err = func() (err error) {
				tmpHttpCtx := hfw.NewHTTPContextWithCtx(httpCtx)
				defer tmpHttpCtx.Cancel()
				defer func(t time.Time) {
					tmpHttpCtx.Infof("Call:%s TryTime:%d CostTime:%s", c.Url, i, time.Since(t))
				}(time.Now())
				c.Headers.Set("Trace-Id", tmpHttpCtx.GetTraceID())
				c.SetContext(httpCtx.Ctx)
				rs, err = c.Request()
				if err != nil {
					tmpHttpCtx.Warnf("Url:%s %s", c.Url, err.Error())
					return
				}
				return
			}()
			if err != nil {
				continue FOR
			}
			defer rs.Close()

			break FOR
		}
	}

	if err != nil {
		return common.NewRespErr(500, err)
	}

	if rs.StatusCode != http.StatusOK {
		err = fmt.Errorf("Call:%s StatusCode:%d", c.Url, rs.StatusCode)
		return common.NewRespErr(500, err)
	}
	if logger.Level() == logger.DEBUG {
		body, err := ioutil.ReadAll(rs.Body)
		if err != nil {
			return common.NewRespErr(500, err)
		}
		httpCtx.Debugf("Call:%v Get:%s", c.Url, string(body))
		err = encoding.JSON.Unmarshal(body, &resp)
	} else {
		err = encoding.JSONIO.Unmarshal(rs.Body, &resp)
	}

	if err != nil {
		return common.NewRespErr(500, err)
	}

	return nil
}

func getApiUrl(addresses []string, uri string) (string, error) {
	n := len(addresses)
	var domain string
	if n == 0 {
		return "", errors.New("nil addresses for call " + uri)
	} else if n == 1 {
		domain = addresses[0]
	} else {
		domain = addresses[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(n)]
	}
	if strings.HasPrefix(domain, "http") == false {
		domain = "http://" + domain
	}
	u, err := neturl.ParseRequestURI(domain)
	if err != nil || len(u.Scheme) == 0 {
		return "", fmt.Errorf("error url: [%s]", domain)
	}

	if len(uri) > 0 {
		u, err = u.Parse(uri)
		if err != nil {
			return "", err
		}
	}

	return u.String(), nil
}

type CallOption interface {
	Do(*curl.Curl) error
}

type TimeOutCallOption struct {
	timeout int
}

func NewTimeOutCallOption(timeout int) TimeOutCallOption {
	return TimeOutCallOption{timeout}
}

func (t TimeOutCallOption) Do(c *curl.Curl) error {
	c.SetTimeout(t.timeout)
	return nil
}

type AddHeadersCallOption struct {
	headers map[string]string
}

func NewAddHeadersCallOption(headers map[string]string) AddHeadersCallOption {
	return AddHeadersCallOption{headers}
}

func (t AddHeadersCallOption) Do(c *curl.Curl) error {
	for k, v := range t.headers {
		c.Headers.Add(k, v)
	}
	return nil
}

type DelHeadersCallOption struct {
	keys []string
}

func NewDelHeadersCallOption(keys ...string) DelHeadersCallOption {
	return DelHeadersCallOption{keys}
}

func (t DelHeadersCallOption) Do(c *curl.Curl) error {
	for _, v := range t.keys {
		c.Headers.Del(v)
	}
	return nil
}
