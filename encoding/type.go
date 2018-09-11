package structs

import (
	"strconv"
	"strings"
)

//把数字也解析成字符串
type Str string

func (a *Str) UnmarshalJSON(data []byte) (err error) {
	s, err := strconv.Unquote(strings.Replace(string(data), "\\/", "/", -1))
	if err == nil {
		//如果本来是字符串
		*a = Str(s)
	} else {
		//如果本来就是数字
		*a = Str(data)
		err = nil
	}

	return
}

//把纯数字的字符串也解析成数字
type Int int

func (a *Int) UnmarshalJSON(data []byte) (err error) {
	s, err := strconv.Unquote(strings.Replace(string(data), "\\/", "/", -1))

	var as int
	if err == nil {
		//如果本来是字符串
		as, err = strconv.Atoi(s)
	} else {
		//如果本来就是数字
		as, err = strconv.Atoi(string(data))
	}

	if err != nil {
		return err
	}
	*a = Int(as)

	return
}
