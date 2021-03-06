package stack

import (
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	logger "github.com/hsyan2008/go-logger"
)

// kill -USR1 pid
// tail file

const (
	timeFormat = "2006-01-02 15:04:05"
)

var (
	stdFile string
)

func SetupStack(file string) {

	stdFile = file

	logger.Infof("Stack enable, Exec `kill -TRAP %d; tail -f %s` to view", os.Getpid(), stdFile)

	c := make(chan os.Signal, 1)
	//win支持的信号参考/usr/lib64/go/src/syscall/types_windows.go
	signal.Notify(c, syscall.SIGTRAP)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	go func() {
	FOR:
		for s := range c {
			switch s {
			case syscall.SIGTRAP:
				dumpStacks()
				logger.Info("Stack already dumped to", stdFile)
			default:
				break FOR
			}
		}
	}()
}

func dumpStacks() {
	buf := make([]byte, 1638400)
	buf = buf[:runtime.Stack(buf, true)]
	writeStack(buf)
}

func writeStack(buf []byte) {
	fd, _ := os.OpenFile(stdFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	defer fd.Close()

	_, _ = fd.WriteString(strings.Repeat("=", 80) + "\n\n")
	_, _ = fd.WriteString(time.Now().Format(timeFormat))
	_, _ = fd.WriteString(" total goroutine:" + strconv.Itoa(runtime.NumGoroutine()))
	_, _ = fd.WriteString(" stdout:" + "\n\n")
	_, _ = fd.Write(buf)
	_, _ = fd.WriteString("\n\n")
}
