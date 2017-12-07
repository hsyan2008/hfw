// 信号处理
// 只支持USR1和QUIT
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

//Wg 业务方调用此变量注册工作
var Wg = new(sync.WaitGroup)

//Shutdown 业务方监听此通道获知通知
var Shutdown = make(chan bool, 3)

func init() {
	go listenSignal()
}

//通知业务方
func sendNotice() {
	for {
		Shutdown <- true
	}
}

//暂时保留，建议不用
func listenSignal() {
	c := make(chan os.Signal, 1)
	//syscall.SIGINT, syscall.SIGTERM，syscall.SIGUSR2已被gracehttp接管，前2者直接退出，后者重启
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGQUIT)
	// Block until a signal is received.
	s := <-c
	logger.Info("recv signal:", s)
	switch s {
	case syscall.SIGUSR1:
		waitShutdownDone()
		//给自己发信号，触发gracehttp重启
		_ = syscall.Kill(os.Getpid(), syscall.SIGUSR2)
	case syscall.SIGQUIT:
		waitShutdownDone()
		//给自己发信号，触发gracehttp退出
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	}
}

func waitShutdownDone() {
	logger.Info("start to shutdown")
	defer logger.Info("shutdown done")

	go sendNotice()

	c := make(chan bool, 1)
	go waitDone(c)

	select {
	case <-time.After(10 * time.Second):
		logger.Warn("waitShutdownDone 10s timeout")
		return
	case <-c:
		return
	}
}

//等待业务方结束
func waitDone(c chan bool) {
	Wg.Wait()
	c <- true
}
