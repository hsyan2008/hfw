package hfw

import (
	"fmt"
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
	if len(httpCtx.Path) == 0 {
		httpCtx.Path = fmt.Sprintf("%s/%s", httpCtx.Controller, httpCtx.Action)
	}

	if len(httpCtx.Path) > 0 {
		var ok bool
		if instance, ok = routeMapMethod[httpCtx.Path+"for"+strings.ToLower(httpCtx.Request.Method)]; ok {
			return instance, instance.methodName
		}

		if instance, ok = routeMap[httpCtx.Path]; ok {
			return instance, instance.methodName
		}
	}

	//取现有的第一个作为默认
	for _, instance = range routeMap {
		return instance, "NotFound"
	}

	for _, instance = range routeMapMethod {
		return instance, "NotFound"
	}

	panic("no route find")
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
