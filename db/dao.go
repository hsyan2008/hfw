package db

const DefaultPageSize = 1000

type Dao interface {
	UpdateById(Model) (int64, error)
	UpdateByIds(Model, Cond, []interface{}) (int64, error)
	UpdateByWhere(Model, Cond, Cond) (int64, error)
	Insert(Model) (int64, error)
	InsertMulti(interface{}) (int64, error)
	SearchOne(Model, Cond) (bool, error)
	Search(Model, interface{}, Cond) error
	SearchAndCount(Model, interface{}, Cond) (int64, error)
	GetMulti(Model, interface{}, ...interface{}) error
	Count(Model, Cond) (int64, error)

	EnableCache(Model)
	DisableCache(Model)
	ClearCache(Model)
}
