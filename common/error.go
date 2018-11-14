package common

import "fmt"

type RespErr struct {
	ErrNo  int64  `json:"err_no"`
	ErrMsg string `json:"err_msg"`
}

func (repErr *RespErr) Error() string {
	return repErr.ErrMsg
}

func (repErr *RespErr) String() string {
	return fmt.Sprintf("ErrNo:%d ErrMsg:%s", repErr.ErrNo, repErr.ErrMsg)
}

func NewRespErr(ErrNo int64, ErrMsg string) *RespErr {
	return &RespErr{
		ErrNo:  ErrNo,
		ErrMsg: ErrMsg,
	}
}
