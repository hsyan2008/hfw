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
	for pos := 0; pos < len(hosts); pos++ {
		key := strings.Join(hosts[pos:], ".")
		if isAllow, ok := pac[key]; ok {
			return isAllow
		}
	}

	return false
}
