package db

import "errors"

const DefaultPageSize = 1000

var DefaultDao Dao

var ErrDaoNotInited = errors.New("dao not inited")

type Dao interface {
	UpdateById(Model, Model) (int64, error)
	UpdateByIds(Model, Cond, []interface{}) (int64, error)
	UpdateByWhere(Model, Cond, Cond) (int64, error)
	Insert(Model, Model) (int64, error)
	InsertMulti(Model, interface{}) (int64, error)
	SearchOne(Model, Cond) (bool, error)
	Search(Model, interface{}, Cond) error
	SearchAndCount(Model, interface{}, Cond) (int64, error)
	GetMulti(Model, interface{}, ...interface{}) error
	Count(Model, Cond) (int64, error)

	DeleteByIds(Model, interface{}) (int64, error)
	DeleteByWhere(Model, Cond) (int64, error)

	EnableCache(Model)
	DisableCache(Model)
	ClearCache(Model)
}
