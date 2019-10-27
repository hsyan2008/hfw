// +build  mssql sqlserver

package db

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"

	//mssql
	_ "github.com/denisenkom/go-mssqldb"
)

func init() {
	dnsFuncMap["mssql"] = getMssqlDns
	dnsFuncMap["sqlserver"] = getMssqlDns
}

func getMssqlDns(dbConfig configs.DbStdConfig) string {
	return fmt.Sprintf("odbc:user id=%s;password=%s;server=%s;port=%s;database=%s;%s",
		dbConfig.Username, dbConfig.Password, dbConfig.Address, dbConfig.Port,
		dbConfig.Dbname, dbConfig.Params)
}
