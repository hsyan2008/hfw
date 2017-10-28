package hfw

//手动匹配路由
import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/hsyan2008/go-logger/logger"
)

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Result
//Layout的功能未实现 TODO
type HTTPContext struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Layout         string
	Controll       string
	Action         string
	Path           string
	TemplateFile   string
	IsJSON         bool
	IsZip          bool
	//404和500页面被自动更改content-type，导致压缩后有问题，暂时不压缩
	IsError bool
	Data    map[string]interface{}
	FuncMap map[string]interface{}
	Result
}

//GetForm 优先post和put,然后get
func (ctx *HTTPContext) GetForm(key string) string {
	return ctx.Request.FormValue(key)
}

//GetFormInt 优先post和put,然后get，转为int
func (ctx *HTTPContext) GetFormInt(key string) int {
	n, _ := strconv.Atoi(ctx.GetForm(key))
	return n
}

//ControllerInterface ..
//Init和Finish必定会执行，而且不允许被修改
// Before和After之间是业务逻辑，所有Before也是必定会执行
//用户手动StopRun()后，中止业务逻辑，跳过After，继续Finish
type ControllerInterface interface {
	Init(*HTTPContext)
	Before()
	After()
	Finish()
	Redirect(string)
	Output()
	NotFound()
	ServerError()
	StopRun()
}

//确认Controller实现了接口 ControllerInterface
var _ ControllerInterface = &Controller{}

//ErrStopRun ..
var ErrStopRun = errors.New("user stop run")

//Controller ..
type Controller struct {
	HTTPContext
}

//Init ..
func (ctl *Controller) Init(ctx *HTTPContext) {
	// logger.Debug("Controller init")

	ctl.HTTPContext = *ctx
	ctl.Data = make(map[string]interface{})
	ctl.FuncMap = make(map[string]interface{})

	if strings.Contains(ctl.Request.URL.RawQuery, "format=json") {
		ctl.IsJSON = true
	} else if strings.Contains(ctl.Request.Header.Get("Accept"), "application/json") {
		ctl.IsJSON = true
	}

	if strings.Contains(ctl.Request.Header.Get("Accept-Encoding"), "gzip") {
		ctl.IsZip = true
	}
}

//Before ..
func (ctl *Controller) Before() {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After() {
	// logger.Debug("Controller After")
}

//Finish ..
func (ctl *Controller) Finish() {

	ctl.Output()
}

//StopRun ..
func (ctl *Controller) StopRun() {
	// logger.Debug("StopRun")
	panic(ErrStopRun)

	//考虑用runtime.Goexit()，
	//经测试，会执行defer，但连接在这里就中断了，浏览器拿不到结果
}

//NotFound ..
func (ctl *Controller) NotFound() {

	ctl.ResponseWriter.WriteHeader(http.StatusNotFound)
	ctl.IsError = true

	ctl.ErrNo = 99404
	ctl.ErrMsg = "NotFound"
}

//ServerError ..
//不要手动调用，用于捕获未知错误，手动请用Throw
//该方法不能使用StopRun，也不能panic，因为会被自动调用
func (ctl *Controller) ServerError() {

	ctl.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	ctl.IsError = true

	ctl.ErrNo = 99500
	ctl.ErrMsg = "ServerError"
}

//Redirect ..
func (ctl *Controller) Redirect(url string) {
	http.Redirect(ctl.ResponseWriter, ctl.Request, url, http.StatusFound)
	ctl.StopRun()
}

//ThrowException ..
func (ctl *Controller) ThrowException(code int64, msg string) {
	ctl.ErrNo = code
	ctl.ErrMsg = msg
	ctl.StopRun()
}

//CheckErr ..
func (ctl *Controller) CheckErr(err error) {
	if nil != err {
		logger.Error(err)
		ctl.ThrowException(99500, "系统错误")
	}
}

//Output ..
func (ctl *Controller) Output() {
	// logger.Debug("Output")
	if ctl.ResponseWriter.Header().Get("Location") != "" {
		return
	}
	if ctl.TemplateFile == "" || ctl.IsJSON {
		ctl.RenderJSON()
	} else {
		ctl.RenderFile()
	}
}

var templatesCache = struct {
	list map[string]*template.Template
	l    *sync.RWMutex
}{
	list: make(map[string]*template.Template),
	l:    &sync.RWMutex{},
}

//RenderFile ..
func (ctl *Controller) RenderFile() {

	ctl.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")

	var (
		t   *template.Template
		err error
		ok  bool
	)
	if Config.Template.IsCache {
		templatesCache.l.RLock()
		if t, ok = templatesCache.list[ctl.TemplateFile]; !ok {
			templatesCache.l.RUnlock()
			if len(ctl.FuncMap) == 0 {
				t = template.Must(template.ParseFiles(Config.Template.HTMLPath + ctl.TemplateFile))
			} else {
				t = template.Must(template.New(filepath.Base(ctl.TemplateFile)).Funcs(ctl.FuncMap).ParseFiles(Config.Template.HTMLPath + ctl.TemplateFile))
			}
			t = template.Must(t.ParseGlob(Config.Template.HTMLPath + "/widgets/*.html"))

			templatesCache.l.Lock()
			templatesCache.list[ctl.TemplateFile] = t
			templatesCache.l.Unlock()
		} else {
			templatesCache.l.RUnlock()
		}
	} else {
		if len(ctl.FuncMap) == 0 {
			t = template.Must(template.ParseFiles(Config.Template.HTMLPath + ctl.TemplateFile))
		} else {
			t = template.Must(template.New(filepath.Base(ctl.TemplateFile)).Funcs(ctl.FuncMap).ParseFiles(Config.Template.HTMLPath + ctl.TemplateFile))
		}
		t = template.Must(t.ParseGlob(Config.Template.HTMLPath + "/widgets/*.html"))
	}

	if !ctl.IsError && ctl.IsZip {
		ctl.ResponseWriter.Header().Del("Content-Length")
		ctl.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(ctl.ResponseWriter)
		defer func() {
			_ = writer.Close()
		}()
		err = t.Execute(writer, ctl)
		if err != nil {
			logger.Warn(err)
		}
		ctl.CheckErr(err)
	} else {
		err = t.Execute(ctl.ResponseWriter, ctl)
		if err != nil {
			logger.Warn(err)
		}
		ctl.CheckErr(err)
	}

}

//RenderJSON ..
func (ctl *Controller) RenderJSON() {

	ctl.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")

	if len(ctl.Data) > 0 && ctl.Results == nil {
		ctl.Results = ctl.Data
	}

	b, err := json.Marshal(ctl.Result)
	ctl.CheckErr(err)

	if !ctl.IsError && ctl.IsZip {
		ctl.ResponseWriter.Header().Del("Content-Length")
		ctl.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(ctl.ResponseWriter)
		defer func() {
			_ = writer.Close()
		}()
		_, err = writer.Write(b)
		ctl.CheckErr(err)
	} else {
		_, err = ctl.ResponseWriter.Write(b)
		ctl.CheckErr(err)
	}
}
