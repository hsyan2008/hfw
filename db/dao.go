package db

const DefaultPageSize = 1000

type Dao interface {
	UpdateById(interface{}) error
	UpdateByIds(interface{}, map[string]interface{}, []interface{}) error
	UpdateByWhere(interface{}, map[string]interface{}, map[string]interface{}) error
	Insert(interface{}) error
	SearchOne(interface{}) error
	Search(interface{}, map[string]interface{}) error
	GetMulti(interface{}, ...interface{}) error
	Count(interface{}, map[string]interface{}) (int64, error)

	EnableCache(interface{})
	DisableCache(interface{})
}
