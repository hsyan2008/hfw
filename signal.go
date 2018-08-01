// 信号处理
//kill -INT pid 终止
//kill -TERM pid 重启
//需要调用Wg.Add()
//需要监听Shutdown通道
package hfw

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hsyan2008/go-logger/logger"
)

type SignalContext struct {
	IsHTTP bool `json:"-"`
	//Wg 业务方调用此变量注册工作
	Wg *sync.WaitGroup `json:"-"`
	//done 业务方调用Shutdowned函数获取所有任务已经退出的通知
	done chan bool

	//Shutdown 业务方手动监听此通道获知通知
	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`
}

var signalContext *SignalContext

func init() {
	signalContext = &SignalContext{
		Wg:   new(sync.WaitGroup),
		done: make(chan bool),
	}
	signalContext.Ctx, signalContext.Cancel = context.WithCancel(context.Background())
}

//GetSignalContext 一般用于其他包或者非http程序
func GetSignalContext() *SignalContext {
	return signalContext
}

//gracehttp外，增加两个信号支持
func (ctx *SignalContext) listenSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	logger.Infof("Exec `kill -INT %d` will graceful exit", PID)
	if ctx.IsHTTP {
		logger.Infof("Exec `kill -TERM %d` will graceful restart", PID)
	}
	s := <-c
	logger.Info("recv signal:", s)
	go ctx.doShutdownDone()
	if ctx.IsHTTP {
		logger.Info("Stopping http server")
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
		logger.Info("Stopping console server")
		//暂时不做重启
	}
}

func (ctx *SignalContext) doShutdownDone() {
	logger.Info("doShutdownDone start.")
	defer logger.Info("doShutdownDone done.")

	go ctx.waitDone()

	if randPortListener != nil {
		_ = randPortListener.Close()
	}

	timeout := 30
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		logger.Warnf("doShutdownDone %ds timeout", timeout)
		close(ctx.done)
	case <-ctx.done:
	}
}

//通知业务方，并等待业务方结束
func (ctx *SignalContext) waitDone() {
	//context包来取消，以通知业务方
	ctx.Cancel()
	//等待业务方完成退出
	ctx.WgWait()
	//表示全部完成
	close(ctx.done)
}

//Shutdowned 获取是否已经全部结束，暂时只有run.go里用到
func (ctx *SignalContext) Shutdowned() {
	<-ctx.done
}

func (ctx *SignalContext) WgAdd() {
	ctx.Wg.Add(1)
}

func (ctx *SignalContext) WgDone() {
	ctx.Wg.Done()
}

func (ctx *SignalContext) WgWait() {
	ctx.Wg.Wait()
}
