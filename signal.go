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

	logger "github.com/hsyan2008/go-logger"
)

type SignalContext struct {
	IsHTTP bool `json:"-"`
	//Wg 业务方调用此变量注册工作
	Wg *sync.WaitGroup `json:"-"`
	//done 业务方调用Shutdowned函数获取所有任务已经退出的通知
	done chan bool

	mu    *sync.Mutex
	doing bool

	//Shutdown 业务方手动监听此通道获知通知
	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`
}

var signalContext *SignalContext

func init() {
	signalContext = &SignalContext{
		Wg:   new(sync.WaitGroup),
		done: make(chan bool),
		mu:   new(sync.Mutex),
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
	signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

	logger.Infof("Exec `kill -INT %d` will graceful exit", PID)
	logger.Infof("Exec `kill -TERM %d` will graceful restart", PID)

	s := <-c
	logger.Info("recv signal:", s)
	go ctx.doShutdownDone()
	if ctx.IsHTTP {
		logger.Info("Stopping http server")
	} else {
		logger.Info("Stopping console server")
		switch s {
		case syscall.SIGHUP, syscall.SIGTERM:
			execSpec := &syscall.ProcAttr{
				Env:   os.Environ(),
				Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
			}
			_, _, err := syscall.StartProcess(os.Args[0], os.Args, execSpec)
			if err != nil {
				logger.Errorf("failed to forkexec: %v", err)
			}
		case syscall.SIGQUIT, syscall.SIGINT:
		}
	}
}

func (ctx *SignalContext) doShutdownDone() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.doing {
		return
	}
	ctx.doing = true

	logger.Info("doShutdownDone start.")
	defer logger.Info("doShutdownDone done.")

	go ctx.waitDone()

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
	logger.Info("signal ctx cancel")
	ctx.Cancel()
	//等待业务方完成退出
	logger.Info("signal ctx waitgroup wait done start")
	ctx.WgWait()
	//表示全部完成
	logger.Info("signal ctx waitgroup wait done end")
	close(ctx.done)
}

//Shutdowned 获取是否已经全部结束，暂时只有run.go里用到
func (ctx *SignalContext) Shutdowned() {
	go ctx.doShutdownDone()
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
