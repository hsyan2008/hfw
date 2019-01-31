package db

const DefaultPageSize = 1000

type Dao interface {
	UpdateById(interface{}) (int64, error)
	UpdateByIds(interface{}, Cond, []interface{}) (int64, error)
	UpdateByWhere(interface{}, Cond, Cond) (int64, error)
	Insert(interface{}) (int64, error)
	SearchOne(interface{}, Cond) error
	Search(interface{}, Cond) error
	GetMulti(interface{}, ...interface{}) error
	Count(interface{}, Cond) (int64, error)

	EnableCache(interface{})
	DisableCache(interface{})
}
