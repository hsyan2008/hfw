package common

type RegisterInfo struct {
	Host           string
	Port           int
	ServerName     string
	UpdateInterval int64
}

type Register interface {
	Register(info RegisterInfo) error
	UnRegister() error
}
