package hfw

//手动匹配路由
import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/session"
)

//ControllerInterface ..
//init和finish必定会执行，而且不允许被修改
// Before和After之间是业务逻辑，所有Before也是必定会执行
//用户手动StopRun()后，中止业务逻辑，跳过After，继续Finish
type ControllerInterface interface {
	Init(*HTTPContext)
	Before(*HTTPContext)
	After(*HTTPContext)
	Finish(*HTTPContext)
	NotFound(*HTTPContext)
	ServerError(*HTTPContext)
}

//确认Controller实现了接口 ControllerInterface
var _ ControllerInterface = &Controller{}

//ErrStopRun ..
var ErrStopRun = errors.New("user stop run")

//Controller ..
type Controller struct {
}

//Init ..
func (ctl *Controller) Init(httpCtx *HTTPContext) {

	var err error

	// logger.Debug("Controller init")

	if strings.Contains(httpCtx.Request.URL.RawQuery, "format=json") {
		httpCtx.IsJSON = true
	} else if strings.Contains(httpCtx.Request.Header.Get("Accept"), "application/json") {
		httpCtx.IsJSON = true
	}

	if strings.Contains(httpCtx.Request.Header.Get("Accept-Encoding"), "gzip") {
		httpCtx.IsZip = true
	}

	// _ = httpCtx.Request.ParseMultipartForm(2 * 1024 * 1024)

	httpCtx.Session, err = session.NewSession(httpCtx.Request, DefaultRedisIns, Config)
	httpCtx.CheckErr(err)
}

//Before ..
func (ctl *Controller) Before(httpCtx *HTTPContext) {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After(httpCtx *HTTPContext) {
	// logger.Debug("Controller After")
	if websocket.IsWebSocketUpgrade(httpCtx.Request) || httpCtx.isDownload {
		return
	}
}

//Finish ..
func (ctl *Controller) Finish(httpCtx *HTTPContext) {
	if websocket.IsWebSocketUpgrade(httpCtx.Request) || httpCtx.isDownload {
		return
	}

	if httpCtx.Session != nil {
		httpCtx.Session.Close(httpCtx.Request, httpCtx.ResponseWriter)
	}

	httpCtx.Output()
}

//NotFound ..
func (ctl *Controller) NotFound(httpCtx *HTTPContext) {

	httpCtx.ResponseWriter.WriteHeader(http.StatusNotFound)
	httpCtx.IsError = true

	httpCtx.ErrNo = 404
	httpCtx.ErrMsg = "NotFound"
}

//ServerError ..
//不要手动调用，用于捕获未知错误，手动请用Throw
//该方法不能使用StopRun，也不能panic，因为会被自动调用
func (ctl *Controller) ServerError(httpCtx *HTTPContext) {

	httpCtx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	httpCtx.IsError = true

	httpCtx.ErrNo = 500
	httpCtx.ErrMsg = "ServerError"
}

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

	//考虑用runtime.Goexit()，
	//经测试，会执行defer，但连接在这里就中断了，浏览器拿不到结果
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

//CheckErr ..
func (httpCtx *HTTPContext) CheckErr(err error) {
	if nil != err {
		logger.Error(err)
		httpCtx.ThrowException(500, "系统错误")
	}
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
