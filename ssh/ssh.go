package ssh

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/encoding"
	"golang.org/x/crypto/ssh"
)

//SSHConfig ..
type SSHConfig struct {
	Id      string `toml:"id"`
	Addr    string `toml:"addr"`
	User    string `toml:"user"`
	Auth    string `toml:"auth"`
	Phrase  string `toml:"phrase"`
	Timeout time.Duration
}

type mode uint

const (
	//直连
	NormalMode = iota
	//通过跳板机
	RemoteMode
)

//SSH ..
type SSH struct {
	m      mode
	c      *ssh.Client
	close  chan bool
	config SSHConfig
	ref    int
}

var mt = new(sync.Mutex)

var sshIns = make(map[string]*SSH)

//NewSSH 建立第一个ssh连接，一般是跳板机
func NewSSH(sshConfig SSHConfig) (ins *SSH, err error) {
	mt.Lock()
	defer mt.Unlock()

	gb, err := encoding.Gob.Marshal(sshConfig)
	if err != nil {
		return
	}
	key := common.Md5(string(gb))
	if ins, ok := sshIns[key]; ok {
		sshIns[key].ref += 1
		return ins, err
	}

	ins = &SSH{
		ref:   1,
		close: make(chan bool),
		m:     NormalMode,
	}
	ins.SetConfig(sshConfig)

	err = ins.Dial()
	if err == nil {
		sshIns[key] = ins
	}

	return
}

func (this *SSH) Dial() (err error) {

	if this.config.Addr == "" {
		return errors.New("err sshConfig")
	}

	this.c, err = ssh.Dial("tcp", this.config.Addr, this.getSshClientConfig())

	if err == nil {
		this.keepalive()
	}

	return
}

//DialRemote 通过跳板连接其他服务器
func (this *SSH) DialRemote(sshConfig SSHConfig) (ins *SSH, err error) {

	if sshConfig.Addr == "" {
		return nil, errors.New("err sshConfig")
	}

	ins = &SSH{
		ref:   1,
		close: make(chan bool),
		m:     RemoteMode,
	}
	ins.SetConfig(sshConfig)

	rc, err := this.Connect(sshConfig.Addr)
	if err != nil {
		return
	}

	conn, nc, req, err := ssh.NewClientConn(rc, "", ins.getSshClientConfig())
	if err != nil {
		return
	}

	ins.c = ssh.NewClient(conn, nc, req)

	if err == nil {
		ins.keepalive()
	}

	return
}

func (this *SSH) Connect(addr string) (conn net.Conn, err error) {
	return this.c.Dial("tcp", addr)
}

func (this *SSH) Close() {
	mt.Lock()
	defer mt.Unlock()

	logger.Warn(this.config, this.ref)
	if this.ref > 1 {
		this.ref -= 1
		return
	}

	close(this.close)
	_ = this.c.Close()
}

func (this *SSH) Config() SSHConfig {
	return this.config
}

func (this *SSH) SetConfig(sshConfig SSHConfig) {
	if sshConfig.Timeout == 0 {
		sshConfig.Timeout = 10
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

	if _, err = os.Stat(auth); err == nil {
		key, _ = ioutil.ReadFile(auth)
	}

	//密码
	if len(key) == 0 {
		if len(auth) < 50 {
			return ssh.Password(auth)
		} else {
			key = []byte(auth)
		}
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
	if this.c == nil {
		return
	}

	go func() {
		t := time.NewTicker(this.config.Timeout * time.Second)
		for {
			select {
			case <-this.close:
				t.Stop()
				return
			case <-t.C:
				go func() {
					_ = this.Keepalive()
				}()
			}
		}
	}()
}

func (this *SSH) Keepalive() (err error) {
	if this.c == nil {
		return errors.New("keepalive no ins")
	}

	sess, err := this.c.NewSession()
	if err != nil {
		return
	}
	defer func() {
		_ = sess.Close()
	}()
	if err = sess.Shell(); err != nil {
		return
	}

	return sess.Wait()
}
