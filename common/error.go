package common

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

var errorMap = map[int64]string{
	400: "request error",
	500: "system error",
}

func SetErrorMap(m map[int64]string) {
	for k, v := range m {
		errorMap[k] = v
	}
}

func AddErrorMap(errNo int64, errMsg string) {
	errorMap[errNo] = errMsg
}

func GetErrorMap(errNo int64) string {
	return errorMap[errNo]
}

type RespErr struct {
	file  string
	line  int
	errNo int64
	err   error
}

func (respErr *RespErr) ErrNo() int64 {
	if respErr == nil {
		return 0
	}
	return respErr.errNo
}

func (respErr *RespErr) ErrMsg() string {
	if respErr == nil {
		return ""
	}
	return respErr.err.Error()
}

func (respErr *RespErr) Err() error {
	if respErr == nil {
		return nil
	}
	return respErr.err
}

func (respErr *RespErr) Error() string {
	return respErr.String()
}

func (respErr *RespErr) String() string {
	if respErr == nil {
		return ""
	}
	return fmt.Sprintf("[RespErr %s:%d N:%d M:%s]",
		respErr.file, respErr.line, respErr.ErrNo(), respErr.ErrMsg())
}

//记录调用本函数的位置
func NewRespErr(errNo int64, i interface{}) (respErr *RespErr) {
	if errNo == 0 || i == nil {
		return nil
	}
	if r, ok := i.(*RespErr); ok && r != nil {
		r.errNo = errNo
		return r
	}

	respErr = &RespErr{
		errNo: errNo,
	}

	switch e := i.(type) {
	case error:
		respErr.err = e
	case string:
		respErr.err = errors.New(e)
	default:
		respErr.err = errors.New(fmt.Sprintf("%v", i))
	}

	respErr.file, respErr.line = GetCaller(1)

	return
}

func GetCaller(depth int) (file string, line int) {
	_, file, line, _ = runtime.Caller(depth + 1)
	if GOPATH != "" {
		file = strings.ReplaceAll(file, GOPATH, "")
	}

	return
}
