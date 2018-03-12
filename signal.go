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

//shutdowned 业务方调用Shutdowned函数获取已经完成shutdown的通知
var shutdowned = make(chan bool, 3)

var isHttp = false

func init() {
	go listenSignal()
}

//通知业务方
func sendNotice() {
	for {
		Shutdown <- true
	}
}

//gracehttp外，增加两个信号支持
func listenSignal() {
	c := make(chan os.Signal, 1)
	//syscall.SIGINT, syscall.SIGTERM，syscall.SIGUSR2已被gracehttp接管，前2者直接退出，后者重启
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received.
	s := <-c
	logger.Info("recv signal:", s)
	go waitShutdownDone()
	if isHttp {
		logger.Info("start to stop http")
		p, _ := os.FindProcess(os.Getpid())
		switch s {
		case syscall.SIGTERM:
			//给自己发信号，触发gracehttp重启
			_ = p.Signal(syscall.SIGTERM)
		case syscall.SIGINT:
			//给自己发信号，触发gracehttp退出
			_ = p.Signal(syscall.SIGQUIT)
		}
	} else {
		logger.Info("start to stop console")
		//暂时不做重启
	}
}

func waitShutdownDone() {
	logger.Info("start to shutdown")
	defer logger.Info("shutdown done")

	go sendNotice()

	c := make(chan bool, 1)
	go waitDone(c)

	go func() {
		if randPortListener != nil {
			_ = randPortListener.Close()
		}
	}()

	select {
	case <-time.After(10 * time.Second):
		logger.Warn("waitShutdownDone 10s timeout")
	case <-c:
	}

	shutdowned <- true
}

//等待业务方结束
func waitDone(c chan bool) {
	Wg.Wait()
	c <- true
}

//
func Shutdowned() {
	<-shutdowned
}
