package db

import "time"

type Cond map[string]interface{}

type Model interface {
	AutoIncrColName() string
	AutoIncrColValue() int64
	TableName() string
}

type Models struct {
	Id        int       `json:"id" xorm:"not null pk autoincr INT(10)"`
	IsDeleted int       `json:"is_deleted" xorm:"not null default 0 TINYINT(1)"`
	UpdatedAt time.Time `json:"updated_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP updated"`
	CreatedAt time.Time `json:"created_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP created"`
}

func (m *Models) GetId() (val int) {
	if m == nil {
		return
	}
	return m.Id
}

func (m *Models) GetIsDeleted() (val int) {
	if m == nil {
		return
	}
	return m.IsDeleted
}

func (m *Models) GetUpdatedAt() (val time.Time) {
	if m == nil {
		return
	}
	return m.UpdatedAt
}

func (m *Models) GetCreatedAt() (val time.Time) {
	if m == nil {
		return
	}
	return m.CreatedAt
}
