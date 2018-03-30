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

//ControllerInterface ..
//init和finish必定会执行，而且不允许被修改
// Before和After之间是业务逻辑，所有Before也是必定会执行
//用户手动StopRun()后，中止业务逻辑，跳过After，继续Finish
type ControllerInterface interface {
	init(*HTTPContext)
	Before(*HTTPContext)
	After(*HTTPContext)
	finish(*HTTPContext)
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
func (ctl *Controller) init(ctx *HTTPContext) {

	Wg.Add(1)

	// logger.Debug("Controller init")

	if strings.Contains(ctx.Request.URL.RawQuery, "format=json") {
		ctx.IsJSON = true
	} else if strings.Contains(ctx.Request.Header.Get("Accept"), "application/json") {
		ctx.IsJSON = true
	}

	if strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") {
		ctx.IsZip = true
	}

	_ = ctx.Request.ParseMultipartForm(2 * 1024 * 1024)

	if Config.Session.SessID != "" {
		var sessId string
		cookie, err := ctx.Request.Cookie(Config.Session.SessID)
		if err == nil {
			sessId = cookie.Value
		}
		ctx.Session, err = NewSession(sessId)
		ctx.CheckErr(err)
	}
}

//Finish ..
func (ctl *Controller) finish(ctx *HTTPContext) {

	defer Wg.Done()

	// if Config.Session.SessID != "" {
	// 	cookie := http.Cookie{Name: Config.Session.SessID, Value: ctx.Session.newid, Path: "/", HttpOnly: true}
	// 	http.SetCookie(ctx.ResponseWriter, &cookie)
	// 	ctx.Session.Rename()
	// }

	ctx.Output()
}

//Before ..
func (ctl *Controller) Before(ctx *HTTPContext) {
	// logger.Debug("Controller Before")
}

//After ..
func (ctl *Controller) After(ctx *HTTPContext) {
	// logger.Debug("Controller After")
}

//NotFound ..
func (ctl *Controller) NotFound(ctx *HTTPContext) {

	ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
	ctx.IsError = true

	ctx.ErrNo = 99404
	ctx.ErrMsg = "NotFound"
}

//ServerError ..
//不要手动调用，用于捕获未知错误，手动请用Throw
//该方法不能使用StopRun，也不能panic，因为会被自动调用
func (ctl *Controller) ServerError(ctx *HTTPContext) {

	ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
	ctx.IsError = true

	ctx.ErrNo = 99500
	ctx.ErrMsg = "ServerError"
}

//HTTPContext ..
//渲染模板的数据放Data
//Json里的数据放Result
//Layout的功能未实现 TODO
type HTTPContext struct {
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Session        *Session
	Layout         string
	Controll       string
	Action         string
	Path           string
	Template       string
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
		ctx.ThrowException(99500, "系统错误")
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
		defer func() {
			_ = writer.Close()
		}()
		err = t.Execute(writer, ctx)
		if err != nil {
			logger.Warn(err)
		}
		ctx.CheckErr(err)
	} else {
		err = t.Execute(ctx.ResponseWriter, ctx)
		if err != nil {
			logger.Warn(err)
		}
		ctx.CheckErr(err)
	}

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
		// return ctx.renderFile()
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
	templateFilePath := filepath.Join(Config.Template.HTMLPath, ctx.TemplateFile)
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

	b, err := json.Marshal(ctx.Result)
	ctx.CheckErr(err)

	if !ctx.IsError && ctx.IsZip {
		ctx.ResponseWriter.Header().Del("Content-Length")
		ctx.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(ctx.ResponseWriter)
		defer func() {
			_ = writer.Close()
		}()
		_, err = writer.Write(b)
		ctx.CheckErr(err)
	} else {
		_, err = ctx.ResponseWriter.Write(b)
		ctx.CheckErr(err)
	}
}
