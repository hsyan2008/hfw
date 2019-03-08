package hfw

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
