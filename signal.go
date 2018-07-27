// 信号处理
//kill -INT pid 终止
//kill -TERM pid 重启
//需要调用Wg.Add()
//需要监听Shutdown通道
package hfw

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hsyan2008/go-logger/logger"
)

type context struct {
	IsHTTP bool
	//Wg 业务方调用此变量注册工作
	Wg *sync.WaitGroup
	//Shutdown 业务方手动监听此通道获知通知
	Shutdown chan bool
	//Done 业务方调用Shutdowned函数获取已经完成shutdown的通知
	Done chan bool
}

var Ctx *context

func init() {
	Ctx = &context{
		Wg:       new(sync.WaitGroup),
		Shutdown: make(chan bool),
		Done:     make(chan bool),
	}
}

//gracehttp外，增加两个信号支持
func (ctx *context) listenSignal() {

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	logger.Infof("Do `kill -INT %d` will graceful exit", PID)
	if ctx.IsHTTP {
		logger.Infof("Do `kill -TERM %d` will graceful restart", PID)
	}
	s := <-c
	logger.Info("recv signal:", s)
	go ctx.waitShutdownDone()
	if ctx.IsHTTP {
		logger.Info("start to stop http")
		p, _ := os.FindProcess(os.Getpid())
		switch s {
		case syscall.SIGTERM:
			//给自己发信号，触发gracehttp重启
			_ = p.Signal(syscall.SIGHUP)
		case syscall.SIGINT:
			//给自己发信号，触发gracehttp退出
			_ = p.Signal(syscall.SIGQUIT)
		}
	} else {
		logger.Info("start to stop console")
		//暂时不做重启
	}
}

func (ctx *context) waitShutdownDone() {
	logger.Info("start to shutdown")
	defer logger.Info("shutdown done")

	go ctx.waitDone()

	if randPortListener != nil {
		_ = randPortListener.Close()
	}

	timeout := 30
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		logger.Warnf("waitShutdownDone %ds timeout", timeout)
		close(ctx.Done)
	case <-ctx.Done:
	}
}

//通知业务方，并等待业务方结束
func (ctx *context) waitDone() {
	//通知业务方
	close(ctx.Shutdown)
	//等待业务方完成退出
	ctx.Wg.Wait()
	//表示全部完成
	close(ctx.Done)
}

//Shutdowned 获取是否已经结束，暂时只有run.go里用到
func (ctx *context) Shutdowned() {
	<-ctx.Done
}

func (ctx *context) WgAdd() {
	ctx.Wg.Add(1)
}

func (ctx *context) WgDone() {
	ctx.Wg.Done()
}

func (ctx *context) WgWait() {
	ctx.Wg.Wait()
}
