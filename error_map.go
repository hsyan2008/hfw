package hfw

var errorMap map[int64]string

func SetErrorMap(m map[int64]string) {
	errorMap = m
}

func AddErrorMap(errNo int64, errMsg string) {
	errorMap[errNo] = errMsg
}

func GetErrorMap(errNo int64) string {
	return errorMap[errNo]
}
