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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/grpc/server"
	"github.com/hsyan2008/hfw/signal"
)

var httpCtxPool = &sync.Pool{
	New: func() interface{} {
		return new(HTTPContext)
	},
}

//Router 写测试用例会调用
func Router(w http.ResponseWriter, r *http.Request) {

	signal.GetSignalContext().WgAdd()
	defer signal.GetSignalContext().WgDone()

	if logger.Level() == logger.DEBUG {
		ip := common.GetClientIP(r)
		logger.Debugf("From: %s, Host: %s, Method: %s, Uri: %s %s", ip, r.Host, r.Method, r.URL.String(), "start")
		startTime := time.Now()
		defer func() {
			logger.Debugf("From: %s, Host: %s, Method: %s, Uri: %s %s CostTime: %s",
				ip, r.Host, r.Method, r.URL.String(), "end", time.Since(startTime))
		}()
	}

	onlineNum := atomic.AddUint32(&online, 1)
	logger.Info("online", onlineNum)
	defer func() {
		logger.Info("offline", atomic.AddUint32(&online, ^uint32(0)))
	}()
	err := checkConcurrence(onlineNum)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		logger.Warn(err)
		return
	}

	if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		if server.GetServer() == nil {
			http.Error(w, "grpc server has not init", http.StatusInternalServerError)
			return
		}
		server.GetServer().ServeHTTP(w, r) // gRPC Server
		return
	}

	if len(routeMap) == 0 && len(routeMapMethod) == 0 {
		panic("nil router map")
	}

	httpCtx := httpCtxPool.Get().(*HTTPContext)
	defer httpCtxPool.Put(httpCtx)
	//初始化httpCtx
	httpCtx.init(w, r)
	httpCtx.Controller, httpCtx.Action, _ = formatURL(httpCtx.Request.URL.Path)
	httpCtx.Ctx, httpCtx.Cancel = context.WithCancel(signal.GetSignalContext().Ctx)
	defer httpCtx.Cancel()
	initValue := []reflect.Value{
		reflect.ValueOf(httpCtx),
	}

	//如果用户关闭连接
	go closeNotify(httpCtx)

	instance, action := findInstance(httpCtx)
	httpCtx.Log().Debugf("Query Path: %s -> Call: %s/%s", httpCtx.Request.URL.String(), instance.controllerName, action)
	reflectVal := instance.reflectVal

	//注意方法必须是大写开头，否则无法调用
	reflectVal.MethodByName("Init").Call(initValue)
	defer reflectVal.MethodByName("Finish").Call(initValue)

	defer recoverPanic(httpCtx, reflectVal, initValue)

	reflectVal.MethodByName("Before").Call(initValue)
	defer reflectVal.MethodByName("After").Call(initValue)

	reflectVal.MethodByName(action).Call(initValue)

}

func recoverPanic(httpCtx *HTTPContext, reflectVal reflect.Value, initValue []reflect.Value) {
	//注意recover只能执行一次
	if err := recover(); err != nil {
		//用户触发的
		if err == ErrStopRun {
			return
		}
		httpCtx.Log().Fatal(err, string(common.GetStack()))

		reflectVal.MethodByName("ServerError").Call(initValue)
	}
}

func closeNotify(httpCtx *HTTPContext) {
	if common.IsGoTest() {
		return
	}
	//panic: net/http: CloseNotify called after ServeHTTP finished
	defer func() {
		if err := recover(); err != nil {
			httpCtx.Log().Warn("closeNotify: ", err)
		}
	}()
	select {
	case <-httpCtx.Ctx.Done():
		return
	case <-httpCtx.Request.Context().Done():
		httpCtx.Cancel()
		return
	}
}

var online uint32

func checkConcurrence(onlineNum uint32) (err error) {
	if common.IsGoTest() || Config.Server.Concurrence <= 0 {
		return nil
	}

	if onlineNum > uint32(Config.Server.Concurrence) {
		return errors.New("checkConcurrence: too many concurrence")
	}
	return nil
}

//Handler 暂时只支持2段
func Handler(pattern string, handler ControllerInterface) (err error) {

	if !routeInit {
		routeInit = true
		http.HandleFunc("/", Router)
		http.HandleFunc("/logger/adjust", loggerAdjust)
	}

	controller, _, leave := formatURL(pattern)
	if leave != "" {
		return fmt.Errorf("pattern must only 1 or 2 segment, got %s", pattern)
	}

	reflectVal := reflect.ValueOf(handler)
	rt := reflectVal.Type()
	//controllerName和controller不一定相等
	controllerName := reflect.Indirect(reflectVal).Type().Name()

	if c, ok := routeMapRegister[pattern]; ok {
		if c != controllerName {
			panic(fmt.Sprintf("%s has register controller:%s", pattern, c))
		}
		return
	}
	routeMapRegister[pattern] = controllerName

	numMethod := rt.NumMethod()
	//注意方法必须是大写开头，否则无法调用
	for i := 0; i < numMethod; i++ {
		m := rt.Method(i).Name
		switch m {
		case "Init", "Before", "After", "Finish", "NotFound", "ServerError":
		default:
			actions, method, isMethod := getRequestMethod(m)
			value := &instance{
				reflectVal:     reflectVal,
				controllerName: controllerName,
				methodName:     rt.Method(i).Name,
			}
			if defaultInstance == nil {
				defaultInstance = value
			}
			for _, action := range actions {
				if isMethod {
					path := fmt.Sprintf("%s/%sfor%s", controller, action, method)
					if _, ok := routeMapMethod[path]; ok {
						panic(path + " exist")
					}
					routeMapMethod[path] = value
					logger.Infof("pattern: %s register routeMapMethod: %s", pattern, path)
				} else {
					path := fmt.Sprintf("%s/%s", controller, action)
					if _, ok := routeMap[path]; ok {
						panic(path + " exist")
					}
					routeMap[path] = value
					logger.Infof("pattern: %s register routeMap: %s", pattern, path)
				}
			}
		}
	}

	return
}

//HandlerFunc register HandleFunc
func HandlerFunc(pattern string, h http.HandlerFunc) {
	logger.Infof("HandlerFunc: %s", pattern)
	http.HandleFunc(pattern, h)
}

//StaticHandler ...
//如pattern=css,dir=./static，则css在./static下
func StaticHandler(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(common.GetAppPath(), dir)
	}
	if pattern != "/" {
		pattern = "/" + strings.Trim(pattern, "/") + "/"
	}
	logger.Info("StaticHandler", pattern, dir)
	http.Handle(pattern, http.FileServer(http.Dir(dir)))
}

//StaticStripHandler ...
//如pattern=css,dir=./static/css，则css就是./static/css
func StaticStripHandler(pattern string, dir string) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(common.GetAppPath(), dir)
	}
	if pattern != "/" {
		pattern = "/" + strings.Trim(pattern, "/") + "/"
	}
	logger.Info("StaticStripHandler", pattern, dir)
	http.Handle(pattern, http.StripPrefix(pattern, http.FileServer(http.Dir(dir))))
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

//调整logger的设置
func loggerAdjust(w http.ResponseWriter, r *http.Request) {
	logger.Info("change logger level to", r.FormValue("level"))
	logger.SetLevelStr(r.FormValue("level"))
}
