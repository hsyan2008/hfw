package hfw

//手动匹配路由
import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/grpc/server"
	"github.com/hsyan2008/hfw/prometheus"
)

//Router 写测试用例会调用
func Router(w http.ResponseWriter, r *http.Request) {

	//grpc
	if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		if server.GetServer() == nil {
			http.Error(w, "grpc server has not init", http.StatusInternalServerError)
			return
		}
		//防止出现并没有Trace_id的情况
		r.Header.Set(common.GrpcHTTPTraceIDKey, common.GetTraceIDFromRequest(r))
		server.GetServer().ServeHTTP(w, r) // gRPC Server
		return
	}

	//初始化httpCtx
	httpCtx := initCtx(w, r)
	defer httpCtx.Cancel()

	//如果用户关闭连接
	go closeNotify(httpCtx)

	prometheus.RequestsTotal(r.URL.Path, r.Method)
	defer func(path, method string, startTime time.Time) {
		costTime := time.Since(startTime)
		httpCtx.Mixf("Path:%s Method:%s CostTime:%s", path, method, costTime)
		prometheus.RequestsCosttime(path, method, costTime)
	}(r.URL.Path, r.Method, time.Now())

	onlineNum := atomic.AddUint32(&online, 1)
	httpCtx.Mixf("From:%s Path:%s Online:%d", r.RemoteAddr, r.URL.String(), onlineNum)
	defer func() {
		atomic.AddUint32(&online, ^uint32(0))
	}()
	err := checkConcurrence(onlineNum)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		httpCtx.Warn(err)
		return
	}

	if len(routeMap) == 0 && len(routeMapMethod) == 0 {
		httpCtx.Warn(httpCtx.Request.URL.Path, "nil routeMap or routeMapMethod")
		(&Controller{}).NotFound(httpCtx)
		return
	}

	initValue := []reflect.Value{
		reflect.ValueOf(httpCtx),
	}

	instance, methodName := findInstanceByPath(httpCtx)
	httpCtx.Debugf("Path:%s -> Call:%s/%s", httpCtx.Request.URL.Path, httpCtx.Controller, httpCtx.Action)
	reflectVal := instance.reflectVal

	//注意方法必须是大写开头，否则无法调用
	reflectVal.MethodByName("Init").Call(initValue)
	defer reflectVal.MethodByName("Finish").Call(initValue)

	defer recoverPanic(httpCtx, reflectVal, initValue)

	reflectVal.MethodByName("Before").Call(initValue)
	defer reflectVal.MethodByName("After").Call(initValue)

	reflectVal.MethodByName(methodName).Call(initValue)
}

func recoverPanic(httpCtx *HTTPContext, reflectVal reflect.Value, initValue []reflect.Value) {
	//注意recover只能执行一次
	if err := recover(); err != nil {
		//用户触发的
		if err == ErrStopRun {
			return
		}
		httpCtx.Fatal(err, string(common.GetStack()))

		reflectVal.MethodByName("ServerError").Call(initValue)
	}
}

func closeNotify(httpCtx *HTTPContext) {
	if common.IsGoTest() {
		return
	}
	<-httpCtx.Request.Context().Done()
	httpCtx.Cancel()
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

func init() {
	http.HandleFunc("/logger/adjust", loggerAdjust)
}

var isInit bool

//Handler 暂时只支持2段
func Handler(pattern string, handler ControllerInterface) (err error) {
	if isInit == false {
		isInit = true
		http.HandleFunc("/", Router)
	}

	controllerPath := completeURL(pattern)

	reflectVal := reflect.ValueOf(handler)
	rt := reflectVal.Type()
	//controllerName和controller不一定相等
	controllerName := reflect.Indirect(reflectVal).Type().Name()

	if c, ok := routeMapRegister[controllerPath]; ok {
		if c != controllerName {
			panic(fmt.Sprintf("%s has register controller:%s", pattern, c))
		}
		return
	}
	routeMapRegister[controllerPath] = controllerName

	numMethod := rt.NumMethod()
	//注意方法必须是大写开头，否则无法调用
	for i := 0; i < numMethod; i++ {
		m := rt.Method(i).Name
		switch m {
		case "Init", "Before", "After", "Finish", "NotFound", "ServerError":
		default:
			actions, method, isMethod := getActionsAndMethod(m)
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
					path := fmt.Sprintf("%s/%sfor%s", controllerPath, action, method)
					if _, ok := routeMapMethod[path]; ok {
						panic(path + "has exist")
					}
					routeMapMethod[path] = value
					logger.Infof("pattern: %s register in routeMapMethod: %s", pattern, path)
				} else {
					path := fmt.Sprintf("%s/%s", controllerPath, action)
					if _, ok := routeMap[path]; ok {
						panic(path + "has exist")
					}
					routeMap[path] = value
					logger.Infof("pattern: %s register in routeMap: %s", pattern, path)
				}
			}
		}
	}

	return
}

//HandlerFunc register HandlerFunc
func HandlerFunc(pattern string, h http.HandlerFunc) {
	logger.Infof("HandlerFunc: %s", pattern)
	if pattern == "/" || pattern == "/logger/adjust" {
		panic("http: multiple registrations for " + pattern)
	}
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		httpCtx := initCtx(w, r)
		defer httpCtx.Cancel()

		prometheus.RequestsTotal(r.URL.Path, r.Method)
		defer func(path, method string, startTime time.Time) {
			costTime := time.Since(startTime)
			httpCtx.Mixf("Path:%s Method:%s CostTime:%s", path, method, costTime)
			prometheus.RequestsCosttime(path, method, costTime)
		}(r.URL.Path, r.Method, time.Now())

		onlineNum := atomic.AddUint32(&online, 1)
		httpCtx.Mixf("From:%s Path:%s Online:%d", r.RemoteAddr, r.URL.String(), onlineNum)
		defer func() {
			atomic.AddUint32(&online, ^uint32(0))
			if err := recover(); err != nil {
				if err == ErrStopRun {
					return
				}
				httpCtx.Fatal(err, string(common.GetStack()))
			}
		}()
		err := checkConcurrence(onlineNum)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			httpCtx.Warn(err)
			return
		}
		h(w, r)
	})
}

//Handle register Handle
func Handle(pattern string, h http.Handler) {
	HandlerFunc(pattern, h.ServeHTTP)
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

//调整logger的设置
func loggerAdjust(w http.ResponseWriter, r *http.Request) {
	logger.Info("change logger level to", r.FormValue("level"))
	logger.SetLevelStr(r.FormValue("level"))
}
