package hfw

//手动匹配路由
import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/hsyan2008/go-logger/logger"
)

var ctxPool = &sync.Pool{
	New: func() interface{} {
		return new(HTTPContext)
	},
}

//Router ..
type Router struct {
	C ControllerInterface
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	logger.Debug(r.Method, r.URL.String(), "start")
	defer logger.Debug(r.Method, r.URL.String(), "end")

	Ctx.WgAdd()
	defer Ctx.WgDone()

	//去掉前缀并把url补全为2段
	trimURL := strings.Trim(strings.ToLower(r.URL.Path), "/")
	if urlPrefix != "" {
		trimURL = strings.TrimPrefix(trimURL, urlPrefix)
	}
	//如果url为/，切分后为1个空元素
	if trimURL == "" {
		trimURL = Config.Route.DefaultController
	}
	urls := strings.SplitN(trimURL, "/", 3)

	if len(urls) == 0 {
		urls = append(urls, Config.Route.DefaultController)
	}
	if len(urls) == 1 {
		urls = append(urls, Config.Route.DefaultAction)
	}

	//放入pool里
	ctx := ctxPool.Get().(*HTTPContext)
	defer ctxPool.Put(ctx)
	ctx.ResponseWriter = w
	ctx.Request = r
	ctx.Controll = urls[0]
	ctx.Action = urls[1]
	ctx.Path = fmt.Sprintf("%s/%s", ctx.Controll, ctx.Action)
	logger.Infof("Parse Result: Controll:%s, Action:%s", ctx.Controll, ctx.Action)
	// ctx.TemplateFile = fmt.Sprintf("%s.html", ctx.Path)
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

	reflectVal := reflect.ValueOf(router.C)
	rt := reflectVal.Type()
	ct := reflect.Indirect(reflectVal).Type()

	//初始化Controller
	initValue := []reflect.Value{
		reflect.ValueOf(ctx),
	}

	//注意方法必须是大写开头，否则无法调用
	router.C.init(ctx)
	defer router.C.finish(ctx)

	defer func() {
		//注意recover只能执行一次
		if err := recover(); err != nil {
			//用户触发的
			if err == ErrStopRun {
				return
			}
			buf := make([]byte, 1<<20)
			num := runtime.Stack(buf, false)
			logger.Warn(num, string(buf))

			errMsg := fmt.Sprint(err)
			logger.Warn(errMsg)
			router.C.ServerError(ctx)
		}
	}()

	var action string
	//非法的Controller
	if strings.ToLower(ct.Name()) != ctx.Controll {
		action = "NotFound"
	} else {
		numMethod := rt.NumMethod()
		for i := 0; i < numMethod; i++ {
			if strings.ToLower(rt.Method(i).Name) == ctx.Action {
				action = rt.Method(i).Name
				break
			}
		}
		method := strings.Title(strings.ToLower(r.Method))
		if _, ok := rt.MethodByName(action + method); ok {
			action = action + method
		}
		//非法的Action
		if action == "" {
			action = "NotFound"
		}
	}

	router.C.Before(ctx)
	logger.Debugf("Query Path: %s -> Call: %s/%s", ctx.Path, ct.Name(), action)
	reflectVal.MethodByName(action).Call(initValue)
	router.C.After(ctx)
}

var urlPrefix string

func SetUrlPrefix(str string) {
	urlPrefix = strings.Trim(str, "/")
	if urlPrefix == "" {
		return
	}
	urlPrefix += "/"
}

//RegisterRoute ..
func RegisterRoute(pattern string, handler ControllerInterface) {
	s := "/" + strings.Trim(pattern, "/")
	if s == "/" {
		http.Handle(s, &Router{C: handler})
	} else {
		//如果没有这个，会重定向
		http.Handle(s, &Router{C: handler})
		//如果没有这个，会匹配到/
		http.Handle(s+"/", &Router{C: handler})
	}
}

//RegisterStatic ..
func RegisterStatic(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(APPPATH, dir)
	}
	s := "/" + strings.Trim(pattern, "/")
	if s == "/" {
		http.Handle(s, http.FileServer(http.Dir(dir)))
	} else {
		//最后一定要加上/
		http.Handle(s+"/", http.FileServer(http.Dir(dir)))
	}
}

//RegisterFile .. favicon.ico
func RegisterFile(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(APPPATH, dir)
	}
	s := "/" + strings.Trim(pattern, "/")
	http.Handle(s, http.FileServer(http.Dir(dir)))
}
