package redis

type RedisInterface interface {
	IsExist(string) (bool, error)
	Set(string, interface{}) error
	MSet(...interface{}) error
	Get(string) (interface{}, error)
	MGet(...string) (map[string]interface{}, error)
	Incr(string) (int64, error)
	Decr(string) (int64, error)
	IncrBy(string, int64) (int64, error)
	DecrBy(string, int64) (int64, error)
	Del(...string) (bool, error)
	SetNx(string, interface{}) (bool, error)
	SetEx(string, interface{}, int) error
	SetNxEx(string, interface{}, int) (bool, error)
	HExists(string, string) (bool, error)
	HSet(string, string, interface{}) error
	HGet(string, string) (interface{}, error)
	HIncrBy(string, string, int64) (int64, error)
	HDel(string, ...string) error
	Rename(string, string) error
	RenameNx(string, string) (bool, error)
	Expire(string, int32) (bool, error)
}
