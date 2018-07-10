package curl

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
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
	"unicode/utf8"

	"github.com/axgle/mahonia"
	"github.com/hsyan2008/hfw2/common"
)

type Response struct {
	Headers    map[string]string `json:"headers"`
	Cookie     string            `json:"cookie"`
	Url        string            `json:"url"`
	FollowUrls []string          `json:"follow_urls"`
	Body       []byte            `json:"body"`
}

var stopRedirect = errors.New("no redirects allowed")

type Curl struct {
	Url, Method, Cookie, Referer string

	Headers map[string]string
	Options map[string]bool

	RedirectCount int

	//string格式
	PostString string
	//key=>value格式
	PostFields neturl.Values
	//文件
	PostFiles map[string]string

	timeout time.Duration

	followUrls []string
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

var httpclient = &http.Client{
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

func (curls *Curl) SetTimeout(t int) {
	curls.timeout = time.Duration(t)
}

func (curls *Curl) SetOptions(options map[string]bool) {
	for k, v := range options {
		curls.SetOption(k, v)
	}
}
func (curls *Curl) SetOption(key string, val bool) {
	curls.Options[key] = val
}

func (curls *Curl) Request() (rs Response, err error) {
	var httprequest *http.Request
	var httpresponse *http.Response

	if "" != curls.PostString || len(curls.PostFields) > 0 || len(curls.PostFiles) > 0 {
		curls.Method = "post"
	}

	if curls.Method == "post" {
		httprequest, err = curls.postForm()
		if err != nil {
			return
		}
	} else {
		httprequest, _ = http.NewRequest("GET", curls.Url, nil)
	}

	if curls.Headers != nil {
		for key, value := range curls.Headers {
			httprequest.Header.Add(key, value)
		}
	}

	if curls.Cookie != "" {
		httprequest.Header.Add("Cookie", curls.Cookie)
	}
	if curls.Referer != "" {
		httprequest.Header.Add("Referer", curls.Referer)
	}

	c := make(chan bool, 1)
	go func() {
		httpresponse, err = httpclient.Do(httprequest)
		c <- true
	}()

	select {
	case <-time.After(curls.timeout * time.Second):
		tr.CancelRequest(httprequest)
		err = errors.New("request time out")
	case <-c:
	}

	if nil != err {
		//不是重定向里抛出的错误
		urlError, ok := err.(*neturl.Error)
		if !ok || urlError.Err != stopRedirect {
			return rs, err
		}
	}

	defer func() {
		_ = httpresponse.Body.Close()
	}()

	return curls.curlResponse(httpresponse)
}

func (curls *Curl) postForm() (httprequest *http.Request, err error) {

	if curls.PostString != "" {
		b := strings.NewReader(curls.PostString)
		httprequest, _ = http.NewRequest("POST", curls.Url, b)
		if v, ok := curls.Headers["Content-Type"]; ok {
			httprequest.Header.Add("Content-Type", v)
		} else {
			httprequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")
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
			defer func() {
				_ = fh.Close()
			}()

			_, err = io.Copy(fileWriter, fh)
			if err != nil {
				return nil, err
			}
		}
		//必须在这里，不能defer
		_ = bodyWriter.Close()

		httprequest, _ = http.NewRequest("POST", curls.Url, b)
		httprequest.Header.Add("Content-Type", bodyWriter.FormDataContentType())
	}

	delete(curls.Headers, "Content-Type")

	return httprequest, nil
}

//处理获取的页面
func (curls *Curl) curlResponse(resp *http.Response) (response Response, err error) {
	response.Body, err = curls.getBody(resp)
	if err != nil {
		return
	}
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

				return curls.Request()
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

	return response, nil
}

func (curls *Curl) getBody(resp *http.Response) (body []byte, err error) {

	//如果出现302或301，已经表示是不自动重定向 或者出现200才读
	if resp.StatusCode == http.StatusOK || resp.StatusCode == 299 {
		if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
			var reader *gzip.Reader
			reader, err = gzip.NewReader(resp.Body)
			if err != nil {
				return
			}
			body, err = ioutil.ReadAll(reader)
		} else if strings.Contains(resp.Header.Get("Content-Encoding"), "deflate") {
			reader := flate.NewReader(resp.Body)
			defer func() {
				_ = reader.Close()
			}()
			body, err = ioutil.ReadAll(reader)
		} else {
			body, err = ioutil.ReadAll(resp.Body)
		}
		if err != nil {
			return
		}
	}

	return bytes.TrimSpace(body), nil
}

func (curl *Curl) toUtf8(body string) string {
	if utf8.ValidString(body) == false {
		enc := mahonia.NewDecoder("gb18030")
		body = enc.ConvertString(body)
	}

	return body
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
