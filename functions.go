package hfw

import (
	"crypto/md5"
	"fmt"
	"os"
)

//Result ..
type Result struct {
	ErrNo   int64       `json:"err_no"`
	ErrMsg  string      `json:"err_msg"`
	Results interface{} `json:"results"`
}

//Max ..
func Max(i int, j ...int) int {
	for _, v := range j {
		if v > i {
			i = v
		}
	}
	return i
}

//Min ..
func Min(i int, j ...int) int {
	for _, v := range j {
		if v < i {
			i = v
		}
	}
	return i
}

//Md5 ..
func Md5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

//IsExist ...
func IsExist(filepath string) bool {
	_, err := os.Stat(filepath)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}
