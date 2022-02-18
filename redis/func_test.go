package redis

import (
	"os"
	"testing"

	"github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/configs"
	"github.com/stretchr/testify/assert"
)

var isCluster = true

func InitCluster() {
	var err error
	DefaultIns, err = New(configs.RedisConfig{
		IsCluster: true,
		Addresses: []string{
			"192.168.0.197:7000",
			"192.168.0.197:7001",
		},
		Prefix: "redis_test_",
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
		Prefix: "redis_test_",
	})
	if err != nil {
		logger.Warn(err)
	}
}

func setup() {
	if isCluster {
		InitCluster()
	} else {
		InitSimlpe()
	}
}
func teardown() {
	DefaultIns.Close()
}

func TestMain(m *testing.M) {
	//集群-json编码
	isCluster = true
	setup()
	code1 := m.Run()
	teardown()

	//集群-原生
	isCluster = true
	setup()
	DefaultIns.Marshal = nil
	DefaultIns.Unmarshal = nil
	code2 := m.Run()
	teardown()

	//非集群-json编码
	isCluster = false
	setup()
	code3 := m.Run()
	teardown()

	//非集群-原生
	isCluster = false
	setup()
	DefaultIns.Marshal = nil
	DefaultIns.Unmarshal = nil
	code4 := m.Run()
	teardown()

	os.Exit(code1 + code2 + code3 + code4)
}

type s struct {
	A string
	B string
}

func TestSet1(t *testing.T) {
	assert := assert.New(t)
	Del("setaa", "setab", "strutct")

	ok, err := Set("setaa", 1)
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = Set("setaa", 1, "NX")
	if assert.Nil(err) {
		assert.False(ok)
	}

	ok, err = Set("setab", 1, "NX")
	if assert.Nil(err) {
		assert.True(ok)
	}

	if DefaultIns.Marshal != nil {
		ok, err = Set("strutct", s{A: "abc", B: "bbb"})
		if assert.Nil(err) {
			assert.True(ok)
		}
	}
}

func TestGet(t *testing.T) {
	assert := assert.New(t)

	var i int
	b, err := Get(&i, "setaa")
	if assert.Nil(err) {
		assert.True(b)
		assert.Equal(1, i, "setaa not get")
	}

	b, err = Get(&i, "setab")
	if assert.Nil(err) {
		assert.True(b)
		assert.Equal(1, i, "setab not get")
	}

	b, err = Get(&i, "notkey")
	if assert.Nil(err) {
		assert.False(b)
		assert.Equal(1, i, "notkey not get")
	}

	if DefaultIns.Unmarshal != nil {
		var s s
		b, err = Get(&s, "strutct")
		if assert.Nil(err) {
			assert.True(b)
			assert.Equal("abc", s.A)
			assert.Equal("bbb", s.B)
		}
	}
}

func TestMSet(t *testing.T) {
	assert := assert.New(t)

	err := MSet("msetaa", 1, "msetbb", 2)
	assert.Nil(err)

	if DefaultIns.Marshal != nil {
		err = MSet("msetmapa", map[string]string{"a": "a", "b": "a"}, "msetmapb", map[string]string{"a": "a", "b": "a"})
		assert.Nil(err)

		err = MSet("msetstrutct", struct {
			A string
			B string
		}{A: "abc", B: "bbb"})
		assert.Nil(err)
	}
}

func TestMGet(t *testing.T) {
	assert := assert.New(t)
	TestMSet(t)

	if DefaultIns.Unmarshal != nil {
		i, err := MGet("msetaa", "msetcc", "msetbb")
		if assert.Nil(err) {
			var j int
			err = Unmarshal(i[0], &j)
			assert.Nil(err)
			assert.Equal(1, j, "msetaa is not 1")

			err = Unmarshal(i[2], &j)
			assert.Nil(err)
			assert.Equal(2, j, "msetbb is not 2")

			assert.Equal(0, len(i[1]), "msetcc is not empty")
		}

		i, err = MGet("msetmapa", "msetmapb")
		if assert.Nil(err) {
			var k map[string]string
			for _, v := range i {
				err = Unmarshal(v, &k)
				assert.Nil(err)
				assert.Equal("a", k["a"], "key a is not a")
				assert.Equal("a", k["b"], "key b is not a")
			}
		}

		i, err = MGet("msetstrutct")
		if assert.Nil(err) {
			var l s
			for _, v := range i {
				err = Unmarshal(v, &l)
				assert.Nil(err)
				assert.Equal("abc", l.A, "field A is not abc")
				assert.Equal("bbb", l.B, "field B is not abc")
			}
		}
	}
}

func TestIsExist(t *testing.T) {
	assert := assert.New(t)

	ok, err := IsExist("msetaa")
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = IsExist("msetcc")
	if assert.Nil(err) {
		assert.False(ok)
	}
}

func TestIncr1(t *testing.T) {
	assert := assert.New(t)
	Del("testincr")

	i, err := Incr("testincr")
	if assert.Nil(err) {
		assert.EqualValues(1, i, "testincr is not 1")
	}

	i, err = Incr("testincr")
	if assert.Nil(err) {
		assert.EqualValues(2, i, "testincr is not 2")
	}
}

func TestDecr1(t *testing.T) {
	assert := assert.New(t)
	Del("testincr")

	i, err := Decr("testincr")
	if assert.Nil(err) {
		assert.EqualValues(-1, i, "testincr is not -1")
	}

	i, err = Decr("testincr")
	if assert.Nil(err) {
		assert.EqualValues(-2, i, "testincr is not -2")
	}
}

func TestIncrBy(t *testing.T) {
	assert := assert.New(t)
	Del("testincr")

	i, err := IncrBy("testincr", 2)
	if assert.Nil(err) {
		assert.EqualValues(2, i, "testincr is not 2")
	}

	i, err = IncrBy("testincr", 2)
	if assert.Nil(err) {
		assert.EqualValues(4, i, "testincr is not 4")
	}
}

func TestDecrBy(t *testing.T) {
	assert := assert.New(t)
	Del("testincr")

	i, err := DecrBy("testincr", 2)
	if assert.Nil(err) {
		assert.EqualValues(-2, i, "testincr is not -2")
	}

	i, err = DecrBy("testincr", 2)
	if assert.Nil(err) {
		assert.EqualValues(-4, i, "testincr is not -4")
	}
}

func TestDel(t *testing.T) {
	assert := assert.New(t)
	MSet("testdela", 1, "testdelb", 2)

	i, err := Del("testdela", "testdelb", "testdelc")
	if assert.Nil(err) {
		assert.EqualValues(2, i, "del num is not 2")
	}
}

func TestSetNx1(t *testing.T) {
	assert := assert.New(t)
	Del("SetNx")

	ok, err := SetNx("SetNx", 1)
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = SetNx("SetNx", 1)
	if assert.Nil(err) {
		assert.False(ok)
	}
}

func TestSetEx(t *testing.T) {
	assert := assert.New(t)

	err := SetEx("SetEx", 1, 5)
	assert.Nil(err)
	i, err := Ttl("SetEx")
	if assert.Nil(err) {
		assert.EqualValues(5, i, "SetEx expiration is not 5")
	}
}

func TestSetNxEx(t *testing.T) {
	assert := assert.New(t)
	Del("SetNxEx")

	ok, err := SetNxEx("SetNxEx", 1, 5)
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = SetNxEx("SetNxEx", 1, 5)
	if assert.Nil(err) {
		assert.False(ok)
	}
}

func TestHSet(t *testing.T) {
	assert := assert.New(t)

	err := HSet("hmap", "name", "tt")
	assert.Nil(err)

	err = HSet("hmap", "age", 25)
	assert.Nil(err)
}

func TestHGet(t *testing.T) {
	assert := assert.New(t)
	TestHSet(t)

	var name string
	b, err := HGet(&name, "hmap", "name")
	if assert.Nil(err) {
		assert.True(b)
		assert.Equal("tt", name, "hmap name is not tt")
	}

	var not string
	b, err = HGet(&not, "hmap", "not")
	if assert.Nil(err) {
		assert.False(b)
		assert.Equal("", not, "hmap not is not empty")
	}
}

func TestHExists(t *testing.T) {
	assert := assert.New(t)
	TestHSet(t)

	ok, err := HExists("hmap", "name")
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = HExists("hmap", "not")
	if assert.Nil(err) {
		assert.False(ok)
	}
}

func TestHIncrBy(t *testing.T) {
	assert := assert.New(t)
	// HDel("hmap", "click")
	Del("hmap")

	i, err := HIncrBy("hmap", "click", 1)
	if assert.Nil(err) {
		assert.EqualValues(1, i)
	}

	i, err = HIncrBy("hmap", "click", 1)
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}
}

func TestHDel(t *testing.T) {
	assert := assert.New(t)
	TestHSet(t)

	i, err := HDel("hmap", "name", "age", "not")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}
}

func TestZAdd(t *testing.T) {
	assert := assert.New(t)
	Del("zset")

	i, err := ZAdd("zset", 1, "one")
	if assert.Nil(err) {
		assert.EqualValues(1, i)
	}

	i, err = ZAdd("zset", 2, "two", 3, "three")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}

	i, err = ZAdd("zset", "NX", 2, "two", 4, "four")
	if assert.Nil(err) {
		assert.EqualValues(1, i)
	}

	i, err = ZAdd("zset", "INCR", 2, "two")
	if assert.Nil(err) {
		assert.EqualValues(4, i)
	}
}

func TestZRem(t *testing.T) {
	assert := assert.New(t)
	TestZAdd(t)

	i, err := ZRem("zset", "one", "five")
	if assert.Nil(err) {
		assert.EqualValues(1, i)
	}
}

func TestZIncrBy(t *testing.T) {
	assert := assert.New(t)
	TestZAdd(t)

	i, err := ZIncrBy("zset", "one", 2)
	if assert.Nil(err) {
		assert.EqualValues(3, i)
	}
}

func TestZRange(t *testing.T) {
	assert := assert.New(t)
	TestZAdd(t)

	i, err := ZRange("zset", 1, 5)
	if assert.Nil(err) {
		assert.Greater(len(i), 0)
	}
}

func TestRename1(t *testing.T) {
	assert := assert.New(t)

	err := Rename("Rename_no_old", "Rename_no_new")
	assert.NotNil(err)

	Set("Rename_old", 1)
	err = Rename("Rename_old", "Rename_new")
	assert.Nil(err)

	var i int
	b, err := Get(&i, "Rename_new")
	if assert.Nil(err) {
		assert.True(b)
		assert.EqualValues(1, i)
	}
}

func TestRenameNx(t *testing.T) {
	assert := assert.New(t)

	_, err := RenameNx("RenameNx_no_old", "RenameNx_no_new")
	assert.NotNil(err)

	Del("RenameNx_new")
	Set("RenameNx_old", 1)
	ok, err := RenameNx("RenameNx_old", "RenameNx_new")
	if assert.Nil(err) {
		assert.True(ok)
	}
	var i int
	b, err := Get(&i, "RenameNx_new")
	if assert.Nil(err) {
		assert.True(b)
		assert.EqualValues(1, i)
	}

	Set("RenameNx_old2", 1)
	ok, err = RenameNx("RenameNx_old2", "RenameNx_new")
	if assert.Nil(err) {
		assert.False(ok)
	}
}

func TestExpire(t *testing.T) {
	assert := assert.New(t)

	Set("Expire", 1)
	ok, err := Expire("Expire", 10)
	if assert.Nil(err) {
		assert.True(ok)
	}

	i, err := Ttl("Expire")
	if assert.Nil(err) {
		assert.EqualValues(10, i)
	}
}

func TestTtl(t *testing.T) {
	assert := assert.New(t)

	Set("Expire", 1)
	Expire("Expire", 10)

	i, err := Ttl("Expire")
	if assert.Nil(err) {
		assert.EqualValues(10, i)
	}
}

func TestGeoAdd(t *testing.T) {
	assert := assert.New(t)
	Del("Sicily")

	i, err := GeoAdd("Sicily", 13.361389, 38.115556, "Palermo", 15.087269, 37.502669, "Catania")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}
}

func TestGeoDist(t *testing.T) {
	assert := assert.New(t)
	TestGeoAdd(t)

	i, err := GeoDist("Sicily", "Palermo", "Catania")
	if assert.Nil(err) {
		assert.Greater(i, 0.0)
	}
}

func TestGeoHash(t *testing.T) {
	assert := assert.New(t)
	TestGeoAdd(t)

	i, err := GeoHash("Sicily", "Palermo", "Catania", "NonExisting")
	if assert.Nil(err) {
		assert.EqualValues(3, len(i))
	}
}

func TestGeoPos(t *testing.T) {
	assert := assert.New(t)
	TestGeoAdd(t)

	i, err := GeoPos("Sicily", "Palermo", "Catania", "NonExisting")
	if assert.Nil(err) {
		assert.Greater(len(i), 0)
	}
}

func TestGeoRadius(t *testing.T) {
	assert := assert.New(t)
	TestGeoAdd(t)

	i, err := GeoRadius("Sicily", 15, 37, 200, "km", "WITHDIST", "WITHCOORD")
	if assert.Nil(err) {
		assert.Greater(len(i), 0)
	}
}

func TestGeoRadiusByMember(t *testing.T) {
	assert := assert.New(t)
	TestGeoAdd(t)

	GeoAdd("Sicily", 13.583333, 37.316667, "Agrigento")
	i, err := GeoRadiusByMember("Sicily", "Agrigento", 100, "km", "WITHDIST", "WITHCOORD")
	if assert.Nil(err) {
		assert.Greater(len(i), 0)
	}
}

func TestLPush(t *testing.T) {
	assert := assert.New(t)

	num, err := LPush("lists", 1, 3, 4)
	if assert.Nil(err) {
		assert.GreaterOrEqual(num, int64(3))
	}
}

func TestLPop(t *testing.T) {
	assert := assert.New(t)
	Del("lists")
	var i int
	b, err := LPop(&i, "lists")
	if assert.Nil(err) {
		assert.False(b)
		assert.EqualValues(0, i)
	}

	TestLPush(t)
	b, err = LPop(&i, "lists")
	if assert.Nil(err) {
		assert.True(b)
		assert.Greater(i, 0)
	}
}

func TestRPush(t *testing.T) {
	assert := assert.New(t)

	num, err := RPush("lists", 1, 3, 4)
	if assert.Nil(err) {
		assert.GreaterOrEqual(num, int64(3))
	}
}

func TestRPop(t *testing.T) {
	assert := assert.New(t)
	Del("lists")
	var i int
	b, err := RPop(&i, "lists")
	if assert.Nil(err) {
		assert.False(b)
		assert.EqualValues(0, i)
	}

	TestLPush(t)
	b, err = RPop(&i, "lists")
	if assert.Nil(err) {
		assert.True(b)
		assert.Greater(i, 0)
	}
}

func TestLLen(t *testing.T) {
	assert := assert.New(t)
	Del("lists")
	b, err := LLen("lists")
	if assert.Nil(err) {
		assert.EqualValues(0, b)
	}

	TestLPush(t)
	b, err = LLen("lists")
	if assert.Nil(err) {
		assert.Greater(b, int64(0))
	}
}

func TestSAdd(t *testing.T) {
	assert := assert.New(t)
	Del("sadd")

	i, err := SAdd("sadd", 1, "one")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}

	i, err = SAdd("sadd", 2, "two", 3, "three")
	if assert.Nil(err) {
		assert.EqualValues(4, i)
	}
}

func TestSDiffStore(t *testing.T) {
	assert := assert.New(t)

	sAddKey1 := "sadd1"
	sAddKey2 := "sadd2"
	sDiffStoreKey1 := "sdiff_store"

	Del(sAddKey1)
	Del(sAddKey2)
	Del(sDiffStoreKey1)

	i, err := SAdd(sAddKey1, 1, "one")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}

	i, err = SAdd(sAddKey2, 2, "one", 3, "three")
	if assert.Nil(err) {
		assert.EqualValues(4, i)
	}

	i, err = SDiffStore(sDiffStoreKey1, GetPrefix()+sAddKey2, GetPrefix()+sAddKey1)
	if assert.Nil(err) {
		assert.EqualValues(3, i)
	}
}

func TestSCard(t *testing.T) {
	assert := assert.New(t)
	Del("sadd")

	i, err := SAdd("sadd", 1, "one")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}

	i, err = SCard("sadd")
	if assert.Nil(err) {
		assert.EqualValues(2, i)
	}

	i, err = SAdd("sadd", 2, "one", 3, "three")
	if assert.Nil(err) {
		assert.EqualValues(3, i)
	}

	i, err = SCard("sadd")
	if assert.Nil(err) {
		assert.EqualValues(5, i)
	}
}

func TestSIsMember(t *testing.T) {
	assert := assert.New(t)

	TestSAdd(t)

	ok, err := SIsMember("sadd", "one")
	if assert.Nil(err) {
		assert.True(ok)
	}

	ok, err = SIsMember("sadd", "five")
	if assert.Nil(err) {
		assert.False(ok)
	}
}
