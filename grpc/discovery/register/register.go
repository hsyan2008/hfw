package register

type RegisterInfo struct {
	Host           string
	Port           int
	ServiceName    string
	UpdateInterval int64
}

type Register interface {
	Register(info RegisterInfo) error
	UnRegister() error
}
