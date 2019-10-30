package pac

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/curl"
)

var pacUrl = "https://pac.itzmx.com/abc.pac"
var pacFile = filepath.Join(common.GetAppPath(), "abc.pac")

func LoadFromPac() (err error) {
	fileInfo, err := os.Stat(pacFile)
	if err != nil {
		err = updatePacFile()
		if err != nil {
			return err
		}
	} else if time.Now().Unix()-fileInfo.ModTime().Unix() > 86400 {
		//超过一天就更新一下
		go func() {
			_ = updatePacFile()
		}()
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
		if strings.Contains(line, "\": 1") {
			fileds := strings.Split(line, "\"")
			if len(fileds) == 3 {
				Add(fileds[1], true)
			}
		}
	}

	return err
}

func updatePacFile() (err error) {
	c := curl.NewGet(context.Background(), pacUrl)
	res, err := c.Request()
	if err != nil {
		return
	}
	err = ioutil.WriteFile(pacFile, res.Body, 0600)

	return
}
