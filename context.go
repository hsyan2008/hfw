package hfw

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/session"
)

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Response
//Layout的功能未实现 TODO
type HTTPContext struct {
	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	ResponseWriter http.ResponseWriter `json:"-"`
	Request        *http.Request       `json:"-"`
	Session        *session.Session    `json:"-"`
	Layout         string              `json:"-"`
	//对应的struct名称，大小写一致
	Controller string `json:"-"`
	//对应的struct方法的名称，大小写一致
	Action string `json:"-"`
	Path   string `json:"-"`

	IsZip bool `json:"-"`
	//404和500页面被自动更改content-type，导致压缩后有问题，暂时不压缩
	IsError bool `json:"-"`

	//html文本
	Template string `json:"-"`
	//模板文件
	TemplateFile string `json:"-"`
	//主要用于模板渲染
	Data    map[string]interface{} `json:"-"`
	FuncMap map[string]interface{} `json:"-"`

	IsJSON bool `json:"-"`
	//返回的json是否包含Header
	HasHeader bool `json:"-"`
	//是否只返回Response.Results里的数据
	IsOnlyResults   bool `json:"-"`
	common.Response `json:"response"`
	Header          interface{} `json:"header"`

	//如果是下载文件，不执行After和Finish
	IsCloseRender bool `json:"-"`

	log *logger.Log
}

func (httpCtx *HTTPContext) init(w http.ResponseWriter, r *http.Request) {
	httpCtx.ResponseWriter = w
	httpCtx.Request = r
	httpCtx.Session = nil

	httpCtx.Controller = ""
	httpCtx.Action = ""
	httpCtx.Path = ""

	httpCtx.Layout = ""
	httpCtx.Template = ""
	httpCtx.TemplateFile = ""
	httpCtx.IsJSON = false
	httpCtx.IsZip = false
	httpCtx.IsError = false
	httpCtx.Data = make(map[string]interface{})
	httpCtx.FuncMap = make(map[string]interface{})

	httpCtx.IsCloseRender = false

	httpCtx.HasHeader = false
	httpCtx.Header = nil
	httpCtx.ErrNo = 0
	httpCtx.ErrMsg = ""
	httpCtx.Results = nil

	httpCtx.log = httpCtx.Log()
}

func (httpCtx *HTTPContext) Log() *logger.Log {
	if httpCtx.log == nil {
		httpCtx.log = logger.NewLog()
		httpCtx.log.SetTraceID(uuid.New().String())
	}

	return httpCtx.log
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

//ErrStopRun ..
var ErrStopRun = errors.New("user stop run")

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

//ThrowCheck
func (httpCtx *HTTPContext) ThrowCheck(errNo int64, i interface{}) {
	if i == nil || errNo == 0 {
		return
	}
	var errMsg string
	switch e := i.(type) {
	case *common.RespErr:
		errNo = e.ErrNo()
		errMsg = e.ErrMsg()
		httpCtx.Log().Output(3, fmt.Sprintf("[ThrowCheck] %s", e.Error()))
	default:
		errMsg = fmt.Sprintf("%v", e)
		httpCtx.Log().Output(3, fmt.Sprintf("[ThrowCheck] No:%d Msg:%v", errNo, errMsg))
	}

	httpCtx.ErrNo = errNo
	httpCtx.ErrMsg = common.GetErrorMap(errNo)
	if len(httpCtx.ErrMsg) == 0 {
		httpCtx.ErrMsg = errMsg
	}

	if httpCtx.ErrNo < 100000 && Config.AppID > 0 {
		httpCtx.ErrNo = Config.AppID*100000 + httpCtx.ErrNo
	}

	httpCtx.StopRun()
}

//CheckErr
func (httpCtx *HTTPContext) CheckErr(errNo int64, i interface{}) (int64, string) {
	var errMsg string
	if i == nil || errNo == 0 {
		return 0, errMsg
	}
	switch e := i.(type) {
	case *common.RespErr:
		errNo = e.ErrNo()
		errMsg = e.ErrMsg()
		httpCtx.Log().Output(3, fmt.Sprintf("[CheckErr] %s", e.Error()))
	default:
		errMsg = fmt.Sprintf("%v", e)
		httpCtx.Log().Output(3, fmt.Sprintf("[CheckErr] No:%d Msg:%v", errNo, errMsg))
	}

	httpCtx.ErrMsg = common.GetErrorMap(errNo)
	if httpCtx.ErrMsg == "" {
		errMsg = httpCtx.ErrMsg
	}

	if errNo < 100000 && Config.AppID > 0 {
		errNo = Config.AppID*100000 + errNo
	}

	return errNo, errMsg
}

//SetDownloadMode ..
func (httpCtx *HTTPContext) SetDownloadMode(filename string) {
	httpCtx.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))
	httpCtx.IsCloseRender = true
}

func (httpCtx *HTTPContext) GetCookie(key string) (s string) {
	cookie, _ := httpCtx.Request.Cookie(key)
	if cookie != nil {
		return cookie.Value
	}

	return
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
