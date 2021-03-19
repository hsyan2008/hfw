package redis

import (
	"testing"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/configs"
)

func InitCluster() {
	var err error
	DefaultIns, err = New(configs.RedisConfig{
		IsCluster: true,
		Addresses: []string{
			"192.168.0.197:7000",
			"192.168.0.197:7001",
		},
		Prefix: "stock_",
	})
	if err != nil {
		logger.Warn(err)
	}
}

func InitSimlpe() {
	var err error
	DefaultIns, err = New(configs.RedisConfig{
		IsCluster: false,
		Addresses: []string{
			"localhost:6379",
		},
		Prefix: "stock_",
	})
	if err != nil {
		logger.Warn(err)
	}
}

func TestIsExist(t *testing.T) {
	InitCluster()
	t.Log(IsExist("aaa"))
	t.Log(IsExist("aaabbb"))
}

func TestSet(t *testing.T) {
	InitCluster()
	t.Log(Set("setaa", 1))
	t.Log(Set("setaa", 1, "NX"))
}

func TestMSetCluster(t *testing.T) {
	InitCluster()
	t.Log(MSet("msetaa", 1, "msetbb", 2))
}

func TestMSetSimple(t *testing.T) {
	InitSimlpe()
	t.Log(MSet("msetaa", 1, "msetbb", 2))
	t.Log(MSet("msetmapa", map[string]string{"a": "a", "b": "a"}, "msetmapb", map[string]string{"a": "a", "b": "a"}))
}
