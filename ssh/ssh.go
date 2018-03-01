package ssh

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/hsyan2008/go-logger/logger"
	"golang.org/x/crypto/ssh"
)

//SSHConfig ..
type SSHConfig struct {
	Addr    string `toml:"addr"`
	User    string `toml:"user"`
	Auth    string `toml:"auth"`
	Phrase  string `toml:"phrase"`
	Timeout time.Duration
}

//SSH ..
type SSH struct {
	c *ssh.Client
}

//NewSSH 建立第一个ssh连接，一般是跳板机
func NewSSH(sshConfig SSHConfig) (ssh *SSH, err error) {

	ssh = &SSH{}

	err = ssh.Dial(sshConfig)

	return ssh, err
}

func (this *SSH) Dial(sshConfig SSHConfig) (err error) {

	if sshConfig.Addr == "" {
		return errors.New("err sshConfig")
	}

	this.c, err = ssh.Dial("tcp", sshConfig.Addr, this.getSshClientConfig(sshConfig))

	return
}

//DialRemote 通过跳板连接其他服务器
func (this *SSH) DialRemote(sshConfig SSHConfig) (err error) {

	if sshConfig.Addr == "" {
		return errors.New("err sshConfig")
	}

	rc, err := this.c.Dial("tcp", sshConfig.Addr)
	if err != nil {
		return err
	}

	conn, nc, req, err := ssh.NewClientConn(rc, "", this.getSshClientConfig(sshConfig))
	if err != nil {
		return err
	}

	this.c = ssh.NewClient(conn, nc, req)

	return
}

func (this *SSH) getSshClientConfig(sshConfig SSHConfig) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: sshConfig.User,
		Auth: []ssh.AuthMethod{
			this.getAuth(sshConfig),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshConfig.Timeout * time.Second,
	}
}

func (this *SSH) getAuth(sshConfig SSHConfig) ssh.AuthMethod {
	//是文件
	var key []byte
	var err error

	if _, err = os.Stat(sshConfig.Auth); err == nil {
		key, _ = ioutil.ReadFile(sshConfig.Auth)
	}

	//密码
	if len(key) == 0 {
		if len(sshConfig.Auth) < 50 {
			return ssh.Password(sshConfig.Auth)
		} else {
			key = []byte(sshConfig.Auth)
		}
	}

	var signer ssh.Signer
	if sshConfig.Phrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(sshConfig.Phrase))
	} else {
		signer, err = ssh.ParsePrivateKey(key)
	}
	if err != nil {
		panic("err private key:" + err.Error())
	}
	return ssh.PublicKeys(signer)
}

//一个Session只能执行一次
func (this *SSH) Exec(cmd string) ([]byte, error) {

	sess, err := this.c.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = sess.Close()
	}()

	return sess.CombinedOutput(cmd)
}

func (this *SSH) Close() {
	_ = this.c.Close()
}

func keepalive(s *ssh.Client) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warn("keepalive error")
			err = errors.New("keepalive error")
		}
	}()
	if s == nil {
		return errors.New("ssh Client is nil")
	}

	sess, err := s.NewSession()
	if err != nil {
		logger.Warn("keepalive NewSession error")
		return err
	}
	defer func() {
		_ = sess.Close()
	}()
	if err = sess.Shell(); err != nil {
		logger.Warn("keepalive shell error")
		return err
	}
	err = sess.Wait()
	if err != nil {
		logger.Warn("keepalive wait", err)
	}

	return
}
