package pac

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hsyan2008/hfw/common"
)

var pacFile = filepath.Join(common.GetAppPath(), "abc.pac")

func LoadFromPac() (err error) {
	fileInfo, err := os.Stat(pacFile)
	if err != nil || time.Now().Unix()-fileInfo.ModTime().Unix() > 86400 {
		//超过一天就更新一下
		return updatePacFile()
	}

	body, err := ioutil.ReadFile(pacFile)
	if err != nil {
		return err
	}

	return parsePac(string(body))
}

func parsePac(body string) (err error) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) == 2 {
			if fields[1] == "1" {
				Add(fields[0], true)
			} else {
				Add(fields[0], false)
			}
		}
	}

	return err
}

func updatePacFile() (err error) {
	err = LoadGwflist()
	if err != nil {
		return
	}

	f, err := os.Create(pacFile)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for host, ok := range pac {
		if ok {
			w.WriteString(fmt.Sprintf("%s:1\n", host))
		} else {
			w.WriteString(fmt.Sprintf("%s:0\n", host))
		}
	}

	return w.Flush()
}
