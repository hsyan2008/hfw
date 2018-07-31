package ssh

import (
	"errors"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/encoding"
	"golang.org/x/crypto/ssh"
)

//SSHConfig ..
type SSHConfig struct {
	Id       string        `toml:"id"`
	Addr     string        `toml:"addr"`
	User     string        `toml:"user"`
	Auth     string        `toml:"auth"`
	Phrase   string        `toml:"phrase"`
	Timeout  time.Duration `toml:"timeout"`
	SkipKeep bool          `toml:"skipKeep"`
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

	close chan bool
	timer *time.Timer

	mt *sync.Mutex
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
	if ins, ok = sshIns[key]; ok {
		ins.mt.Lock()
		defer ins.mt.Unlock()
		if ins.ref > 0 {
			defer mt.Unlock()
			ins.ref += 1
			return ins, err
		}
	} else {
		ins = &SSH{
			ref:   0,
			close: make(chan bool),
			m:     NormalSSHMode,
			mt:    new(sync.Mutex),
		}
		ins.SetConfig(sshConfig)
		ins.timer = time.NewTimer(ins.config.Timeout * time.Second)
		sshIns[key] = ins

		ins.mt.Lock()
		defer ins.mt.Unlock()
	}

	//不用defer，是防止Dial阻塞并发
	mt.Unlock()

	if ins.ref > 0 {
		ins.ref += 1
		return
	}

	err = ins.Dial()
	if err == nil {
		ins.ref += 1
	}

	return
}

//到0后，保留连接
func (this *SSH) Close() {

	this.mt.Lock()
	defer this.mt.Unlock()

	this.ref -= 1

	if this.ref <= 0 {
		this.close <- true
		_ = this.c.Close()
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
		go this.keepalive()
	}

	return
}

func (this *SSH) dial() (c *ssh.Client, err error) {
	return ssh.Dial("tcp", this.config.Addr, this.getSshClientConfig())
}

//DialRemote 通过跳板连接其他服务器
func (this *SSH) DialRemote(sshConfig SSHConfig) (ins *SSH, err error) {

	if sshConfig.Addr == "" {
		return nil, errors.New("err sshConfig")
	}

	ins = &SSH{
		ref:    1,
		close:  make(chan bool),
		m:      RemoteSSHMode,
		mt:     new(sync.Mutex),
		preIns: this,
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

	conn, nc, req, err := ssh.NewClientConn(rc, "", this.getSshClientConfig())
	if err != nil {
		return
	}

	return ssh.NewClient(conn, nc, req), nil
}

func (this *SSH) Connect(addr string) (conn net.Conn, err error) {
	return this.c.Dial("tcp", addr)
}

func (this *SSH) Config() SSHConfig {
	return this.config
}

func (this *SSH) SetConfig(sshConfig SSHConfig) {
	if sshConfig.Timeout == 0 {
		sshConfig.Timeout = 30
	}

	this.config = sshConfig
}

func (this *SSH) getSshClientConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: this.config.User,
		Auth: []ssh.AuthMethod{
			this.getAuth(),
		},
		//如果没有这个，会提示需要know_hosts文件
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         this.config.Timeout * time.Second,
	}
}

func (this *SSH) getAuth() ssh.AuthMethod {
	//是文件
	var key []byte
	var err error
	auth := this.config.Auth
	phrase := this.config.Phrase

	if common.IsExist(auth) {
		key, _ = ioutil.ReadFile(auth)
	}

	//密码
	if len(key) == 0 {
		if len(auth) < 50 {
			return ssh.Password(auth)
		}
		key = []byte(auth)
	}

	var signer ssh.Signer
	if phrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(phrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		panic("err private key:" + err.Error())
	}
	return ssh.PublicKeys(signer)
}

func (this *SSH) keepalive() {
	//因为jumpserver.org的问题，无法检测，所以不检测
	if this.config.SkipKeep {
		return
	}
	for {
		this.timer.Reset(this.config.Timeout * time.Second)
		select {
		case <-this.close:
			this.timer.Stop()
			return
		case <-this.timer.C:
			go this.keep()
		}
	}
}

func (this *SSH) keep() {
	err := this.Check()
	if err != nil {
		this.mt.Lock()
		defer this.mt.Unlock()
		if this.ref <= 0 {
			//已关闭,退出
			return
		}

		switch this.m {
		case NormalSSHMode:
			_ = this.c.Close()
			this.c, err = this.dial()
		case RemoteSSHMode:
			_ = this.c.Close()
			this.c, err = this.dialRemote()
		default:
			logger.Debug("error sshMode")
		}
		if err != nil {
			logger.Warn(err)
			this.timer.Reset(0)
		}
	}
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
