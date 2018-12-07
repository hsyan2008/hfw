package db

import (
	"time"
)

type Cond map[string]interface{}

type Models struct {
	Id        int       `json:"id" xorm:"not null pk autoincr INT(10)"`
	IsDeleted int       `json:"is_deleted" xorm:"not null default 0 TINYINT(1)"`
	UpdatedAt time.Time `json:"updated_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP updated"`
	CreatedAt time.Time `json:"created_at" xorm:"not null default 'CURRENT_TIMESTAMP' TIMESTAMP created"`
}
