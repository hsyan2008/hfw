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
func (ctl *Controller) Init(httpContext *HTTPContext) {

	var err error

	// logger.Debug("Controller init")

	if strings.Contains(httpContext.Request.URL.RawQuery, "format=json") {
		httpContext.IsJSON = true
	} else if strings.Contains(httpContext.Request.Header.Get("Accept"), "application/json") {
		httpContext.IsJSON = true
	}

	if strings.Contains(httpContext.Request.Header.Get("Accept-Encoding"), "gzip") {
		httpContext.IsZip = true
	}

	// _ = httpContext.Request.ParseMultipartForm(2 * 1024 * 1024)

	httpContext.Session, err = session.NewSession(httpContext.Request, DefaultRedisIns, Config)
	httpContext.CheckErr(err)
}

//Before ..
func (ctl *Controller) Before(httpContext *HTTPContext) {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After(httpContext *HTTPContext) {
	// logger.Debug("Controller After")
	if websocket.IsWebSocketUpgrade(httpContext.Request) || httpContext.isDownload {
		return
	}
}

//Finish ..
func (ctl *Controller) Finish(httpContext *HTTPContext) {
	if websocket.IsWebSocketUpgrade(httpContext.Request) || httpContext.isDownload {
		return
	}

	if httpContext.Session != nil {
		httpContext.Session.Close(httpContext.Request, httpContext.ResponseWriter)
	}

	httpContext.Output()
}

//NotFound ..
func (ctl *Controller) NotFound(httpContext *HTTPContext) {

	httpContext.ResponseWriter.WriteHeader(http.StatusNotFound)
	httpContext.IsError = true

	httpContext.ErrNo = 404
	httpContext.ErrMsg = "NotFound"
}

//ServerError ..
//不要手动调用，用于捕获未知错误，手动请用Throw
//该方法不能使用StopRun，也不能panic，因为会被自动调用
func (ctl *Controller) ServerError(httpContext *HTTPContext) {

	httpContext.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	httpContext.IsError = true

	httpContext.ErrNo = 500
	httpContext.ErrMsg = "ServerError"
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

func (httpContext *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	httpContext.ResponseWriter = w
	httpContext.Request = r
	httpContext.Layout = ""
	httpContext.Template = ""
	httpContext.TemplateFile = ""
	httpContext.IsJSON = false
	httpContext.IsZip = false
	httpContext.IsError = false
	httpContext.Data = make(map[string]interface{})
	httpContext.FuncMap = make(map[string]interface{})

	httpContext.HasHeader = false
	httpContext.Header = nil
	httpContext.ErrNo = 0
	httpContext.ErrMsg = ""
	httpContext.Results = nil

	httpContext.Controll, httpContext.Action, _ = formatURL(r.URL.Path)
	httpContext.Path = fmt.Sprintf("%s/%s", httpContext.Controll, httpContext.Action)
	// httpContext.TemplateFile = fmt.Sprintf("%s.html", httpContext.Path)
}

//GetForm 优先post和put,然后get
func (httpContext *HTTPContext) GetForm(key string) string {
	return strings.TrimSpace(httpContext.Request.FormValue(key))
}

//GetFormInt 优先post和put,然后get，转为int
func (httpContext *HTTPContext) GetFormInt(key string) int {
	n, _ := strconv.Atoi(httpContext.GetForm(key))
	return n
}

//StopRun ..
func (httpContext *HTTPContext) StopRun() {
	// logger.Debug("StopRun")
	panic(ErrStopRun)

	//考虑用runtime.Goexit()，
	//经测试，会执行defer，但连接在这里就中断了，浏览器拿不到结果
}

//Redirect ..
func (httpContext *HTTPContext) Redirect(url string) {
	http.Redirect(httpContext.ResponseWriter, httpContext.Request, url, http.StatusFound)
	httpContext.StopRun()
}

//ThrowException ..
func (httpContext *HTTPContext) ThrowException(code int64, msg string) {
	httpContext.ErrNo = code
	httpContext.ErrMsg = msg
	httpContext.StopRun()
}

//CheckErr ..
func (httpContext *HTTPContext) CheckErr(err error) {
	if nil != err {
		logger.Error(err)
		httpContext.ThrowException(500, "系统错误")
	}
}

//SetDownloadMode ..
func (httpContext *HTTPContext) SetDownloadMode(filename string) {
	httpContext.ResponseWriter.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))
	httpContext.isDownload = true
}
