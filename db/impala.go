// +build  impala

package db

import (
	"fmt"

	"github.com/hsyan2008/hfw/configs"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/schemas"

	//impala
	_ "github.com/bippio/go-impala"
)

const (
	impalaScheme                = "impala"
	IMPALA       schemas.DBType = impalaScheme
)

func init() {
	dnsFuncMap[impalaScheme] = getImpalaDns
	dialects.RegisterDriver(impalaScheme, impalaDriver{})
	dialects.RegisterDialect(IMPALA, func() dialects.Dialect {
		return dialects.QueryDialect(schemas.MYSQL)
	})
}

func getImpalaDns(dbConfig configs.DbStdConfig) string {
	if dbConfig.Port != "" {
		dbConfig.Address = fmt.Sprintf("%s:%s", dbConfig.Address, dbConfig.Port)
	}
	return fmt.Sprintf("%s://%s:%s@%s/%s?%s", impalaScheme,
		dbConfig.Username, dbConfig.Password, dbConfig.Address, dbConfig.Dbname, dbConfig.Params)
}

type impalaDriver struct {
}

func (impalaDriver) Parse(driverName, connstr string) (uri *dialects.URI, err error) {

	uri, err = dialects.QueryDriver(string(schemas.MYSQL)).Parse(driverName, connstr)
	if err != nil {
		return
	}
	uri.DBType = IMPALA

	return
}
