package hfw

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/session"
)

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Response
//Layout的功能未实现 TODO
type HTTPContext struct {
	*SignalContext `json:"-"`
	Ctx            context.Context    `json:"-"`
	Cancel         context.CancelFunc `json:"-"`

	ResponseWriter http.ResponseWriter `json:"-"`
	Request        *http.Request       `json:"-"`
	Session        *session.Session    `json:"-"`
	Layout         string              `json:"-"`
	Controll       string              `json:"-"`
	Action         string              `json:"-"`
	Path           string              `json:"-"`

	//html文本
	Template string `json:"-"`
	//模板文件
	TemplateFile string `json:"-"`
	IsJSON       bool   `json:"-"`
	IsZip        bool   `json:"-"`
	//404和500页面被自动更改content-type，导致压缩后有问题，暂时不压缩
	IsError bool                   `json:"-"`
	Data    map[string]interface{} `json:"-"`
	FuncMap map[string]interface{} `json:"-"`

	//如果是下载文件，不执行After和Finish
	isDownload bool

	HasHeader       bool `json:"-"`
	common.Response `json:"response"`
	Header          interface{} `json:"header"`
}

func (httpCtx *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	httpCtx.ResponseWriter = w
	httpCtx.Request = r
	httpCtx.Layout = ""
	httpCtx.Template = ""
	httpCtx.TemplateFile = ""
	httpCtx.IsJSON = false
	httpCtx.IsZip = false
	httpCtx.IsError = false
	httpCtx.Data = make(map[string]interface{})
	httpCtx.FuncMap = make(map[string]interface{})

	httpCtx.HasHeader = false
	httpCtx.Header = nil
	httpCtx.ErrNo = 0
	httpCtx.ErrMsg = ""
	httpCtx.Results = nil

	httpCtx.Controll, httpCtx.Action, _ = formatURL(r.URL.Path)
	httpCtx.Path = fmt.Sprintf("%s/%s", httpCtx.Controll, httpCtx.Action)
	// httpCtx.TemplateFile = fmt.Sprintf("%s.html", httpCtx.Path)
}

//GetForm 优先post和put,然后get
func (httpCtx *HTTPContext) GetForm(key string) string {
	return strings.TrimSpace(httpCtx.Request.FormValue(key))
}

//GetFormInt 优先post和put,然后get，转为int
func (httpCtx *HTTPContext) GetFormInt(key string) int {
	n, _ := strconv.Atoi(httpCtx.GetForm(key))
	return n
}

//StopRun ..
func (httpCtx *HTTPContext) StopRun() {
	// logger.Debug("StopRun")
	panic(ErrStopRun)
}

//Redirect ..
func (httpCtx *HTTPContext) Redirect(url string) {
	http.Redirect(httpCtx.ResponseWriter, httpCtx.Request, url, http.StatusFound)
	httpCtx.StopRun()
}

//ThrowException ..
func (httpCtx *HTTPContext) ThrowException(errNo int64, errMsg string) {
	logger.Output(3, "WARN", errNo, errMsg)
	httpCtx.ErrNo = errNo
	httpCtx.ErrMsg = GetErrorMap(errNo)
	if len(httpCtx.ErrMsg) == 0 {
		httpCtx.ErrMsg = errMsg
	}
	httpCtx.StopRun()
}

//ThrowIfError ..
func (httpCtx *HTTPContext) ThrowIfError(errNo int64, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	logger.Output(3, "WARN", errNo, errMsg)
	httpCtx.ErrNo = errNo
	httpCtx.ErrMsg = GetErrorMap(errNo)
	if len(httpCtx.ErrMsg) == 0 {
		httpCtx.ErrMsg = errMsg
	}
	httpCtx.StopRun()
}

//SetDownloadMode ..
func (httpCtx *HTTPContext) SetDownloadMode(filename string) {
	httpCtx.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))
	httpCtx.isDownload = true
}

func (httpCtx *HTTPContext) GetCookie(key string) (s string, err error) {
	cookie, err := httpCtx.Request.Cookie(key)
	if err != nil {
		return
	}

	return cookie.Value, nil
}
func (httpCtx *HTTPContext) SetCookie(key, value string) {
	cookie := &http.Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   httpCtx.Request.URL.Scheme == "https",
	}
	http.SetCookie(httpCtx.ResponseWriter, cookie)
}
