package pac

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hsyan2008/hfw2/curl"
)

var pac = make(map[string]bool)
var isLoaded bool
var mt = new(sync.Mutex)
var pacUrl = "https://pac.itzmx.com/abc.pac"
var pacFile = "abc.pac"

func parseDefault(body string) (err error) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if strings.Contains(line, "\": 1") {
			fileds := strings.Split(line, "\"")
			if len(fileds) == 3 {
				add(fileds[1], true)
			}
		}
	}

	return err
}

func updatePacFile() (err error) {
	c := curl.NewCurl(pacUrl)
	res, err := c.Request()
	if err != nil {
		return
	}
	err = ioutil.WriteFile(pacFile, res.Body, 0600)

	return
}

func LoadDefault() (err error) {
	mt.Lock()
	defer mt.Unlock()
	if isLoaded {
		return
	}

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

	isLoaded = true

	return parseDefault(string(body))
}

func Add(key string, val bool) {
	mt.Lock()
	defer mt.Unlock()
	//一旦调用，就认为已经加载过
	isLoaded = true
	add(key, val)
}
func add(key string, val bool) {
	pac[key] = val
}

func Check(addr string) bool {
	if len(pac) == 0 {
		return false
	}
	host := strings.Split(addr, ":")[0]
	hosts := strings.Split(host, ".")
	pos := 1
	for pos <= len(hosts) {
		tmp := hosts[len(hosts)-pos:]
		tmp1 := strings.Join(tmp, ".")
		if isAllow, ok := pac[tmp1]; ok {
			return isAllow
		} else {
			pos++
		}
	}

	return false
}
