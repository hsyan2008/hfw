package ssh

import (
	"errors"
	"io"
	"net"
	"strings"

	"acln.ro/zerocopy"
	"github.com/hsyan2008/hfw"
)

type ForwardConfig struct {
	SSHConfig
	Inner map[string]ForwardIni

	Indirect map[string]ForwardConfig
}

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
	c      *SSH
	c2     *SSH
	lister net.Listener
}

func NewLocalForward(httpCtx *hfw.HTTPContext, forwardConfig ForwardConfig) (l *Forward, err error) {
	return NewForward(httpCtx, LOCAL, forwardConfig)
}

func NewRemoteForward(httpCtx *hfw.HTTPContext, forwardConfig ForwardConfig) (l *Forward, err error) {
	return NewForward(httpCtx, REMOTE, forwardConfig)
}

func NewForward(httpCtx *hfw.HTTPContext, t ForwardType, forwardConfig ForwardConfig) (l *Forward, err error) {
	if httpCtx == nil {
		return l, errors.New("nil ctx")
	}
	l = &Forward{
		httpCtx: httpCtx,
		t:       t,
	}

	l.c, err = NewSSH(forwardConfig.SSHConfig)
	if err != nil {
		return
	}
	defer l.c.Close()

	for _, fi := range forwardConfig.Inner {
		go l.BindAndAccept(l.c, fi)
	}

	for _, indirect := range forwardConfig.Indirect {
		c, err := l.c.DialRemote(indirect.SSHConfig)
		if err != nil {
			return nil, err
		}
		for _, fi := range indirect.Inner {
			go l.BindAndAccept(c, fi)
		}
	}

	<-l.httpCtx.Done()

	return
}

func (l *Forward) BindAndAccept(c *SSH, fi ForwardIni) {
	var lister net.Listener
	var err error
	if len(fi.Addr) != 0 && len(fi.Bind) != 0 {
		if !strings.Contains(fi.Bind, ":") {
			fi.Bind = ":" + fi.Bind
		}
		if l.t == LOCAL {
			lister, err = net.Listen("tcp", fi.Bind)
		} else if l.t == REMOTE {
			lister, err = c.Listen(fi.Bind)
		}
		if err != nil {
			l.httpCtx.Warn(err)
			return
		}
		defer lister.Close()
		l.httpCtx.Infof("Bind %s forward to %s success, start to accept", lister.Addr().String(), fi.Addr)
		l.Accept(c, lister, fi)
	}
}

func (l *Forward) Accept(c *SSH, listen net.Listener, fi ForwardIni) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			l.Close()
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				l.httpCtx.Error(l.t, err)
			}
			return
		}
		go l.Hand(c, conn, fi)
	}
}

func (l *Forward) Hand(c *SSH, conn net.Conn, fi ForwardIni) {
	var err error
	var con net.Conn
	if l.t == LOCAL {
		con, err = c.Connect(fi.Addr)
	} else {
		con, err = net.Dial("tcp", fi.Addr)
	}
	if err != nil {
		l.httpCtx.Error(err)
		return
	}

	go multiCopy(conn, con)
	go multiCopy(con, conn)
}

func (l *Forward) Close() {
	l.httpCtx.Cancel()
	l.c.Close()
}

func multiCopy(des, src net.Conn) {
	defer func() {
		_ = src.Close()
		_ = des.Close()
	}()

	_, _ = zerocopy.Transfer(des, src)
}
