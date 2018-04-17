package stack

import (
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hsyan2008/go-logger/logger"
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

	logger.Info("Stack is enable, you can do `kill -USR1", os.Getpid(), "; tail", stdFile, "` to view")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
	go func() {
	FOR:
		for s := range c {
			switch s {
			case syscall.SIGUSR1:
				dumpStacks()
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
	defer func() {
		_ = fd.Close()
	}()

	_, _ = fd.WriteString(strings.Repeat("=", 80) + "\n\n")
	_, _ = fd.WriteString(time.Now().Format(timeFormat))
	_, _ = fd.WriteString(" total goroutine:" + strconv.Itoa(runtime.NumGoroutine()))
	_, _ = fd.WriteString(" stdout:" + "\n\n")
	_, _ = fd.Write(buf)
	_, _ = fd.WriteString("\n\n")
}
