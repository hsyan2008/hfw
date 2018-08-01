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
	if websocket.IsWebSocketUpgrade(httpContext.Request) {
		return
	}
}

//Finish ..
func (ctl *Controller) Finish(httpContext *HTTPContext) {
	if websocket.IsWebSocketUpgrade(httpContext.Request) {
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

func (httpContext *HTTPContext) Init(w http.ResponseWriter, r *http.Request) {
	httpContext.ResponseWriter = w
	httpContext.Request = r
	httpContext.Layout = ""
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
	logger.Infof("Parse Result: Controll:%s, Action:%s", httpContext.Controll, httpContext.Action)
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

//Output ..
func (httpContext *HTTPContext) Output() {
	// logger.Debug("Output")
	if httpContext.ResponseWriter.Header().Get("Location") != "" {
		return
	}
	if (httpContext.TemplateFile == "" && httpContext.Template == "") || httpContext.IsJSON {
		httpContext.ReturnJSON()
	} else {
		httpContext.Render()
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
func (httpContext *HTTPContext) Render() {
	httpContext.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	var (
		t   *template.Template
		err error
	)
	t = httpContext.render()

	if !httpContext.IsError && httpContext.IsZip {
		httpContext.ResponseWriter.Header().Del("Content-Length")
		httpContext.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(httpContext.ResponseWriter)
		defer writer.Close()
		err = t.Execute(writer, httpContext)
	} else {
		err = t.Execute(httpContext.ResponseWriter, httpContext)
	}
	httpContext.CheckErr(err)
}

func (httpContext *HTTPContext) render() (t *template.Template) {
	var key string
	var render func() *template.Template
	var ok bool
	if httpContext.Template != "" {
		key = httpContext.Path
		// return httpContext.renderHtml()
		render = httpContext.renderHtml
	} else if httpContext.TemplateFile != "" {
		key = httpContext.TemplateFile
		render = httpContext.renderFile
	}

	if Config.Template.IsCache {
		templatesCache.l.RLock()
		if t, ok = templatesCache.list[key]; !ok {
			templatesCache.l.RUnlock()
			// t = httpContext.render()
			t = render()
			templatesCache.l.Lock()
			templatesCache.list[key] = t
			templatesCache.l.Unlock()
		} else {
			templatesCache.l.RUnlock()
		}
	} else {
		// t = httpContext.render()
		t = render()
	}

	return t
}

func (httpContext *HTTPContext) renderHtml() (t *template.Template) {
	if len(httpContext.FuncMap) == 0 {
		t = template.Must(template.New(httpContext.Path).Parse(httpContext.Template))
	} else {
		t = template.Must(template.New(httpContext.Path).Funcs(httpContext.FuncMap).Parse(httpContext.Template))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}
func (httpContext *HTTPContext) renderFile() (t *template.Template) {
	var templateFilePath string
	if common.IsExist(httpContext.TemplateFile) {
		templateFilePath = httpContext.TemplateFile
	} else {
		templateFilePath = filepath.Join(Config.Template.HTMLPath, httpContext.TemplateFile)
	}
	if !common.IsExist(templateFilePath) {
		httpContext.ThrowException(500, "system error")
	}
	if len(httpContext.FuncMap) == 0 {
		t = template.Must(template.ParseFiles(templateFilePath))
	} else {
		t = template.Must(template.New(filepath.Base(httpContext.TemplateFile)).Funcs(httpContext.FuncMap).ParseFiles(templateFilePath))
	}
	if Config.Template.WidgetsPath != "" {
		widgetsPath := filepath.Join(Config.Template.HTMLPath, Config.Template.WidgetsPath)
		t = template.Must(t.ParseGlob(widgetsPath))
	}

	return
}

//ReturnJSON ..
func (httpContext *HTTPContext) ReturnJSON() {
	httpContext.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	if len(httpContext.Data) > 0 && httpContext.Results == nil {
		httpContext.Results = httpContext.Data
	}

	var w io.Writer
	if !httpContext.IsError && httpContext.IsZip {
		httpContext.ResponseWriter.Header().Del("Content-Length")
		httpContext.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		w = gzip.NewWriter(httpContext.ResponseWriter)
		defer w.(io.WriteCloser).Close()
	} else {
		w = httpContext.ResponseWriter
	}

	var err error
	if httpContext.HasHeader {
		//header + response(err_no + err_msg)
		err = encoding.JSONWriterMarshal(w, httpContext)
	} else {
		//err_no + err_msg
		err = encoding.JSONWriterMarshal(w, httpContext.Response)
	}
	httpContext.CheckErr(err)
}
