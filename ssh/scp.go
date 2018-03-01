package ssh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//实现了目录的上传，未限速
//可以实现过滤，不支持正则过滤，不过滤最外层
func (this *SSH) Scp(src, des, exclude string) (err error) {

	exclude = strings.Replace(exclude, "\r", "", -1)
	tmp := strings.Split(exclude, "\n")
	excludes := make(map[string]string)
	for _, v := range tmp {
		excludes[v] = v
	}

	file, err := os.Open(src)
	if err != nil {
		return
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return
	}
	if fileinfo.Mode().IsDir() {
		//如果srcDir是目录，走这个
		return this.scpDir(src, des, excludes, 0755, true)
	} else if fileinfo.Mode().IsRegular() {
		//如果srcDir是文件，则执行ssh.Run的时候，不用mkdir
		return this.scpDir(src, des, excludes, 0755, false)
	}

	return nil
}

func (this *SSH) scpDir(src, des string, excludes map[string]string, fm os.FileMode, isDir bool) (err error) {

	file, err := os.Open(src)
	if err != nil {
		return
	}
	fileinfo, err := file.Stat()
	if err != nil {
		return
	}
	if fileinfo.Mode().IsDir() {
		des := des + "/" + filepath.Base(src)
		for {
			files, err := file.Readdir(3)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			for _, v := range files {
				if _, ok := excludes[v.Name()]; ok {
					continue
				}
				err = this.scpDir(src+"/"+v.Name(), des, excludes, fileinfo.Mode().Perm(), isDir)
				if err != nil {
					return err
				}
			}
		}
	} else if fileinfo.Mode().IsRegular() {
		return this.scpFile(src, des, fm, isDir)
	}

	return nil
}
func (this *SSH) scpFile(src, des string, fm os.FileMode, isDir bool) (err error) {
	sess, err := this.c.NewSession()
	if err != nil {
		return
	}
	defer func() {
		_ = sess.Close()
	}()

	go func() {
		w, err := sess.StdinPipe()
		if err != nil {
			return
		}
		defer func() {
			_ = w.Close()
		}()
		File, err := os.Open(src)
		if err != nil {
			return
		}
		info, err := File.Stat()
		if err != nil {
			return
		}
		// fmt.Fprintln(w, "C0755", info.Size(), info.Name())
		// fmt.Fprintf(w, "C%#o %d %s\n", info.Mode().Perm(), info.Size(), info.Name())
		//发布代码，文件默认是644
		fmt.Fprintf(w, "C0644 %d %s\n", info.Size(), info.Name())
		_, err = io.Copy(w, File)
		if err != nil {
			return
		}

		fmt.Fprint(w, "\x00")
	}()

	var b bytes.Buffer
	sess.Stdout = &b
	var cmd string
	if isDir {
		// cmd = fmt.Sprintf("mkdir -m %#o -p %s; /usr/bin/scp -qrt %s", fm, des, des)
		//发布代码，目录默认是755
		cmd = fmt.Sprintf("mkdir -m 0755 -p %s; /usr/bin/scp -qrt %s", des, des)
	} else {
		cmd = fmt.Sprintf("/usr/bin/scp -qrt %s", des)
	}
	if err := sess.Run(cmd); err != nil {
		if err.Error() != "Process exited with status 1" {
			return err
		}
	}

	return nil
}
