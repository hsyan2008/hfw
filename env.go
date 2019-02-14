package hfw

//ENVIRONMENT 环境
var ENVIRONMENT string

const (
	DEV  = "dev"
	TEST = "test"
	PROD = "prod"
)

func IsProdEnv() bool {
	return ENVIRONMENT == PROD
}

func IsTestEnv() bool {
	return ENVIRONMENT == TEST
}

func IsDevEnv() bool {
	return ENVIRONMENT == DEV
}
