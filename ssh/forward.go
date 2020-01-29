package ssh

import (
	"errors"
	"net"
	"strings"

	"acln.ro/zerocopy"
	"github.com/hsyan2008/hfw"
)

type ForwardIni struct {
	Addr string `toml:"addr"`
	Bind string `toml:"bind"`
}

type ForwardType uint8

const (
	LOCAL  ForwardType = 1
	REMOTE ForwardType = 2
)

type Forward struct {
	httpCtx *hfw.HTTPContext

	t      ForwardType
	fi     *ForwardIni
	c      *SSH
	c2     *SSH
	step   uint8
	lister net.Listener

	close chan struct{}
}

func NewLocalForward(httpCtx *hfw.HTTPContext, sshConfig SSHConfig, fi *ForwardIni) (l *Forward, err error) {
	return NewForward(httpCtx, LOCAL, sshConfig, fi)
}

func NewRemoteForward(httpCtx *hfw.HTTPContext, sshConfig SSHConfig, fi *ForwardIni) (l *Forward, err error) {
	return NewForward(httpCtx, REMOTE, sshConfig, fi)
}

func NewForward(httpCtx *hfw.HTTPContext, t ForwardType, sshConfig SSHConfig, fi *ForwardIni) (l *Forward, err error) {
	if httpCtx == nil {
		httpCtx = hfw.NewHTTPContext()
	}
	l = &Forward{
		httpCtx: httpCtx,
		step:    1,
		t:       t,
		close:   make(chan struct{}),
	}

	l.c, err = NewSSH(sshConfig)
	if err == nil && fi != nil {
		err = l.Bind(fi)
	}

	return
}

func (l *Forward) Dial(sshConfig SSHConfig, fi *ForwardIni) (err error) {
	l.step++
	if l.step == 2 {
		l.c2, err = l.c.DialRemote(sshConfig)
		if err == nil && fi != nil {
			err = l.Bind(fi)
		}
	}

	return
}

func (l *Forward) Bind(fi *ForwardIni) (err error) {
	if fi != nil && len(fi.Addr) != 0 && len(fi.Bind) != 0 {
		if !strings.Contains(fi.Bind, ":") {
			fi.Bind = ":" + fi.Bind
		}
		l.fi = fi
		if l.t == LOCAL {
			l.lister, err = net.Listen("tcp", l.fi.Bind)
		} else if l.t == REMOTE {
			l.lister, err = l.c.Listen(l.fi.Bind)
		}
		if err == nil {
			l.httpCtx.Infof("Bind %s forward to %s success, start to accept", fi.Bind, fi.Addr)
			go l.Accept()
		}
	} else {
		return errors.New("Err ForwardIni")
	}

	return
}
func (l *Forward) Accept() {
	for {
		select {
		case <-l.close:
			return
		default:
			conn, err := l.lister.Accept()
			if err != nil {
				l.httpCtx.Error(l.t, err)
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				continue
			}
			go l.Hand(conn)
		}
	}
}

func (l *Forward) Hand(conn net.Conn) {
	var err error
	var con net.Conn
	if l.t == LOCAL {
		con, err = l.c.Connect(l.fi.Addr)
	} else {
		con, err = net.Dial("tcp", l.fi.Addr)
	}
	if err != nil {
		l.httpCtx.Error(err)
		return
	}

	go multiCopy(conn, con)
	go multiCopy(con, conn)
}

func (l *Forward) Close() {
	close(l.close)

	_ = l.lister.Close()
	if l.c2 != nil {
		l.c2.Close()
	}
	l.c.Close()
}

func multiCopy(des, src net.Conn) {
	defer func() {
		_ = src.Close()
		_ = des.Close()
	}()

	_, _ = zerocopy.Transfer(des, src)
}
