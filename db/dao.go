package db

import (
	"errors"

	"github.com/hsyan2008/hfw/configs"
)

const DefaultPageSize = 1000

var DefaultDao Dao

var ErrDaoNotInited = errors.New("dao not inited")

type Dao interface {
	GetConf() configs.DbConfig

	IsTableExist(interface{}) (bool, error)
	UpdateByIds(Model, interface{}, []interface{}, ...string) (int64, error)
	UpdateByWhere(Model, Cond, Cond) (int64, error)
	Insert(Model, Model) (int64, error)
	InsertMulti(Model, interface{}) (int64, error)
	SearchOne(Model, Cond) (bool, error)
	Search(Model, interface{}, Cond) error
	SearchAndCount(Model, interface{}, Cond) (int64, error)
	GetByIds(Model, interface{}, []interface{}, ...string) error
	Count(Model, Cond) (int64, error)

	DeleteByIds(Model, interface{}) (int64, error)
	DeleteByWhere(Model, Cond) (int64, error)

	EnableCache(Model)
	DisableCache(Model)
	ClearCache(Model)
}
