package register

import "time"

type RegisterInfo struct {
	Host           string
	Port           int
	ServiceName    string
	UpdateInterval time.Duration
}

type Register interface {
	Register(info RegisterInfo) error
	UnRegister() error
}
