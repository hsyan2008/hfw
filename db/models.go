package db

import (
	"github.com/hsyan2008/hfw2/encoding"
)

type Cond map[string]interface{}

type Models struct {
	Id        int               `json:"id" xorm:"not null pk autoincr INT(10)"`
	IsDeleted int               `json:"is_deleted" xorm:"not null default 0 TINYINT(1)"`
	UpdatedAt encoding.DateTime `json:"updated_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP updated"`
	CreatedAt encoding.DateTime `json:"created_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP created"`
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

func (m *Models) GetUpdatedAt() (val encoding.DateTime) {
	if m == nil {
		return
	}
	return m.UpdatedAt
}

func (m *Models) GetCreatedAt() (val encoding.DateTime) {
	if m == nil {
		return
	}
	return m.CreatedAt
}
