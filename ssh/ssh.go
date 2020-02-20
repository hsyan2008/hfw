package ssh

import (
	"errors"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/hsyan2008/hfw"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/encoding"
	"golang.org/x/crypto/ssh"
)

//SSHConfig ..
type SSHConfig struct {
	Id   string `toml:"id"`
	Addr string `toml:"addr"`
	User string `toml:"user"`
	//证书和密码，可以共存
	Certs     []string      `toml:"certs"`
	Passwords []string      `toml:"passwords"`
	Timeout   time.Duration `toml:"timeout"`
	SkipKeep  bool          `toml:"skipKeep"`
	//以下两个是兼容
	Auth   string `toml:"auth"`
	Phrase string `toml:"phrase"`
}

type sshMode uint

const (
	//直连
	NormalSSHMode sshMode = iota
	//通过跳板机
	RemoteSSHMode
)

//SSH ..
type SSH struct {
	config SSHConfig
	ref    int

	m sshMode
	c *ssh.Client

	preIns *SSH

	timer *time.Timer

	mt *sync.Mutex

	httpCtx *hfw.HTTPContext
}

var mt = new(sync.Mutex)

var sshIns = make(map[string]*SSH)

//NewSSH 建立第一个ssh连接，一般是跳板机
func NewSSH(sshConfig SSHConfig) (ins *SSH, err error) {

	key, err := key(sshConfig)
	if err != nil {
		return
	}

	mt.Lock()
	var ok bool
	if ins, ok = sshIns[key]; !ok {
		ins = &SSH{
			ref:     0,
			m:       NormalSSHMode,
			mt:      new(sync.Mutex),
			httpCtx: hfw.NewHTTPContext(),
		}
		ins.SetConfig(sshConfig)
		ins.timer = time.NewTimer(ins.config.Timeout * time.Second)
		sshIns[key] = ins
	}

	ins.mt.Lock()
	defer ins.mt.Unlock()

	mt.Unlock()

	if ins.ref > 0 {
		ins.ref += 1
		return
	}

	err = ins.Dial()

	return
}

//到0后，保留连接
func (this *SSH) Close() {

	this.mt.Lock()
	defer this.mt.Unlock()

	this.ref -= 1

	if this.ref <= 0 {
		this.httpCtx.Cancel()
		if this.c != nil {
			_ = this.c.Close()
		}
	}
}

func key(sshConfig SSHConfig) (key string, err error) {
	gb, err := encoding.Gob.Marshal(sshConfig)
	if err != nil {
		return
	}
	key = common.Md5(string(gb))

	return
}

func (this *SSH) Dial() (err error) {

	if this.config.Addr == "" {
		return errors.New("err sshConfig")
	}

	this.c, err = this.dial()

	if err == nil {
		this.httpCtx.Info("dial success:", this.config.Addr, this.config.User)
		go this.keepalive()
		this.ref += 1
	} else {
		this.httpCtx.Warn("dial faild:", this.config.Addr, this.config.User, err)
	}

	return
}

func (this *SSH) dial() (c *ssh.Client, err error) {
	scc, err := this.getSshClientConfig()
	if err != nil {
		return
	}
	return ssh.Dial("tcp", this.config.Addr, scc)
}

//DialRemote 通过跳板连接其他服务器
func (this *SSH) DialRemote(sshConfig SSHConfig) (ins *SSH, err error) {

	if sshConfig.Addr == "" {
		return nil, errors.New("err sshConfig")
	}

	ins = &SSH{
		ref:     1,
		m:       RemoteSSHMode,
		mt:      new(sync.Mutex),
		preIns:  this,
		httpCtx: this.httpCtx,
	}
	ins.SetConfig(sshConfig)
	ins.timer = time.NewTimer(ins.config.Timeout * time.Second)

	ins.c, err = ins.dialRemote()

	if err == nil {
		go ins.keepalive()
	}

	return
}

func (this *SSH) dialRemote() (c *ssh.Client, err error) {
	rc, err := this.preIns.Connect(this.config.Addr)
	if err != nil {
		return
	}

	scc, err := this.getSshClientConfig()
	if err != nil {
		return
	}
	conn, nc, req, err := ssh.NewClientConn(rc, "", scc)
	if err != nil {
		return
	}

	return ssh.NewClient(conn, nc, req), nil
}

func (this *SSH) Connect(addr string) (conn net.Conn, err error) {
	if this.c == nil {
		return nil, errors.New("nil client")
	}
	return this.c.Dial("tcp", addr)
}

func (this *SSH) Listen(addr string) (l net.Listener, err error) {
	if this.c == nil {
		return nil, errors.New("nil client")
	}
	return this.c.Listen("tcp", addr)
}

func (this *SSH) ListenTCP(addr string) (l net.Listener, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	return this.c.ListenTCP(tcpAddr)
}

func (this *SSH) Config() SSHConfig {
	return this.config
}

func (this *SSH) SetConfig(sshConfig SSHConfig) {
	if sshConfig.Timeout == 0 {
		sshConfig.Timeout = 4 * 60
	}

	this.config = sshConfig
}

func (this *SSH) getSshClientConfig() (cc *ssh.ClientConfig, err error) {
	auth, err := this.getAuth()
	if err != nil {
		return
	}
	cc = &ssh.ClientConfig{
		User: this.config.User,
		Auth: auth,
		//如果没有这个，会提示需要know_hosts文件
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         this.config.Timeout * time.Second,
	}

	return
}

func (this *SSH) getAuth() (auths []ssh.AuthMethod, err error) {
	//是文件
	var key []byte

	for _, v := range this.config.Certs {
		if common.IsExist(v) {
			this.httpCtx.Info(this.config.Addr, "auth is file")
			key, _ = ioutil.ReadFile(v)
		} else {
			this.httpCtx.Info(this.config.Addr, "auth is key string")
			key = []byte(v)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	for _, v := range this.config.Passwords {
		this.httpCtx.Info(this.config.Addr, "auth is password")
		auths = append(auths, ssh.Password(v))
	}

	if this.config.Auth == "" {
		return
	}

	auth := this.config.Auth
	phrase := this.config.Phrase

	if common.IsExist(auth) {
		this.httpCtx.Info(this.config.Addr, "auth is file")
		key, err = ioutil.ReadFile(auth)
		if err != nil {
			return nil, err
		}
	}

	//密码
	if len(key) == 0 {
		if len(auth) < 50 {
			this.httpCtx.Info(this.config.Addr, "auth is password")
			auths = append(auths, ssh.Password(auth))
			return
		}
		this.httpCtx.Info(this.config.Addr, "auth is key string")
		key = []byte(auth)
	}

	var signer ssh.Signer
	if phrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(phrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		return nil, err
	}
	auths = append(auths, ssh.PublicKeys(signer))

	return
}

func (this *SSH) keepalive() {
	//因为jumpserver.org的问题，无法检测，所以不检测
	if this.config.SkipKeep || this.c == nil {
		return
	}
	for {
		select {
		case <-this.httpCtx.Ctx.Done():
			this.timer.Stop()
			return
		case <-this.timer.C:
			err := this.keep()
			if err != nil {
				this.timer.Reset(0)
			} else {
				this.timer.Reset(this.config.Timeout * time.Second)
			}
		}
	}
}

func (this *SSH) keep() (err error) {
	this.httpCtx.Debug(this.config.Addr, "ping start")
	err = this.Check()
	if err != nil {
		this.httpCtx.Info(this.config.Addr, "ping faild:", err)
		this.mt.Lock()
		defer this.mt.Unlock()
		if this.ref <= 0 {
			//已关闭,退出
			return
		}

		if this.c != nil {
			_ = this.c.Close()
		}

		switch this.m {
		case NormalSSHMode:
			this.c, err = this.dial()
		case RemoteSSHMode:
			this.c, err = this.dialRemote()
		default:
			err = errors.New("error sshMode")
		}
		if err != nil {
			this.httpCtx.Warn(this.config.Addr, "reconnect faild:", err)
		} else {
			this.httpCtx.Info(this.config.Addr, "reconnect success")
		}
	} else {
		this.httpCtx.Debug(this.config.Addr, "ping success")
	}

	return
}

func (this *SSH) Check() (err error) {
	if this.c == nil {
		return errors.New("Check no ins")
	}

	sess, err := this.c.NewSession()
	if err != nil {
		return
	}
	defer sess.Close()
	if err = sess.Shell(); err != nil {
		return
	}

	return sess.Wait()
}
