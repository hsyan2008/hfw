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

var Wg = new(sync.WaitGroup)
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

func listenSignal() {
	c := make(chan os.Signal, 1)
	//syscall.SIGINT, syscall.SIGTERM，syscall.SIGUSR2已被gracehttp接管，前2者直接退出，后者重启
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGQUIT)
	// Block until a signal is received.
	s := <-c
	logger.Info("recv signal:", s)
	switch s {
	case syscall.SIGUSR1:
		wait()
		//给自己发信号，触发gracehttp重启
		_ = syscall.Kill(os.Getpid(), syscall.SIGUSR2)
	case syscall.SIGQUIT:
		wait()
		//给自己发信号，触发gracehttp退出
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	}
}

func wait() {
	logger.Info("start to shutdown")
	defer logger.Info("shutdown done")

	go sendNotice()

	c := make(chan bool, 1)
	go waitDone(c)

	select {
	case <-time.After(15 * time.Second):
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
