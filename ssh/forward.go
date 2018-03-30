package ssh

import (
	"io"
	"net"

	"github.com/hsyan2008/go-logger/logger"
)

type Local struct {
	close chan bool
	//本地地址端口
	bind string
	//远程地址端口
	action string
	c      *SSH
}

func NewLocal(sshConfig SSHConfig, bind, action string) (l *Local, err error) {
	l = &Local{
		bind:   bind,
		action: action,
	}

	l.c, err = NewSSH(sshConfig)

	return
}

func (l *Local) Do() (err error) {
	lister, err := net.Listen("tcp", l.bind)
	if err != nil {
		logger.Error(err)
		return err
	}
	defer func() {
		_ = lister.Close()
		l.c.Close()
	}()

	for {
		select {
		case <-l.close:
			return
		default:
			conn, err := lister.Accept()
			if err != nil {
				logger.Error(err)
				return err
			}

			go l.Hand(conn)
		}
	}
}

func (l *Local) Hand(conn net.Conn) {
	con, err := l.c.Connect(l.action)
	if err != nil {
		logger.Error(err)
		return
	}

	go multiCopy(conn, con)
	go multiCopy(con, conn)
}

func (l *Local) Close() {
	l.close <- true
}

type Remote struct {
}

func multiCopy(des, src net.Conn) {
	defer func() {
		_ = src.Close()
		_ = des.Close()
	}()

	_, _ = io.Copy(des, src)
}
