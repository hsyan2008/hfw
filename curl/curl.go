package curl

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hsyan2008/hfw/common"
)

type Response struct {
	Headers    map[string]string `json:"headers"`
	Cookie     string            `json:"cookie"`
	Url        string            `json:"url"`
	FollowUrls []string          `json:"follow_urls"`
	Body       []byte            `json:"body"`
	BodyReader io.ReadCloser     `json:"-"`

	IsStream bool `json:"-"`

	ctx    context.Context
	cancel context.CancelFunc
}

func (resp *Response) ReadBody() (body []byte, err error) {
	body, err = ioutil.ReadAll(resp.BodyReader)
	if err != nil {
		return
	}

	return bytes.TrimSpace(body), nil
}

func (resp *Response) Close() {
	if resp.BodyReader != nil {
		resp.BodyReader.Close()
	}
	if resp.cancel != nil {
		resp.cancel()
	}
}

var ErrStopRedirect = errors.New("no redirects allowed")
var ErrRequestTimeout = errors.New("do request time out")

type Curl struct {
	Url, Method, Cookie, Referer string

	Headers map[string]string
	Options map[string]bool

	RedirectCount int

	//[]byte格式
	PostBytes []byte
	//string格式
	PostString string
	//key=>value格式
	PostFields       neturl.Values
	PostFieldReaders map[string]io.Reader
	//文件，key是字段名，val是文件路径
	PostFiles neturl.Values
	//流
	PostReader io.Reader

	timeout time.Duration

	followUrls []string
	//是否要把BodyReader读取到Body里
	isStream bool

	ctx    context.Context
	cancel context.CancelFunc
}

var tr = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	Dial: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 3600 * time.Second,
	}).Dial,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// DisableKeepAlives:     true,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
}

var httpClient = &http.Client{
	Transport:     tr,
	CheckRedirect: func(_ *http.Request, via []*http.Request) error { return ErrStopRedirect },
	Jar:           nil,
	Timeout:       0,
}

func NewCurl(url string) *Curl {
	return &Curl{
		Headers:          make(map[string]string),
		Options:          make(map[string]bool),
		followUrls:       make([]string, 0),
		Url:              url,
		PostFields:       neturl.Values{},
		PostFieldReaders: make(map[string]io.Reader),
		PostFiles:        neturl.Values{},
	}
}

func (curls *Curl) SetStream() {
	curls.isStream = true
}

func (curls *Curl) SetMethod(method string) error {
	curls.Method = strings.ToUpper(method)
	switch curls.Method {
	case "OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE", "TRACE", "CONNECT":
		return nil
	default:
		return fmt.Errorf("net/http: invalid method %q", method)
	}
}

func (curls *Curl) SetHeaders(headers map[string]string) {
	curls.Headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	curls.Headers["Accept-Encoding"] = "gzip, deflate"
	curls.Headers["Accept-Language"] = "zh-cn,zh;q=0.8,en-us;q=0.5,en;q=0.3"
	// curls.Headers["Connection"] = "close"
	curls.Headers["Connection"] = "keep-alive"
	curls.Headers["User-Agent"] = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.146 Safari/537.36"

	//使用这个header是因为避免100的状态码
	curls.Headers["Expect"] = ""

	for k, v := range headers {
		curls.SetHeader(http.CanonicalHeaderKey(k), v)
	}
}

func (curls *Curl) SetContext(ctx context.Context) {
	if curls.ctx == nil || curls.ctx.Err() != nil {
		curls.ctx, curls.cancel = context.WithCancel(ctx)
	}
}

func (curls *Curl) SetHeader(key, val string) {
	curls.Headers[key] = val
}

//秒
func (curls *Curl) SetTimeout(t int) {
	curls.SetContext(context.Background())
	curls.timeout = time.Duration(t) * time.Second
}

//毫秒
func (curls *Curl) SetTimeoutMS(t int) {
	curls.SetContext(context.Background())
	curls.timeout = time.Duration(t) * time.Millisecond
}

func (curls *Curl) SetOptions(options map[string]bool) {
	for k, v := range options {
		curls.SetOption(k, v)
	}
}
func (curls *Curl) SetOption(key string, val bool) {
	curls.Options[key] = val
}

//Request 参数不需要传，请使用SetContext
func (curls *Curl) Request(ctxs ...context.Context) (rs *Response, err error) {
	if len(ctxs) > 0 && ctxs[0] != nil {
		curls.SetContext(ctxs[0])
	}

	if curls.timeout <= 0 {
		curls.SetTimeout(5)
	}

	var httpRequest *http.Request
	var httpResponse *http.Response

	httpRequest, err = curls.CreateRequest()
	if err != nil {
		return
	}

	//使用WithTimeout会导致io读取中断
	httpRequest = httpRequest.WithContext(curls.ctx)

	c := make(chan struct{}, 1)
	go func() {
		httpResponse, err = httpClient.Do(httpRequest)
		c <- struct{}{}
	}()

	select {
	case <-time.After(curls.timeout):
		<-c
		err = ErrRequestTimeout
	case <-curls.ctx.Done():
		<-c
		err = curls.ctx.Err()
	case <-c:
		//会影响读取body
		// defer curls.cancel()
	}

	if nil != err {
		//不是重定向里抛出的错误
		urlError, ok := err.(*neturl.Error)
		if !ok || urlError.Err != ErrStopRedirect {
			curls.cancel()
			return nil, err
		}
	}

	return curls.curlResponse(httpResponse)
}

func (curls *Curl) CreateRequest() (httpRequest *http.Request, err error) {
	if curls.PostReader != nil || len(curls.PostBytes) > 0 ||
		"" != curls.PostString || len(curls.PostFields) > 0 || len(curls.PostFieldReaders) > 0 || len(curls.PostFiles) > 0 {
		httpRequest, err = curls.createPostRequest()
	} else {
		if len(curls.Method) == 0 {
			httpRequest, err = http.NewRequest("GET", curls.Url, nil)
		} else {
			httpRequest, err = http.NewRequest(curls.Method, curls.Url, nil)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("CreateRequest failed: %s %#v", err.Error(), err)
	}

	if curls.Headers != nil {
		for key, value := range curls.Headers {
			httpRequest.Header.Add(key, value)
		}
	}

	if len(curls.Cookie) > 0 {
		httpRequest.Header.Add("Cookie", curls.Cookie)
	}
	if len(curls.Referer) > 0 {
		httpRequest.Header.Add("Referer", curls.Referer)
	}

	return
}

func (curls *Curl) createPostRequest() (httpRequest *http.Request, err error) {
	var hasSetHeader bool
	if len(curls.Method) == 0 {
		curls.Method = "POST"
	} else {
		curls.Method = strings.ToUpper(curls.Method)
	}
	if curls.PostReader != nil {
		httpRequest, err = http.NewRequest(curls.Method, curls.Url, curls.PostReader)
	} else if len(curls.PostBytes) > 0 {
		b := bytes.NewReader(curls.PostBytes)
		httpRequest, err = http.NewRequest(curls.Method, curls.Url, b)
	} else if len(curls.PostString) > 0 {
		b := strings.NewReader(curls.PostString)
		httpRequest, err = http.NewRequest(curls.Method, curls.Url, b)
	} else if len(curls.PostFields) > 0 || len(curls.PostFieldReaders) > 0 || len(curls.PostFiles) > 0 {
		var b = &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(b)

		for key, val := range curls.PostFields {
			for _, v := range val {
				_ = bodyWriter.WriteField(key, v)
			}
		}
		for key, val := range curls.PostFieldReaders {
			fileWriter, err := bodyWriter.CreateFormField(key)
			_, err = io.Copy(fileWriter, val)
			if err != nil {
				return nil, err
			}
		}

		//文件
		for key, val := range curls.PostFiles {
			for _, v := range val {
				if !common.IsExist(v) {
					return nil, errors.New(fmt.Sprintf("PostFiles %s => %s not exist", key, v))
				}
				fileWriter, err := bodyWriter.CreateFormFile(key, v)
				if err != nil {
					return nil, err
				}

				fh, err := os.Open(v)
				if err != nil {
					return nil, err
				}

				_, err = io.Copy(fileWriter, fh)
				fh.Close()
				if err != nil {
					return nil, err
				}
			}
		}
		//必须在这里，不能defer
		err = bodyWriter.Close()
		if err != nil {
			return
		}

		httpRequest, err = http.NewRequest(curls.Method, curls.Url, b)
		if err != nil {
			return
		}
		httpRequest.Header.Set("Content-Type", bodyWriter.FormDataContentType())
		hasSetHeader = true
	} else {
		httpRequest, err = http.NewRequest(curls.Method, curls.Url, nil)
	}

	if err != nil {
		return
	}

	if !hasSetHeader {
		if v, ok := curls.Headers["Content-Type"]; ok {
			httpRequest.Header.Set("Content-Type", v)
		}
	}

	delete(curls.Headers, "Content-Type")

	return httpRequest, nil
}

//处理获取的页面
func (curls *Curl) curlResponse(resp *http.Response) (response *Response, err error) {
	defer func() {
		if err != nil {
			resp.Body.Close()
		}
	}()
	response = new(Response)
	response.ctx = curls.ctx
	response.cancel = curls.cancel
	response.Headers = curls.rcHeader(resp.Header)
	response.IsStream = curls.isStream
	location, _ := resp.Location()
	if nil != location {
		locationUrl := location.String()
		response.Headers["Location"] = locationUrl

		//如果不自动重定向，就直接返回
		if curls.Options["redirect"] {
			if curls.RedirectCount < 5 {
				curls.Referer = curls.Url
				curls.RedirectCount++
				curls.followUrls = append(curls.followUrls, curls.Url)
				curls.Url = locationUrl
				curls.Method = "GET"
				curls.PostBytes = nil
				curls.PostString = ""
				curls.PostFields = nil
				curls.PostFieldReaders = nil
				curls.PostFiles = nil
				curls.PostReader = nil
				curls.Cookie = curls.afterCookie(resp)
				resp.Body.Close()

				response, err = curls.Request()
				return
			} else {
				err = errors.New("too much redirect")
				return
			}
		}
	}

	response.Headers["Status"] = resp.Status
	response.Headers["Status-Code"] = strconv.Itoa(resp.StatusCode)
	response.Headers["Proto"] = resp.Proto
	response.Cookie = curls.afterCookie(resp)
	response.Url = curls.Url
	response.FollowUrls = curls.followUrls

	//目前只有200才需要读取body
	if resp.StatusCode == http.StatusOK {
		response.BodyReader, err = curls.getReader(resp)
		if err != nil {
			return response, err
		}
		if !curls.isStream {
			response.Body, err = response.ReadBody()
			if err != nil {
				return response, err
			}
			resp.Body.Close()
		}
	}

	return response, err
}

//需要调用方手动关闭
func (curls *Curl) getReader(resp *http.Response) (r io.ReadCloser, err error) {
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		return gzip.NewReader(resp.Body)
	} else if strings.Contains(resp.Header.Get("Content-Encoding"), "deflate") {
		return flate.NewReader(resp.Body), nil
	}

	return resp.Body, nil
}

//返回结果的时候，转换cookie为字符串
func (curls *Curl) afterCookie(resp *http.Response) string {
	//去掉重复
	rsTmp := make(map[string]string)

	//先处理传进来的cookie
	if curls.Cookie != "" {
		tmp := strings.Split(curls.Cookie, "; ")
		for _, v := range tmp {
			tmpOne := strings.SplitN(v, "=", 2)
			rsTmp[tmpOne[0]] = tmpOne[1]
		}
	}

	//处理新cookie
	for _, v := range resp.Cookies() {
		//过期
		if v.Value == "EXPIRED" {
			delete(rsTmp, v.Name)
			continue
		}
		rsTmp[v.Name] = v.Value
	}
	//用于join
	rs := make([]string, len(rsTmp))
	i := 0
	for k, v := range rsTmp {
		rs[i] = k + "=" + v
		i++
	}

	sort.Strings(rs)

	return strings.TrimSpace(strings.Join(rs, "; "))
}

//整理header
func (curls *Curl) rcHeader(header map[string][]string) map[string]string {
	headers := make(map[string]string, len(header))
	for k, v := range header {
		headers[k] = strings.Join(v, "\n")
	}

	return headers
}
