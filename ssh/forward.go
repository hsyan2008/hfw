package ssh

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/hsyan2008/go-logger/logger"
)

type ForwardIni struct {
	Addr string `toml:"addr"`
	Bind string `toml:"bind"`
}

type LocalForward struct {
	fi     ForwardIni
	c      *SSH
	close  chan bool
	lister net.Listener
}

func NewLocalForward(sshConfig SSHConfig, fi ForwardIni) (l *LocalForward, err error) {
	if fi.Bind == "" || fi.Addr == "" {
		return nil, errors.New("err ini")
	}
	if !strings.Contains(fi.Bind, ":") {
		fi.Bind = ":" + fi.Bind
	}
	l = &LocalForward{
		fi:    fi,
		close: make(chan bool),
	}

	l.c, err = NewSSH(sshConfig)

	if err == nil {
		err = l.Bind()
		if err == nil {
			go l.Accept()
		}
	}

	return
}

func (l *LocalForward) Bind() (err error) {
	l.lister, err = net.Listen("tcp", l.fi.Bind)
	return
}
func (l *LocalForward) Accept() {
	defer func() {
		_ = l.lister.Close()
		l.c.Close()
	}()

	for {
		select {
		case <-l.close:
			return
		default:
			conn, err := l.lister.Accept()
			if err != nil {
				logger.Error(err)
				continue
			}
			go l.Hand(conn)
		}
	}
}

func (l *LocalForward) Hand(conn net.Conn) {
	con, err := l.c.Connect(l.fi.Addr)
	if err != nil {
		logger.Error(err)
		return
	}

	go multiCopy(conn, con)
	go multiCopy(con, conn)
}

func (l *LocalForward) Close() {
	close(l.close)
}

type RemoteForward struct {
}

func multiCopy(des, src net.Conn) {
	defer func() {
		_ = src.Close()
		_ = des.Close()
	}()

	_, _ = io.Copy(des, src)
}
