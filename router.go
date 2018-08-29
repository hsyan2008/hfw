package hfw

//手动匹配路由
import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hsyan2008/go-logger/logger"
)

var httpCtxPool = &sync.Pool{
	New: func() interface{} {
		return new(HTTPContext)
	},
}

var concurrenceChan chan bool

func router(w http.ResponseWriter, r *http.Request) {
	if logger.Level() == logger.DEBUG {
		logger.Debug(r.Method, r.URL.String(), "start")
		startTime := time.Now()
		defer func() {
			logger.Debug("ExecTime:", time.Now().Sub(startTime))
			logger.Debug(r.Method, r.URL.String(), "end")
		}()
	}

	signalContext.WgAdd()
	defer signalContext.WgDone()

	if len(routeMap) == 0 {
		panic("nil router")
	}

	//放入pool里
	httpCtx := httpCtxPool.Get().(*HTTPContext)
	defer httpCtxPool.Put(httpCtx)
	httpCtx.Init(w, r)
	httpCtx.SignalContext = signalContext
	httpCtx.Ctx, httpCtx.Cancel = context.WithCancel(signalContext.Ctx)
	defer httpCtx.Cancel()

	//如果用户关闭连接
	go func() {
		//panic: net/http: CloseNotify called after ServeHTTP finished
		defer func() {
			recover()
		}()
		select {
		case <-httpCtx.Ctx.Done():
			return
		case <-httpCtx.ResponseWriter.(http.CloseNotifier).CloseNotify():
			httpCtx.Cancel()
			return
		}
	}()

	if Config.Server.Concurrence > 0 {
		err := holdConcurrenceChan(httpCtx)
		if err != nil {
			logger.Warn(err)
			return
		}
		defer func() {
			<-concurrenceChan
		}()
	}

	var reflectVal reflect.Value
	var isNotFound bool
	var instance instance
	var ok bool
	if instance, ok = routeMapMethod[httpCtx.Path+"with"+strings.ToLower(r.Method)]; !ok {
		if instance, ok = routeMap[httpCtx.Path]; !ok {
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
		reflect.ValueOf(httpCtx),
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
	logger.Debugf("Query Path: %s -> Call: %s/%s", r.URL.String(), instance.controllerName, action)
	reflectVal.MethodByName(action).Call(initValue)

	reflectVal.MethodByName("After").Call(initValue)
}

func holdConcurrenceChan(httpCtx *HTTPContext) (err error) {
	select {
	//用户关闭连接
	case <-httpCtx.Ctx.Done():
		return httpCtx.Ctx.Err()
	//服务关闭
	case <-signalContext.Ctx.Done():
		return errors.New("shutdown")
	case <-time.After(3 * time.Second):
		hj, ok := httpCtx.ResponseWriter.(http.Hijacker)
		if !ok {
			return errors.New("Hijacker err")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return err
		}
		conn.Close()
		return errors.New("timeout")
	case concurrenceChan <- true:
		return
	}
}

type instance struct {
	reflectVal     reflect.Value
	controllerName string
	methodName     string
}

var routeMap = make(map[string]instance)
var routeMapMethod = make(map[string]instance)
var routeInit bool

//Handler 暂时只支持2段
func Handler(pattern string, handler ControllerInterface) (err error) {

	if !routeInit {
		routeInit = true
		http.HandleFunc("/", router)
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
		m := rt.Method(i).Name
		switch m {
		case "Init", "Before", "After", "Finish", "NotFound", "ServerError":
		default:
			isMethod := false
			//必须With+全大写结尾
			for _, v := range []string{"GET", "POST", "PUT", "DELETE"} {
				if strings.HasSuffix(m, "With"+v) && strings.LastIndex(m, "With"+v) > 0 {
					isMethod = true
					break
				}
			}
			path := fmt.Sprintf("%s/%s", controller, strings.ToLower(m))
			value := instance{
				reflectVal:     reflectVal,
				controllerName: controllerName,
				methodName:     rt.Method(i).Name,
			}
			if isMethod {
				routeMapMethod[path] = value
				logger.Infof("pattern: %s register routeMapMethod: %s", pattern, path)
			} else {
				routeMap[path] = value
				logger.Infof("pattern: %s register routeMap: %s", pattern, path)
			}
		}

	}

	return
}

//StaticHandler ..
func StaticHandler(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(APPPATH, dir)
	}
	s := "/" + strings.Trim(pattern, "/")
	http.Handle(s, http.FileServer(http.Dir(dir)))
}

func formatURL(url string) (controller string, action string, leave string) {
	//去掉前缀并把url补全为2段
	trimURL := strings.Trim(strings.ToLower(url), "/")
	urls := strings.SplitN(trimURL, "/", 3)
	if len(urls) == 1 {
		urls = append(urls, Config.Route.DefaultAction)
	}
	if urls[0] == "" {
		urls[0] = Config.Route.DefaultController
	}
	if urls[1] == "" {
		urls[1] = Config.Route.DefaultAction
	}
	if len(urls) == 3 {
		leave = urls[2]
	}

	return urls[0], urls[1], leave
}
