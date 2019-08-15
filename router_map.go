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
	methodName     string
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
func findInstance(httpCtx *HTTPContext) (instance *instance, action string) {
	httpCtx.Path = fmt.Sprintf("%s/%s", httpCtx.Controller, httpCtx.Action)

	var ok bool
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
	httpCtx.Path = fmt.Sprintf("%s/%s", httpCtx.Controller, httpCtx.Action)

	return defaultInstance, "NotFound"
}

//必须For+全大写结尾
//actions包含小写和下划线两种格式的方法名，已去重
func getRequestMethod(funcName string) (actions []string, method string, isMethod bool) {
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
	//httpCtx.Action会变化，所以先保存
	path := fmt.Sprintf("C:%s M:%s", httpCtx.Controller, httpCtx.Action)
	instance, action := findInstance(httpCtx)
	logger.Debugf("Dispatch %s -> Call: %s/%s", path, instance.controllerName, action)
	reflectVal := instance.reflectVal
	//初始化httpCtx
	initValue := []reflect.Value{
		reflect.ValueOf(httpCtx),
	}
	reflectVal.MethodByName("Before").Call(initValue)
	defer reflectVal.MethodByName("After").Call(initValue)
	reflectVal.MethodByName(action).Call(initValue)
}
