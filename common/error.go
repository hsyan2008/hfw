package common

import (
	"fmt"
	"runtime"
	"strings"
)

type RespErr struct {
	file   string
	line   int
	errNo  int64
	errMsg string
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
	return respErr.errMsg
}

func (respErr *RespErr) Error() string {
	return respErr.String()
}

func (respErr *RespErr) String() string {
	if respErr == nil {
		return ""
	}
	return fmt.Sprintf("[RespErr] File:%s Line:%d No:%d Msg:%s",
		respErr.file, respErr.line, respErr.errNo, respErr.errMsg)
}

func NewRespErr(errNo int64, i interface{}) (respErr *RespErr) {
	respErr = &RespErr{
		errNo:  errNo,
		errMsg: fmt.Sprintf("%v", i),
	}
	_, respErr.file, respErr.line, _ = runtime.Caller(1)
	respErr.file = strings.Replace(respErr.file, GetAppPath(), "", -1)

	return
}
