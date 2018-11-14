package common

import "fmt"

type RespErr struct {
	errNo  int64
	errMsg string
}

func (repErr *RespErr) ErrNo() int64 {
	if repErr == nil {
		return 0
	}
	return repErr.errNo
}

func (repErr *RespErr) ErrMsg() string {
	if repErr == nil {
		return ""
	}
	return repErr.errMsg
}

func (repErr *RespErr) Error() string {
	if repErr == nil {
		return ""
	}
	return repErr.errMsg
}

func (repErr *RespErr) String() string {
	if repErr == nil {
		return ""
	}
	return fmt.Sprintf("ErrNo:%d ErrMsg:%s", repErr.errNo, repErr.errMsg)
}

func NewRespErr(ErrNo int64, ErrMsg string) *RespErr {
	return &RespErr{
		errNo:  ErrNo,
		errMsg: ErrMsg,
	}
}
