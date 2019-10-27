// +build  postgres

package db

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"

	//postgresql
	_ "github.com/lib/pq"
)

func init() {
	dnsFuncMap["postgres"] = getPostgresDns
}

func getPostgresDns(dbConfig configs.DbStdConfig) string {
	// if dbConfig.Port != "" {
	// 	dbConfig.Address = fmt.Sprintf("%s:%s", dbConfig.Address, dbConfig.Port)
	// }
	// return fmt.Sprintf("postgres://%s:%s@%s/%s%s",
	// 	dbConfig.Username, dbConfig.Password, dbConfig.Address, dbConfig.Dbname, dbConfig.Params)
	return fmt.Sprintf("host=%s port=%s user=%s password='%s' dbname=%s %s",
		dbConfig.Address, dbConfig.Port, dbConfig.Username, dbConfig.Password,
		dbConfig.Dbname, dbConfig.Params)
}
