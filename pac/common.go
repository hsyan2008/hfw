package pac

import (
	"strings"
	"sync"
)

var pac = make(map[string]bool)
var mt = new(sync.Mutex)
var isLoad bool

func Reset() (err error) {
	pac = make(map[string]bool)
	isLoad = false

	return LoadDefault()
}

var loadMt = new(sync.Mutex)

func LoadDefault() (err error) {
	if isLoad {
		return
	}
	loadMt.Lock()
	defer loadMt.Unlock()
	if isLoad {
		return
	}
	isLoad = true
	return LoadFromPac()
}

func GetAll() map[string]bool {
	return pac
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
	for pos := 0; pos < len(hosts); pos++ {
		key := strings.Join(hosts[pos:], ".")
		if isAllow, ok := pac[key]; ok {
			return isAllow
		}
	}

	return false
}
