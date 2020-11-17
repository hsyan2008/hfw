package hfw

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/session"
	"github.com/hsyan2008/hfw/signal"
)

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Response
//Layout的功能未实现 TODO
type HTTPContext struct {
	Ctx    context.Context    `json:"-"`
	cancel context.CancelFunc `json:"-"`

	HTTPStatus int `json:"-"`

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

	hijacked bool

	*logger.Logger
}

func NewHTTPContext() *HTTPContext {
	httpCtx := &HTTPContext{}
	httpCtx.Ctx, httpCtx.cancel = context.WithCancel(signal.GetSignalContext().Ctx)
	httpCtx.Logger = logger.NewLogger()
	httpCtx.SetTraceID(common.GetPureUUID())

	return httpCtx
}

func NewHTTPContextWithCtx(ctx *HTTPContext) *HTTPContext {
	httpCtx := &HTTPContext{}
	httpCtx.Ctx, httpCtx.cancel = context.WithCancel(ctx.Ctx)
	httpCtx.Logger = logger.NewLogger()
	if ctx == nil || ctx.Logger == nil {
		httpCtx.SetTraceID(common.GetPureUUID())
	} else {
		httpCtx.SetTraceID(common.GetPureUUID(ctx.GetTraceID()))
		httpCtx.AppendPrefix(httpCtx.GetPrefix())
	}

	return httpCtx
}

func NewHTTPContextWithGrpcIncomingCtx(ctx context.Context) *HTTPContext {
	if h, ok := ctx.(*HTTPContext); ok {
		return h
	}
	httpCtx := &HTTPContext{}
	httpCtx.Ctx, httpCtx.cancel = context.WithCancel(ctx)
	httpCtx.Logger = logger.NewLogger()
	traceID := common.GetTraceIDFromIncomingContext(ctx)
	httpCtx.SetTraceID(traceID)

	return httpCtx
}

func NewHTTPContextWithGrpcOutgoingCtx(ctx context.Context) *HTTPContext {
	if h, ok := ctx.(*HTTPContext); ok {
		return h
	}
	httpCtx := &HTTPContext{}
	httpCtx.Ctx, httpCtx.cancel = context.WithCancel(ctx)
	httpCtx.Logger = logger.NewLogger()
	traceID := common.GetTraceIDFromOutgoingContext(ctx)
	httpCtx.SetTraceID(traceID)

	return httpCtx
}

//因为历史原因，不能去掉Ctx，故手动实现context.Context
func (httpCtx *HTTPContext) Deadline() (deadline time.Time, ok bool) {
	return httpCtx.Ctx.Deadline()
}
func (httpCtx *HTTPContext) Done() <-chan struct{} {
	return httpCtx.Ctx.Done()
}
func (httpCtx *HTTPContext) Err() error {
	return httpCtx.Ctx.Err()
}
func (httpCtx *HTTPContext) Value(key interface{}) interface{} {
	return httpCtx.Ctx.Value(key)
}
func (httpCtx *HTTPContext) Cancel() {
	httpCtx.cancel()
	//不能赋值nil，否则导致打印log报错
	// httpCtx.Logger = nil
}

func (httpCtx *HTTPContext) init(w http.ResponseWriter, r *http.Request) {

	httpCtx.HTTPStatus = http.StatusOK

	httpCtx.ResponseWriter = w
	httpCtx.Request = r

	httpCtx.Data = make(map[string]interface{})
	httpCtx.FuncMap = make(map[string]interface{})

	httpCtx.Logger = logger.NewLogger()
	//grpc
	httpCtx.SetTraceID(r.Header.Get(common.GrpcHTTPTraceIDKey))
	//header
	if httpCtx.GetTraceID() == "" {
		httpCtx.SetTraceID(r.Header.Get("Trace-Id"))
	}
	//path
	if httpCtx.GetTraceID() == "" {
		httpCtx.SetTraceID(r.URL.Query().Get("trace_id"))
	}
	if httpCtx.GetTraceID() == "" {
		httpCtx.SetTraceID(common.GetPureUUID())
	}
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
		httpCtx.Output(2, fmt.Sprintf("[ThrowCheck] %s", e.Error()))
	default:
		errMsg = fmt.Sprintf("%v", e)
		httpCtx.Output(2, fmt.Sprintf("[ThrowCheck] No:%d Msg:%v", errNo, errMsg))
	}

	httpCtx.ErrNo = errNo
	httpCtx.ErrMsg = common.GetErrorMap(errNo)
	if len(httpCtx.ErrMsg) == 0 {
		httpCtx.ErrMsg = errMsg
	}

	if httpCtx.ErrNo < Config.ErrorBase && Config.AppID > 0 {
		httpCtx.ErrNo = Config.AppID*Config.ErrorBase + httpCtx.ErrNo
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
		httpCtx.Output(2, fmt.Sprintf("[CheckErr] %s", e.Error()))
	default:
		errMsg = fmt.Sprintf("%v", e)
		httpCtx.Output(2, fmt.Sprintf("[CheckErr] No:%d Msg:%v", errNo, errMsg))
	}

	httpCtx.ErrMsg = common.GetErrorMap(errNo)
	if httpCtx.ErrMsg != "" {
		errMsg = httpCtx.ErrMsg
	}

	if errNo < Config.ErrorBase && Config.AppID > 0 {
		errNo = Config.AppID*Config.ErrorBase + errNo
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

func (httpCtx *HTTPContext) Hijack() (conn net.Conn, bufrw *bufio.ReadWriter, err error) {
	hj, ok := httpCtx.ResponseWriter.(http.Hijacker)
	if !ok {
		httpCtx.Warn("webserver doesn't support hijacking")
		httpCtx.HTTPStatus = http.StatusInternalServerError
		return
	}

	conn, bufrw, err = hj.Hijack()
	if err != nil {
		httpCtx.Warn("Hijack:", err)
		httpCtx.HTTPStatus = http.StatusInternalServerError
		return
	}

	httpCtx.hijacked = true

	return
}
