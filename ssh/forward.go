package ssh

import (
	"errors"
	"io"
	"net"
	"strings"

	logger "github.com/hsyan2008/go-logger"
)

type ForwardIni struct {
	Addr string `toml:"addr"`
	Bind string `toml:"bind"`
}

type LocalForward struct {
	fi     ForwardIni
	c      *SSH
	c2     *SSH
	step   uint8
	lister net.Listener
}

func NewLocalForward(sshConfig SSHConfig) (l *LocalForward, err error) {
	l = &LocalForward{
		step: 1,
	}

	l.c, err = NewSSH(sshConfig)

	return
}

func (l *LocalForward) Dial(sshConfig SSHConfig) (err error) {
	l.step++
	if l.step == 2 {
		l.c2, err = l.c.DialRemote(sshConfig)
	}

	return
}

func (l *LocalForward) Bind(fi ForwardIni) (err error) {
	if len(fi.Addr) != 0 && len(fi.Bind) != 0 {
		if !strings.Contains(fi.Bind, ":") {
			fi.Bind = ":" + fi.Bind
		}
		l.fi = fi
		l.lister, err = net.Listen("tcp", l.fi.Bind)
		if err == nil {
			logger.Infof("Bind %s forward to %s success, start to accept", fi.Bind, fi.Addr)
			go l.Accept()
		}
	} else {
		return errors.New("Err ForwardIni")
	}

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
	if l.c2 != nil {
		l.c2.Close()
	}
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
