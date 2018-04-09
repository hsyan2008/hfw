package pac

import (
	"io/ioutil"
	"strings"
	"sync"

	"github.com/hsyan2008/hfw2/curl"
)

var pac = make(map[string]bool)
var isLoaded bool
var mt = new(sync.Mutex)

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

func LoadDefault() (err error) {
	mt.Lock()
	defer mt.Unlock()
	if isLoaded {
		return
	}
	c := curl.NewCurl("https://pac.itzmx.com/abc.pac")
	res, err := c.Request()
	if err != nil {
		res.Body, err = ioutil.ReadFile("abc.pac")
	}

	if err != nil {
		return err
	}

	isLoaded = true

	return parseDefault(string(res.Body))
}

func Add(key string, val bool) {
	mt.Lock()
	defer mt.Unlock()
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
