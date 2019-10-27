//默认是mysql
// +build  mysql mariadb
package db

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"

	//mysql
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	dnsFuncMap["mysql"] = getMysqlDns
	dnsFuncMap["mariadb"] = getMysqlDns
}

func getMysqlDns(dbConfig configs.DbStdConfig) string {
	if dbConfig.Port != "" {
		dbConfig.Address = fmt.Sprintf("%s:%s", dbConfig.Address, dbConfig.Port)
	}
	return fmt.Sprintf("%s:%s@%s(%s)/%s%s",
		dbConfig.Username, dbConfig.Password, dbConfig.Protocol,
		dbConfig.Address, dbConfig.Dbname, dbConfig.Params)
}
