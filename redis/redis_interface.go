package redis

type RedisInterface interface {
	IsExist(string) (bool, error)
	Set(string, interface{}) error
	Get(string) (interface{}, error)
	Incr(string) (int64, error)
	Decr(string) (int64, error)
	IncrBy(string, uint64) (int64, error)
	DecrBy(string, uint64) (int64, error)
	Del(string) (bool, error)
	SetNx(string, interface{}) (bool, error)
	SetEx(string, interface{}, int) error
	SetNxEx(string, interface{}, int) (bool, error)
	Hexists(string, string) (bool, error)
	Hset(string, string, interface{}) error
	Hget(string, string) (interface{}, error)
	Hdel(string, string) error
	Rename(string, string) error
	RenameNx(string, string) (bool, error)
	Expire(string, int32) (bool, error)
}
