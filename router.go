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

	if len(routeMap) == 0 {
		panic("nil router")
	}

	//放入pool里
	ctx := ctxPool.Get().(*HTTPContext)
	defer ctxPool.Put(ctx)
	ctx.Init(w, r)

	var reflectVal reflect.Value
	var isNotFound bool
	var instance instance
	var ok bool
	if instance, ok = routeMap[ctx.Path+strings.ToLower(r.Method)]; !ok {
		if instance, ok = routeMap[ctx.Path]; !ok {
			isNotFound = true
			//取默认的
			p := Config.Route.DefaultController + "/" + Config.Route.DefaultAction
			if instance, ok = routeMap[p]; !ok {
				//如果拿不到默认的，就取现有的第一个
				for _, instance = range routeMap {
					break
				}
			}
		}
	}
	reflectVal = instance.reflectVal

	//初始化Controller
	initValue := []reflect.Value{
		reflect.ValueOf(ctx),
	}

	//注意方法必须是大写开头，否则无法调用
	reflectVal.MethodByName("Init").Call(initValue)
	defer reflectVal.MethodByName("Finish").Call(initValue)

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
			reflectVal.MethodByName("ServerError").Call(initValue)
		}
	}()

	reflectVal.MethodByName("Before").Call(initValue)

	var action string
	if isNotFound {
		action = "NotFound"
	} else {
		action = instance.methodName
	}
	logger.Debugf("Query Path: %s -> Call: %s/%s", ctx.Path, instance.controllerName, instance.methodName)
	reflectVal.MethodByName(action).Call(initValue)

	reflectVal.MethodByName("After").Call(initValue)
}

var urlPrefix string

//SetURLPrefix 去除path的前缀
func SetURLPrefix(str string) {
	urlPrefix = strings.Trim(strings.ToLower(str), "/")
	if urlPrefix == "" {
		return
	}
	urlPrefix += "/"
}

type instance struct {
	reflectVal     reflect.Value
	controllerName string
	methodName     string
}

var routeMap = make(map[string]instance)
var routeInit bool

//RegHandler 暂时只支持2段
func RegHandler(pattern string, handler ControllerInterface) (err error) {

	if !routeInit {
		routeInit = true
		http.Handle("/", &Router{})
		// http.HandleFunc("/debug/pprof/", pprof.Index)
		// http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		// http.HandleFunc("/debug/pprof/profile", pprof.Profile)
		// http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		// http.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	controller, _, leave := formatURL(pattern)
	if leave != "" {
		return fmt.Errorf("pattern must only 1 or 2 segment, got %s", pattern)
	}

	reflectVal := reflect.ValueOf(handler)
	rt := reflectVal.Type()
	//controllerName和controller不一定相等
	controllerName := reflect.Indirect(reflectVal).Type().Name()

	numMethod := rt.NumMethod()
	//注意方法必须是大写开头，否则无法调用
	for i := 0; i < numMethod; i++ {
		path := fmt.Sprintf("%s/%s", controller, strings.ToLower(rt.Method(i).Name))
		routeMap[path] = instance{reflectVal, controllerName, rt.Method(i).Name}
	}

	return
}

//RegStaticHandler ..
func RegStaticHandler(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(APPPATH, dir)
	}
	s := "/" + strings.Trim(pattern, "/")
	http.Handle(s, http.FileServer(http.Dir(dir)))
}

func formatURL(url string) (controller string, action string, leave string) {
	//去掉前缀并把url补全为2段
	trimURL := strings.Trim(strings.ToLower(url), "/")
	if urlPrefix != "" {
		trimURL = strings.TrimPrefix(trimURL, urlPrefix)
	}
	urls := strings.SplitN(trimURL, "/", 3)
	if len(urls) == 1 {
		urls = append(urls, Config.Route.DefaultAction)
	}
	if urls[0] == "" {
		urls[0] = strings.ToLower(Config.Route.DefaultController)
	}
	if urls[1] == "" {
		urls[1] = strings.ToLower(Config.Route.DefaultAction)
	}
	if len(urls) == 3 {
		leave = urls[2]
	}

	return urls[0], urls[1], leave
}
