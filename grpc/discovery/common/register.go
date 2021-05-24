package common

type RegisterInfo struct {
	Host           string
	Port           int
	ServerName     string
	ServerId       string
	UpdateInterval int64
	Tags           []string
}

type Register interface {
	Register(info RegisterInfo) error
	UnRegister() error
}
