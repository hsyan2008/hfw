package ssh

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

//实现了文件和目录的上传，未限速
//
//可以实现过滤，不支持正则过滤
//
//文件上传，src和des都可以包含文件名字
//如果des是个已存在的目录，则上传到des下
//如果des不存在，则文件会重命名为des
//
//目录上传，如果src和des的base一样，则src将上传到des的父目录下面，否则上传到des下
func (this *SSH) Scp(src, des, exclude string) (err error) {

	exclude = strings.Replace(exclude, "\r", "", -1)
	tmp := strings.Split(exclude, "\n")
	excludes := make(map[string]string)
	for _, v := range tmp {
		v = strings.TrimSpace(v)
		if v != "" {
			excludes[v] = v
		}
	}

	fileinfo, err := os.Stat(src)
	if err != nil {
		return
	}

	return this.scpDir(src, des, excludes, fileinfo)
}

func (this *SSH) scpDir(src, des string, excludes map[string]string, fileinfo os.FileInfo) (err error) {
	if _, ok := excludes[fileinfo.Name()]; ok {
		return
	}

	if fileinfo.Mode().IsDir() {
		if path.Base(src) != path.Base(des) {
			des = filepath.Join(des, path.Base(src))
		}

		_, _ = this.Exec(fmt.Sprintf("mkdir -m %#o -p %s", fileinfo.Mode().Perm(), pathConvertToUnix(des)))

		file, err := os.Open(src)
		if err != nil {
			return err
		}
		for {
			files, err := file.Readdir(16)
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			for _, v := range files {
				err = this.scpDir(filepath.Join(src, v.Name()), filepath.Join(des, v.Name()), excludes, v)
				if err != nil {
					return err
				}
			}
		}
	} else if fileinfo.Mode().IsRegular() {
		return this.scpFile(src, des, fileinfo)
	}

	return nil
}
func (this *SSH) scpFile(src, des string, fileinfo os.FileInfo) (err error) {
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
		fmt.Fprintf(w, "C%#o %d %s\n", fileinfo.Mode().Perm(), fileinfo.Size(), fileinfo.Name())
		File, err := os.Open(src)
		if err != nil {
			return
		}
		_, err = io.Copy(w, File)
		if err != nil {
			return
		}

		fmt.Fprint(w, "\x00")
	}()

	des = pathConvertToUnix(des)
	//filepath在win下会转换/为\，所以用path
	cmd := fmt.Sprintf("mkdir -m %#o -p %s; /usr/bin/scp -qrt %s", fileinfo.Mode().Perm(), path.Dir(des), des)
	if err := sess.Run(cmd); err != nil {
		if err.Error() != "Process exited with status 1" {
			return err
		}
	}

	return nil
}

func pathConvertToUnix(from string) string {
	if runtime.GOOS == "windows" {
		from = strings.Replace(from, "\\", "/", -1)
	}

	return from
}
