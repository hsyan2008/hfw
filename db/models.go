package db

import (
	"database/sql"
	"time"
)

type Cond map[string]interface{}

type Models struct {
	Id        int       `json:"id" xorm:"not null pk autoincr INT(10)"`
	IsDeleted int       `json:"is_deleted" xorm:"not null default 0 TINYINT(1)"`
	UpdatedAt time.Time `json:"updated_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP updated"`
	CreatedAt time.Time `json:"created_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP created"`

	Dao *NoCacheDao `json:"-" xorm:"-"`
}

func (m *Models) Exec(sqlState string, args ...interface{}) (sql.Result, error) {
	defer m.Dao.ClearCache(m)
	return m.Dao.Exec(sqlState, args...)
}

func (m *Models) Query(args ...interface{}) ([]map[string][]byte, error) {
	return m.Dao.Query(args...)
}

func (m *Models) QueryString(args ...interface{}) ([]map[string]string, error) {
	return m.Dao.QueryString(args...)
}
func (m *Models) QueryInterface(args ...interface{}) ([]map[string]interface{}, error) {
	return m.Dao.QueryInterface(args...)
}
