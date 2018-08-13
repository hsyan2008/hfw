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

	"github.com/hsyan2008/hfw2/common"
)

type Response struct {
	Headers    map[string]string `json:"headers"`
	Cookie     string            `json:"cookie"`
	Url        string            `json:"url"`
	FollowUrls []string          `json:"follow_urls"`
	Body       []byte            `json:"body"`
	BodyReader io.ReadCloser     `json:"-"`
}

func (resp *Response) ReadBody() (body []byte, err error) {
	body, err = ioutil.ReadAll(resp.BodyReader)
	if err != nil {
		return
	}

	return bytes.TrimSpace(body), nil
}

var stopRedirect = errors.New("no redirects allowed")

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
	PostFields neturl.Values
	//文件
	PostFiles map[string]string

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
		Timeout:   3 * time.Second,
		KeepAlive: 3600 * time.Second,
	}).Dial,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// DisableKeepAlives:     true,
	TLSHandshakeTimeout:   3 * time.Second,
	ResponseHeaderTimeout: 3 * time.Second,
}

var httpClient = &http.Client{
	// Transport:     tr,
	CheckRedirect: func(_ *http.Request, via []*http.Request) error { return stopRedirect },
	Jar:           nil,
	Timeout:       0,
}

func NewCurl(url string) *Curl {
	return &Curl{
		Headers:    make(map[string]string),
		Options:    make(map[string]bool),
		followUrls: make([]string, 0),
		Url:        url,
		PostFields: neturl.Values{},
		PostFiles:  make(map[string]string),
	}
}

func (curls *Curl) SetStream() {
	curls.isStream = true
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
		curls.SetHeader(k, v)
	}
}
func (curls *Curl) SetHeader(key, val string) {
	curls.Headers[key] = val
}

//秒
func (curls *Curl) SetTimeout(t int) {
	curls.timeout = time.Duration(t) * time.Second
}

//毫秒
func (curls *Curl) SetTimeoutMS(t int) {
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

func (curls *Curl) Request(ctx context.Context) (rs Response, err error) {
	if ctx == nil {
		return rs, errors.New("err context")
	}

	if curls.timeout <= 0 {
		curls.SetTimeout(5)
	}

	var httpRequest *http.Request
	var httpResponse *http.Response

	if "" != curls.PostString || len(curls.PostFields) > 0 || len(curls.PostFiles) > 0 {
		curls.Method = "post"
	}

	if curls.Method == "post" {
		httpRequest, err = curls.postForm()
		if err != nil {
			return
		}
	} else {
		httpRequest, _ = http.NewRequest("GET", curls.Url, nil)
	}

	if curls.Headers != nil {
		for key, value := range curls.Headers {
			httpRequest.Header.Add(key, value)
		}
	}

	if curls.Cookie != "" {
		httpRequest.Header.Add("Cookie", curls.Cookie)
	}
	if curls.Referer != "" {
		httpRequest.Header.Add("Referer", curls.Referer)
	}

	//使用WithTimeout会导致io读取中断
	curls.ctx, curls.cancel = context.WithCancel(ctx)
	httpRequest = httpRequest.WithContext(curls.ctx)

	c := make(chan bool, 1)
	go func() {
		httpResponse, err = httpClient.Do(httpRequest)
		c <- true
	}()

	select {
	case <-time.After(curls.timeout):
		tr.CancelRequest(httpRequest)
		curls.cancel()
		<-c
		err = errors.New("do request time out")
	case <-curls.ctx.Done():
		tr.CancelRequest(httpRequest)
		<-c
		err = ctx.Err()
	case <-c:
	}

	if nil != err {
		//不是重定向里抛出的错误
		urlError, ok := err.(*neturl.Error)
		if !ok || urlError.Err != stopRedirect {
			return rs, err
		}
	}

	return curls.curlResponse(httpResponse)
}

func (curls *Curl) postForm() (httpRequest *http.Request, err error) {

	if len(curls.PostBytes) > 0 {
		b := bytes.NewReader(curls.PostBytes)
		httpRequest, _ = http.NewRequest("POST", curls.Url, b)
		if v, ok := curls.Headers["Content-Type"]; ok {
			httpRequest.Header.Add("Content-Type", v)
		} else {
			httpRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		}
	} else if curls.PostString != "" {
		b := strings.NewReader(curls.PostString)
		httpRequest, _ = http.NewRequest("POST", curls.Url, b)
		if v, ok := curls.Headers["Content-Type"]; ok {
			httpRequest.Header.Add("Content-Type", v)
		} else {
			httpRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		var b = &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(b)

		for key, val := range curls.PostFields {
			for _, v := range val {
				_ = bodyWriter.WriteField(key, v)
			}
		}

		//文件
		for key, val := range curls.PostFiles {
			if !common.IsExist(val) {
				return nil, errors.New(fmt.Sprintf("PostFiles %s => %s not exist", key, val))
			}
			fileWriter, err := bodyWriter.CreateFormFile(key, val)
			if err != nil {
				return nil, err
			}

			fh, err := os.Open(val)
			if err != nil {
				return nil, err
			}
			defer fh.Close()

			_, err = io.Copy(fileWriter, fh)
			if err != nil {
				return nil, err
			}
		}
		//必须在这里，不能defer
		_ = bodyWriter.Close()

		httpRequest, _ = http.NewRequest("POST", curls.Url, b)
		httpRequest.Header.Add("Content-Type", bodyWriter.FormDataContentType())
	}

	delete(curls.Headers, "Content-Type")

	return httpRequest, nil
}

//处理获取的页面
func (curls *Curl) curlResponse(resp *http.Response) (response Response, err error) {
	response.Headers = curls.rcHeader(resp.Header)
	location, _ := resp.Location()
	if nil != location {
		location_url := location.String()
		response.Headers["Location"] = location_url

		//如果不自动重定向，就直接返回
		if curls.Options["redirect"] {
			if curls.RedirectCount < 5 {
				curls.Referer = curls.Url
				curls.RedirectCount++
				curls.followUrls = append(curls.followUrls, curls.Url)
				curls.Url = location_url
				curls.Method = "get"
				curls.PostString = ""
				curls.PostFields = nil
				curls.PostFiles = nil
				curls.Cookie = curls.afterCookie(resp)

				return curls.Request(curls.ctx)
			} else {
				return response, errors.New("重定向次数过多")
			}
		}
	}

	response.Headers["Status"] = resp.Status
	response.Headers["Status-Code"] = strconv.Itoa(resp.StatusCode)
	response.Headers["Proto"] = resp.Proto
	response.Cookie = curls.afterCookie(resp)
	response.Url = curls.Url
	response.FollowUrls = curls.followUrls

	response.BodyReader, err = curls.getReader(resp)
	if !curls.isStream {
		response.Body, err = response.ReadBody()
	}

	return response, err
}

//需要调用方手动关闭
func (curls *Curl) getReader(resp *http.Response) (r io.ReadCloser, err error) {
	//如果出现302或301，已经表示是不自动重定向 或者出现200才读
	if resp.StatusCode == http.StatusOK || resp.StatusCode == 299 {
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			return gzip.NewReader(resp.Body)
		} else if strings.Contains(resp.Header.Get("Content-Encoding"), "deflate") {
			return flate.NewReader(resp.Body), nil
		} else {
			return resp.Body, nil
		}
	}

	return nil, nil
}

//返回结果的时候，转换cookie为字符串
func (curls *Curl) afterCookie(resp *http.Response) string {
	//去掉重复
	rs_tmp := make(map[string]string)

	//先处理传进来的cookie
	if curls.Cookie != "" {
		tmp := strings.Split(curls.Cookie, "; ")
		for _, v := range tmp {
			tmp_one := strings.SplitN(v, "=", 2)
			rs_tmp[tmp_one[0]] = tmp_one[1]
		}
	}

	//处理新cookie
	for _, v := range resp.Cookies() {
		//过期
		if v.Value == "EXPIRED" {
			delete(rs_tmp, v.Name)
			continue
		}
		rs_tmp[v.Name] = v.Value
	}
	//用于join
	rs := make([]string, len(rs_tmp))
	i := 0
	for k, v := range rs_tmp {
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
