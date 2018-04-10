package pac

import (
	"strings"
	"sync"
)

var pac = make(map[string]bool)
var isLoaded bool
var mt = new(sync.Mutex)

func LoadDefault() (err error) {
	err = LoadGwflist()
	if err == nil {
		return
	}

	return LoadFromPac()
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
