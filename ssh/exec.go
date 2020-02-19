package ssh

import (
	"errors"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

//一个Session只能执行一次，获取ssh执行命令的结果
func (this *SSH) Exec(cmd string) (string, error) {
	if this.c == nil {
		return "", errors.New("nil client")
	}

	sess, err := this.c.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	c, err := sess.CombinedOutput(cmd)

	return string(c), err
}

//一个Session只能执行一次，直接把结果输出到终端
//不允许超过3s
func (this *SSH) ExecWithPty(cmd string, timeout time.Duration) error {
	if this.c == nil {
		return errors.New("nil client")
	}

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
		defer terminal.Restore(fd, oldState)

		sess, err := this.c.NewSession()
		if err != nil {
			return err
		}
		defer sess.Close()

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
		if timeout > 0 {
			go func(sess *ssh.Session) {
				select {
				case <-time.After(timeout * time.Second):
					if !done {
						_ = sess.Close()
					}
				}

			}(sess)
		}
		err = sess.Wait()
		done = true

		return err
	} else {
		return errors.New("no terminal")
	}
}

//一个Session只能执行一次，进入ssh模式
func (this *SSH) Shell() error {
	if this.c == nil {
		return errors.New("nil client")
	}

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
		defer terminal.Restore(fd, oldState)

		sess, err := this.c.NewSession()
		if err != nil {
			return err
		}
		defer sess.Close()

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
