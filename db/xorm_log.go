package db

import (
	"fmt"

	"github.com/go-xorm/core"
	logger "github.com/hsyan2008/go-logger"
)

type xormLog struct {
	isShowSQL bool
}

func (this *xormLog) Debug(v ...interface{}) {
	logger.Output(4, "DEBUG", v...)
}

func (this *xormLog) Debugf(format string, v ...interface{}) {
	logger.Output(4, "DEBUG", fmt.Sprintf(format, v...))
}

func (this *xormLog) Info(v ...interface{}) {
	logger.Output(4, "INFO", v...)
}

func (this *xormLog) Infof(format string, v ...interface{}) {
	logger.Output(4, "INFO", fmt.Sprintf(format, v...))
}

func (this *xormLog) Warn(v ...interface{}) {
	logger.Output(4, "WARN", v...)
}

func (this *xormLog) Warnf(format string, v ...interface{}) {
	logger.Output(4, "WARN", fmt.Sprintf(format, v...))
}

func (this *xormLog) Error(v ...interface{}) {
	logger.Output(4, "ERROR", v...)
}

func (this *xormLog) Errorf(format string, v ...interface{}) {
	logger.Output(4, "ERROR", fmt.Sprintf(format, v...))
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
