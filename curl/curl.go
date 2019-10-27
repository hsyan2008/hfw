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
	"strings"
	"sync"
	"time"
)

type Response struct {
	*http.Response
	Body io.ReadCloser

	cancel context.CancelFunc
}

func (response *Response) wrap(curls *Curl) (err error) {
	response.Body, err = response.getReader()
	if err != nil {
		io.Copy(ioutil.Discard, response.Response.Body)
		response.Response.Body.Close()
	}

	response.cancel = curls.cancel

	return
}

func (response *Response) getReader() (r io.ReadCloser, err error) {
	if strings.Contains(response.Response.Header.Get("Content-Encoding"), "gzip") {
		return gzip.NewReader(response.Response.Body)
	} else if strings.Contains(response.Response.Header.Get("Content-Encoding"), "deflate") {
		return flate.NewReader(response.Response.Body), nil
	}

	return response.Response.Body, nil
}

func (response *Response) Close() {
	response.cancel()
	if response.Body != nil {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}
}

var ErrStopRedirect = errors.New("no redirects allowed")
var ErrRequestTimeout = errors.New("do request time out")

type Curl struct {
	Url, method string

	cookies []*http.Cookie

	Headers http.Header

	autoRedirect bool

	//流
	PostReader io.Reader
	//[]byte格式
	PostBytes []byte
	//string格式
	PostString string
	//key=>value格式
	PostFields       neturl.Values
	PostFieldReaders map[string]io.Reader
	//文件，key是字段名，val是文件路径
	PostFiles neturl.Values

	timeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	proxyURL string
}

func New(ctx context.Context, method string, url string) (curls *Curl) {
	curls = &Curl{
		Url:              url,
		Headers:          http.Header{},
		PostFields:       neturl.Values{},
		PostFiles:        neturl.Values{},
		PostFieldReaders: make(map[string]io.Reader),
	}
	curls.SetContext(ctx)

	curls.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	curls.Headers.Set("Accept-Encoding", "gzip, deflate")
	curls.Headers.Set("Accept-Language", "zh-cn,zh;q=0.8,en-us;q=0.5,en;q=0.3")
	// curls.Headers.Set("Connection"] = "close"
	curls.Headers.Set("Connection", "keep-alive")
	curls.Headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.146 Safari/537.36")

	//使用这个header是因为避免100的状态码
	curls.Headers.Set("Expect", "")

	curls.method = strings.ToUpper(method)

	return
}

func NewGet(ctx context.Context, url string) *Curl {
	return New(ctx, http.MethodGet, url)
}

func NewPost(ctx context.Context, url string) *Curl {
	return New(ctx, http.MethodPost, url)
}

func (curls *Curl) SetContext(ctx context.Context) {
	curls.ctx, curls.cancel = context.WithCancel(ctx)
}

func (curls *Curl) SetAutoRedirect() {
	curls.autoRedirect = true
}

func (curls *Curl) SetHeaders(headers map[string]string) {
	for k, v := range headers {
		curls.Headers.Set(k, v)
	}
}

func (curls *Curl) SetCookies(cookies []*http.Cookie) {
	curls.cookies = cookies
}

//秒
func (curls *Curl) SetTimeout(t int) {
	curls.SetTimeoutMS(t * 1000)
}

//毫秒
func (curls *Curl) SetTimeoutMS(t int) {
	curls.timeout = time.Duration(t) * time.Millisecond
}

func (curls *Curl) SetProxy(proxyURL string) {
	curls.proxyURL = proxyURL
}

//以字节流的方式
func (curls *Curl) SetPostReader(r io.Reader) {
	curls.PostReader = r
}

//以字节的方式
func (curls *Curl) SetPostBytes(b []byte) {
	curls.PostBytes = b
}

//以字符串的方式
func (curls *Curl) SetPostString(s string) {
	curls.PostString = s
}

//以key=>value方式，以下三个都是key=>value方式，可以共同使用
func (curls *Curl) SetPostField(key, value string) {
	curls.PostFields.Add(key, value)
}

//以流的方式
func (curls *Curl) SetPostFieldReader(key string, r io.Reader) {
	curls.PostFieldReaders[key] = r
}

//以上传文件的方式
func (curls *Curl) SetPostFile(key, path string) {
	curls.PostFiles.Add(key, path)
}

func (curls *Curl) Request() (rs *Response, err error) {

	if curls.timeout <= 0 {
		curls.SetTimeout(5)
	}
	rs = new(Response)

	httpRequest, err := curls.CreateRequest()
	if err != nil {
		return
	}

	httpRequest = httpRequest.WithContext(curls.ctx)

	httpClient, err := curls.getHttpClient()
	if err != nil {
		return
	}

	c := make(chan struct{}, 1)
	go func() {
		rs.Response, err = httpClient.Do(httpRequest)
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
	} else {
		err = rs.wrap(curls)
	}

	return
}

func (curls *Curl) CreateRequest() (httpRequest *http.Request, err error) {
	if curls.PostReader != nil || len(curls.PostBytes) > 0 ||
		curls.PostString != "" || len(curls.PostFields) > 0 ||
		len(curls.PostFieldReaders) > 0 || len(curls.PostFiles) > 0 {
		httpRequest, err = curls.createPostRequest()
	} else {
		httpRequest, err = http.NewRequest(curls.method, curls.Url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("CreateRequest failed: %s %#v", err.Error(), err)
	}

	if curls.Headers != nil {
		httpRequest.Header = curls.Headers
	}

	for _, cookie := range curls.cookies {
		httpRequest.AddCookie(cookie)
	}

	return
}

func (curls *Curl) createPostRequest() (httpRequest *http.Request, err error) {
	if curls.PostReader != nil {
		// httpRequest, err = http.NewRequest(curls.method, curls.Url, curls.PostReader)
	} else if len(curls.PostBytes) > 0 {
		curls.PostReader = bytes.NewReader(curls.PostBytes)
	} else if len(curls.PostString) > 0 {
		curls.PostReader = strings.NewReader(curls.PostString)
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
		curls.Headers.Set("Content-Type", bodyWriter.FormDataContentType())
		curls.PostReader = b
	}

	return http.NewRequest(curls.method, curls.Url, curls.PostReader)
}

var clientMap = new(sync.Map)

func (curls *Curl) getHttpClient() (hc *http.Client, err error) {

	key := fmt.Sprintf("%s||%t", curls.proxyURL, curls.autoRedirect)

	if i, ok := clientMap.Load(key); ok {
		return i.(*http.Client), nil
	}

	proxy := http.ProxyFromEnvironment
	urlParse, err := neturl.Parse(curls.proxyURL)
	if err != nil {
		return nil, err
	}
	if urlParse != nil && urlParse.Host != "" {
		proxy = http.ProxyURL(urlParse)
	}

	hc = &http.Client{
		Transport: &http.Transport{
			Proxy: proxy,
			Dial: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 3600 * time.Second,
			}).Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// DisableKeepAlives:     true,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if curls.autoRedirect {
				return nil
			}
			return ErrStopRedirect
		},
		Jar:     nil,
		Timeout: 0,
	}

	clientMap.Store(key, hc)

	return hc, nil
}
