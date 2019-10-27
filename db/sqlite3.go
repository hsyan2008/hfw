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
		dbConfig.Params = fmt.Sprintf("?_auth_user=%s&_auth_pass=%s",
			dbConfig.Username, dbConfig.Password)
	} else {
		dbConfig.Params = fmt.Sprintf("?%s&_auth_user=%s&_auth_pass=%s",
			dbConfig.Params, dbConfig.Username, dbConfig.Password)
	}
	return fmt.Sprintf("file:%s%s",
		dbConfig.Address, dbConfig.Params)
}
