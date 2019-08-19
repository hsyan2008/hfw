package hfw

import (
	"fmt"
	"reflect"
	"strings"

	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/encoding"
)

type instance struct {
	reflectVal     reflect.Value
	controllerName string
	//方法名字
	methodName string
}

var (
	routeMap         = make(map[string]*instance)
	routeMapMethod   = make(map[string]*instance)
	routeMapRegister = make(map[string]string)
	routeInit        bool
	defaultInstance  *instance
)

//controller如果有下划线，可以直接在注册的时候指定
//action的下划线，可以自动处理
func findInstanceByPath(httpCtx *HTTPContext) (instance *instance, action string) {
	var ok bool

	defer func() {
		httpCtx.Controller = instance.controllerName
		httpCtx.Action = action
	}()

	var inputPath string
	if httpCtx.Path == "" {
		inputPath = httpCtx.Request.URL.Path
	} else {
		inputPath = httpCtx.Path
	}

	//假设url上没有action
	controllerPath := completeURL(inputPath)
	httpCtx.Action = Config.Route.DefaultAction
	httpCtx.Path = fmt.Sprintf("%s/%s", controllerPath, httpCtx.Action)
	httpCtx.Log().Warn(controllerPath, httpCtx.Action, httpCtx.Path)
	if instance, ok = routeMapMethod[httpCtx.Path+"for"+httpCtx.Request.Method]; ok {
		return instance, instance.methodName
	}
	if instance, ok = routeMap[httpCtx.Path]; ok {
		return instance, instance.methodName
	}

	//假设url上最后一段是action
	tmp := strings.Split(controllerPath, "/")
	controllerPath = strings.Join(tmp[:len(tmp)-1], "/")
	httpCtx.Action = tmp[len(tmp)-1]
	httpCtx.Path = fmt.Sprintf("%s/%s", controllerPath, httpCtx.Action)
	httpCtx.Log().Warn(controllerPath, httpCtx.Action, httpCtx.Path)
	if instance, ok = routeMapMethod[httpCtx.Path+"for"+httpCtx.Request.Method]; ok {
		return instance, instance.methodName
	}
	if instance, ok = routeMap[httpCtx.Path]; ok {
		return instance, instance.methodName
	}

	if defaultInstance == nil {
		panic("no default route find")
	}

	httpCtx.Action = strings.ToLower("NotFound")

	return defaultInstance, "NotFound"
}

func completeURL(url string) string {
	//去掉前缀并把url补全为2段
	trimURL := strings.Trim(strings.ToLower(url), "/")
	if trimURL == "" {
		trimURL = Config.Route.DefaultController
	}

	return trimURL
}

//actions包含小写和下划线两种格式的方法名，已去重
func getActionsAndMethod(funcName string) (actions []string, method string, isMethod bool) {
	if len(funcName) == 0 {
		return
	}
	action := funcName
	tmp := strings.Split(funcName, "For")
	tmpLen := len(tmp)
	if tmpLen > 1 {
		method = tmp[tmpLen-1]
		switch method {
		case "OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE", "TRACE", "CONNECT":
			isMethod = true
			action = strings.TrimSuffix(funcName, fmt.Sprintf("For%s", method))
		default:
			method = ""
		}
	}

	actions = append(actions, strings.ToLower(action))
	snakeAction := encoding.Snake(action)
	if actions[0] != snakeAction {
		actions = append(actions, snakeAction)
	}

	return
}

//修改httpCtx.Path后重新寻找执行action
func DispatchRoute(httpCtx *HTTPContext) {
	//这里httpCtx的Controller和Action是传过来的c和m，在findInstanceByPath会修改成正确的
	httpCtx.Path = fmt.Sprintf("C:%s M:%s", httpCtx.Controller, httpCtx.Action)
	instance, action := findInstanceByPath(httpCtx)
	logger.Debugf("Dispatch %s -> Call: %s/%s", httpCtx.Path, httpCtx.Controller, httpCtx.Action)
	reflectVal := instance.reflectVal
	//初始化httpCtx
	initValue := []reflect.Value{
		reflect.ValueOf(httpCtx),
	}
	reflectVal.MethodByName("Before").Call(initValue)
	defer reflectVal.MethodByName("After").Call(initValue)
	reflectVal.MethodByName(action).Call(initValue)
}
