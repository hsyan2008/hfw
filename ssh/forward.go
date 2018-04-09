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
		fi: fi,
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
	for {
		conn, err := l.lister.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.Error(err)
			continue
		}
		go l.Hand(conn)
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
	_ = l.lister.Close()
	l.c.Close()
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
