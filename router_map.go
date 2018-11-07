package hfw

import (
	"reflect"
	"strings"

	logger "github.com/hsyan2008/go-logger"
)

type instance struct {
	reflectVal     reflect.Value
	controllerName string
	methodName     string
}

var routeMap = make(map[string]instance)
var routeMapMethod = make(map[string]instance)
var routeMapRegister = make(map[string]string)
var routeInit bool

func findInstance(httpCtx *HTTPContext) (instance instance, action string) {
	var ok bool
	if instance, ok = routeMapMethod[httpCtx.Path+"for"+strings.ToLower(httpCtx.Request.Method)]; !ok {
		if instance, ok = routeMap[httpCtx.Path]; !ok {
			//取默认的
			p := Config.Route.DefaultController + "/" + Config.Route.DefaultAction
			if instance, ok = routeMap[p]; !ok {
				//如果拿不到默认的，就取现有的第一个
				for _, instance = range routeMap {
					break
				}
			}
			return instance, "NotFound"
		}
	}

	return instance, instance.methodName
}

//修改httpCtx.Path后重新寻找执行action
func ReRouter(httpCtx *HTTPContext) {
	instance, action := findInstance(httpCtx)
	reflectVal := instance.reflectVal
	logger.Debugf("Query Path: %s -> Call: %s/%s", httpCtx.Request.URL.String(), instance.controllerName, action)
	//初始化httpCtx
	initValue := []reflect.Value{
		reflect.ValueOf(httpCtx),
	}
	reflectVal.MethodByName("Before").Call(initValue)
	defer reflectVal.MethodByName("After").Call(initValue)
	reflectVal.MethodByName(action).Call(initValue)
}
