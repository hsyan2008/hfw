package hfw

//手动匹配路由
import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/websocket"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/encoding"
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
func (ctl *Controller) Init(ctx *HTTPContext) {

	var err error

	// logger.Debug("Controller init")

	if strings.Contains(ctx.Request.URL.RawQuery, "format=json") {
		ctx.IsJSON = true
	} else if strings.Contains(ctx.Request.Header.Get("Accept"), "application/json") {
		ctx.IsJSON = true
	}

	if strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") {
		ctx.IsZip = true
	}

	// _ = ctx.Request.ParseMultipartForm(2 * 1024 * 1024)

	ctx.Session, err = session.NewSession(ctx.Request, DefaultRedisIns, Config)
	ctx.CheckErr(err)
}

//Before ..
func (ctl *Controller) Before(ctx *HTTPContext) {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After(ctx *HTTPContext) {
	// logger.Debug("Controller After")
	if websocket.IsWebSocketUpgrade(ctx.Request) {
		return
	}
}

//Finish ..
func (ctl *Controller) Finish(ctx *HTTPContext) {
	if websocket.IsWebSocketUpgrade(ctx.Request) {
		return
	}

	if ctx.Session != nil {
		ctx.Session.Close(ctx.Request, ctx.ResponseWriter)
	}
	ctx.Output()
}

//NotFound ..
func (ctl *Controller) NotFound(ctx *HTTPContext) {

	ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
	ctx.IsError = true

	ctx.ErrNo = 404
	ctx.ErrMsg = "NotFound"
}

//ServerError ..
//不要手动调用，用于捕获未知错误，手动请用Throw
//该方法不能使用StopRun，也不能panic，因为会被自动调用
func (ctl *Controller) ServerError(ctx *HTTPContext) {

	ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	ctx.IsError = true

	ctx.ErrNo = 500
	ctx.ErrMsg = "ServerError"
}

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Response
//Layout的功能未实现 TODO
type HTTPContext struct {
	ResponseWriter http.ResponseWriter `json:"-"`
	Request        *http.Request       `json:"-"`
	Session        *session.Session    `json:"-"`
	Layout         string              `json:"-"`
	Controll       string              `json:"-"`
	Action         string              `json:"-"`
	Path           string              `json:"-"`
	Template       string              `json:"-"`
	TemplateFile   string              `json:"-"`
	IsJSON         bool                `json:"-"`
	IsZip          bool                `json:"-"`
	//404和500页面被自动更改content-type，导致压缩后有问题，暂时不压缩
	IsError bool                   `json:"-"`
	Data    map[string]interface{} `json:"-"`
	FuncMap map[string]interface{} `json:"-"`

	HasHeader       bool `json:"-"`
	common.Response `json:"response"`
	Header          interface{} `json:"header"`
}

func (ctx *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	ctx.ResponseWriter = w
	ctx.Request = r
	ctx.Layout = ""
	ctx.TemplateFile = ""
	ctx.IsJSON = false
	ctx.IsZip = false
	ctx.IsError = false
	ctx.Data = make(map[string]interface{})
	ctx.FuncMap = make(map[string]interface{})

	ctx.HasHeader = false
	ctx.Header = nil
	ctx.ErrNo = 0
	ctx.ErrMsg = ""
	ctx.Results = nil

	ctx.Controll, ctx.Action, _ = formatURL(r.URL.Path)
	ctx.Path = fmt.Sprintf("%s/%s", ctx.Controll, ctx.Action)
	logger.Infof("Parse Result: Controll:%s, Action:%s", ctx.Controll, ctx.Action)
	// ctx.TemplateFile = fmt.Sprintf("%s.html", ctx.Path)
}

//GetForm 优先post和put,然后get
func (ctx *HTTPContext) GetForm(key string) string {
	return strings.TrimSpace(ctx.Request.FormValue(key))
}

//GetFormInt 优先post和put,然后get，转为int
func (ctx *HTTPContext) GetFormInt(key string) int {
	n, _ := strconv.Atoi(ctx.GetForm(key))
	return n
}

//StopRun ..
func (ctx *HTTPContext) StopRun() {
	// logger.Debug("StopRun")
	panic(ErrStopRun)

	//考虑用runtime.Goexit()，
	//经测试，会执行defer，但连接在这里就中断了，浏览器拿不到结果
}

//Redirect ..
func (ctx *HTTPContext) Redirect(url string) {
	http.Redirect(ctx.ResponseWriter, ctx.Request, url, http.StatusFound)
	ctx.StopRun()
}

//ThrowException ..
func (ctx *HTTPContext) ThrowException(code int64, msg string) {
	ctx.ErrNo = code
	ctx.ErrMsg = msg
	ctx.StopRun()
}

//CheckErr ..
func (ctx *HTTPContext) CheckErr(err error) {
	if nil != err {
		logger.Error(err)
		ctx.ThrowException(500, "系统错误")
	}
}

//Output ..
func (ctx *HTTPContext) Output() {
	// logger.Debug("Output")
	if ctx.ResponseWriter.Header().Get("Location") != "" {
		return
	}
	if (ctx.TemplateFile == "" && ctx.Template == "") || ctx.IsJSON {
		ctx.ReturnJSON()
	} else {
		ctx.Render()
	}
}

var templatesCache = struct {
	list map[string]*template.Template
	l    *sync.RWMutex
}{
	list: make(map[string]*template.Template),
	l:    &sync.RWMutex{},
}

//Render ..
func (ctx *HTTPContext) Render() {
	ctx.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	var (
		t   *template.Template
		err error
	)
	t = ctx.render()

	if !ctx.IsError && ctx.IsZip {
		ctx.ResponseWriter.Header().Del("Content-Length")
		ctx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(ctx.ResponseWriter)
		defer writer.Close()
		err = t.Execute(writer, ctx)
	} else {
		err = t.Execute(ctx.ResponseWriter, ctx)
	}
	ctx.CheckErr(err)
}

func (ctx *HTTPContext) render() (t *template.Template) {
	var key string
	var render func() *template.Template
	var ok bool
	if ctx.Template != "" {
		key = ctx.Path
		// return ctx.renderHtml()
		render = ctx.renderHtml
	} else if ctx.TemplateFile != "" {
		key = ctx.TemplateFile
		render = ctx.renderFile
	}

	if Config.Template.IsCache {
		templatesCache.l.RLock()
		if t, ok = templatesCache.list[key]; !ok {
			templatesCache.l.RUnlock()
			// t = ctx.render()
			t = render()
			templatesCache.l.Lock()
			templatesCache.list[key] = t
			templatesCache.l.Unlock()
		} else {
			templatesCache.l.RUnlock()
		}
	} else {
		// t = ctx.render()
		t = render()
	}

	return t
}

func (ctx *HTTPContext) renderHtml() (t *template.Template) {
	if len(ctx.FuncMap) == 0 {
		t = template.Must(template.New(ctx.Path).Parse(ctx.Template))
	} else {
		t = template.Must(template.New(ctx.Path).Funcs(ctx.FuncMap).Parse(ctx.Template))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}
func (ctx *HTTPContext) renderFile() (t *template.Template) {
	var templateFilePath string
	if common.IsExist(ctx.TemplateFile) {
		templateFilePath = ctx.TemplateFile
	} else {
		templateFilePath = filepath.Join(Config.Template.HTMLPath, ctx.TemplateFile)
	}
	if !common.IsExist(templateFilePath) {
		ctx.ThrowException(500, "system error")
	}
	if len(ctx.FuncMap) == 0 {
		t = template.Must(template.ParseFiles(templateFilePath))
	} else {
		t = template.Must(template.New(filepath.Base(ctx.TemplateFile)).Funcs(ctx.FuncMap).ParseFiles(templateFilePath))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}

//ReturnJSON ..
func (ctx *HTTPContext) ReturnJSON() {
	ctx.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(ctx.Data) > 0 && ctx.Results == nil {
		ctx.Results = ctx.Data
	}

	var w io.Writer
	if !ctx.IsError && ctx.IsZip {
		ctx.ResponseWriter.Header().Del("Content-Length")
		ctx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(ctx.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = ctx.ResponseWriter
	}

	var err error
	if ctx.HasHeader {
		//header + response(err_no + err_msg)
		err = encoding.JSONWriterMarshal(w, ctx)
	} else {
		//err_no + err_msg
		err = encoding.JSONWriterMarshal(w, ctx.Response)
	}
	ctx.CheckErr(err)
}
