package common

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/axgle/mahonia"
	"github.com/google/uuid"
)

//Response ..
type Response struct {
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

//IsDir ...
func IsDir(filepath string) bool {
	f, err := os.Stat(filepath)
	if err != nil {
		return false
	}
	return f.IsDir()
}

//转换为当前操作系统支持的编码
//linux和mac为utf8
//win为GBK
func ToOsCode(text string) string {
	if runtime.GOOS == "windows" {
		enc := mahonia.NewEncoder(("gbk"))
		return enc.ConvertString(text)
	}

	return text
}

func Uuid() string {
	if id, err := uuid.NewRandom(); err == nil {
		return id.String()
	}

	return ""
}

//获取客户端ip
func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if strings.Contains(ip, "127.0.0.1") || ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}

	if ip == "" {
		return r.RemoteAddr
	}

	return ip
}

func ConvertToInt(v interface{}) int {
	switch tmp := v.(type) {
	case uint8:
		return int(tmp)
	case uint16:
		return int(tmp)
	case uint32:
		return int(tmp)
	case uint64:
		return int(tmp)
	case uint:
		return int(tmp)
	case int8:
		return int(tmp)
	case int16:
		return int(tmp)
	case int32:
		return int(tmp)
	case int64:
		return int(tmp)
	case int:
		return tmp
	}

	return 0
}

//把中文转成\u8981之类的unicode编码
//参考http://blog.cyeam.com/json/2014/08/04/go_json
//utf8包
func UtfToUnicode(d []byte) (reader *bytes.Buffer, err error) {
	reader = new(bytes.Buffer)
	for len(d) > 0 {
		r, size := utf8.DecodeRune(d)
		rint := int(r)
		if rint < 128 {
			_, err = reader.WriteRune(r)
			if err != nil {
				return nil, err
			}
		} else {
			_, err = reader.WriteString(fmt.Sprintf("%s%04x", `\u`, r))
			if err != nil {
				return nil, err
			}
		}
		d = d[size:]
	}

	return
}

//用于打印panic时的堆栈
func GetStack() []byte {
	buf := make([]byte, 1<<12) //16kb
	num := runtime.Stack(buf, false)

	return buf[:num]
}
