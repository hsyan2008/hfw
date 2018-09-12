package hfw

//手动匹配路由
import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
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
	httpCtx.ThrowCheck(500, err)
}

//Before ..
func (ctl *Controller) Before(httpCtx *HTTPContext) {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After(httpCtx *HTTPContext) {
	// logger.Debug("Controller After")
	if websocket.IsWebSocketUpgrade(httpCtx.Request) || httpCtx.IsCloseRender {
		return
	}
}

//Finish ..
func (ctl *Controller) Finish(httpCtx *HTTPContext) {
	if websocket.IsWebSocketUpgrade(httpCtx.Request) || httpCtx.IsCloseRender {
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
