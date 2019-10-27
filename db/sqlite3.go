// +build  sqlite3

package db

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"

	//sqlite
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	dnsFuncMap["sqlite3"] = getSqlite3Dns
}

func getSqlite3Dns(dbConfig configs.DbStdConfig) string {
	if len(dbConfig.Params) == 0 {
		dbConfig.Params = "?cache=shared&mode=memory"
	}
	return fmt.Sprintf("file:%s%s&_auth_user=%s&_auth_pass=%s",
		dbConfig.Address, dbConfig.Params, dbConfig.Username, dbConfig.Password)
}
