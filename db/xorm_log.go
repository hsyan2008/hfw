package db

import (
	"github.com/hsyan2008/go-logger"
	"xorm.io/core"
)

type xormLog struct {
	*logger.Logger
	isShowSQL bool
}

func newXormLog() *xormLog {
	return &xormLog{
		Logger: logger.NewLogger(),
	}
}

func (this *xormLog) Level() core.LogLevel {
	return core.LogLevel(logger.Level())
}

func (this *xormLog) SetLevel(l core.LogLevel) {
	logger.SetLevel(logger.LEVEL(l))
}

func (this *xormLog) ShowSQL(show ...bool) {
	this.isShowSQL = show[0]
}

func (this *xormLog) IsShowSQL() bool {
	return this.isShowSQL
}
