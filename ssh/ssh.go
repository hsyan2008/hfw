package ssh

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
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

//SSH ..
type SSH struct {
	c      *ssh.Client
	close  chan bool
	config SSHConfig
}

//NewSSH 建立第一个ssh连接，一般是跳板机
func NewSSH(sshConfig SSHConfig) (ins *SSH, err error) {

	ins = &SSH{}
	ins.close = make(chan bool, 1)
	ins.config = sshConfig

	err = ins.Dial(sshConfig)

	return
}

func (this *SSH) Dial(sshConfig SSHConfig) (err error) {

	if sshConfig.Addr == "" {
		return errors.New("err sshConfig")
	}

	this.c, err = ssh.Dial("tcp", sshConfig.Addr, this.getSshClientConfig(sshConfig))

	if err == nil {
		this.Keepalive()
	}

	return
}

//DialRemote 通过跳板连接其他服务器
func (this *SSH) DialRemote(sshConfig SSHConfig) (ins *SSH, err error) {

	if sshConfig.Addr == "" {
		return nil, errors.New("err sshConfig")
	}

	ins = &SSH{}
	ins.close = make(chan bool, 1)
	ins.config = sshConfig

	rc, err := this.c.Dial("tcp", sshConfig.Addr)
	if err != nil {
		return
	}

	conn, nc, req, err := ssh.NewClientConn(rc, "", this.getSshClientConfig(sshConfig))
	if err != nil {
		return
	}

	ins.c = ssh.NewClient(conn, nc, req)

	if err == nil {
		ins.Keepalive()
	}

	return
}

func (this *SSH) Close() {
	this.close <- true
	_ = this.c.Close()
}

func (this *SSH) Config() SSHConfig {
	return this.config
}

func (this *SSH) getSshClientConfig(sshConfig SSHConfig) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: sshConfig.User,
		Auth: []ssh.AuthMethod{
			this.getAuth(sshConfig),
		},
		//如果没有这个，会说需要know_host文件
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

//一个Session只能执行一次，获取ssh执行命令的结果
func (this *SSH) Exec(cmd string) (string, error) {

	sess, err := this.c.NewSession()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = sess.Close()
	}()

	c, err := sess.CombinedOutput(cmd)

	return string(c), err
}

//一个Session只能执行一次，直接把结果输出到终端
//不允许超过3s
func (this *SSH) ExecWithPty(cmd string) error {

	fd := 0
	if terminal.IsTerminal(fd) {
		termWidth, termHeight, err := terminal.GetSize(fd)
		if err != nil {
			return err
		}

		oldState, err := terminal.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer func() {
			_ = terminal.Restore(fd, oldState)
		}()

		sess, err := this.c.NewSession()
		if err != nil {
			return err
		}
		defer func() {
			_ = sess.Close()
		}()

		//如果没有stdin，top之类的命令无法操作
		// sess.Stdin = os.Stdin
		sess.Stdout = os.Stdout
		sess.Stderr = os.Stderr

		modes := ssh.TerminalModes{
			ssh.ECHO:          0,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := sess.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
			return err
		}

		// return sess.Run(cmd)

		_ = sess.Start(cmd)
		done := false
		go func(sess *ssh.Session) {
			select {
			case <-time.After(3 * time.Second):
				if !done {
					_ = sess.Close()
				}
			}

		}(sess)
		err = sess.Wait()
		done = true

		return err
	} else {
		return errors.New("no terminal")
	}
}

//一个Session只能执行一次，进入ssh模式
func (this *SSH) Shell() error {

	fd := 0
	if terminal.IsTerminal(fd) {
		termWidth, termHeight, err := terminal.GetSize(fd)
		if err != nil {
			return err
		}

		oldState, err := terminal.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer func() {
			_ = terminal.Restore(fd, oldState)
		}()

		sess, err := this.c.NewSession()
		if err != nil {
			return err
		}
		defer func() {
			_ = sess.Close()
		}()

		sess.Stdin = os.Stdin
		sess.Stdout = os.Stdout
		sess.Stderr = os.Stderr

		modes := ssh.TerminalModes{
			ssh.ECHO: 1,
		}

		if err := sess.RequestPty("xterm", termHeight, termWidth, modes); err != nil {
			return err
		}

		if err := sess.Shell(); err != nil {
			return err
		}

		return sess.Wait()
	} else {
		return errors.New("no terminal")
	}
}

func (this *SSH) Keepalive() {
	if this.c == nil {
		return
	}
	go func() {
		t := time.NewTimer(this.config.Timeout * time.Second)
		for {
			select {
			case <-this.close:
				t.Stop()
				return
			case <-t.C:
				go func() {
					_ = this.keepalive()
				}()
			}
		}
	}()
}

func (this *SSH) keepalive() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = errors.New("keepalive error")
		}
	}()
	if this.c == nil {
		return
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
