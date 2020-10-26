// +build  impala

package db

import (
	"testing"

	"github.com/hsyan2008/hfw/configs"
)

type OdsPositionInfo struct {
	Id     string `json:"ID" xorm:"ID pk autoincr BIGINT(20)"`
	Uuid   string `json:"UUID" xorm:"UUID VARCHAR(256)"`
	SiteId string `json:"SITE_ID" xorm:"SITE_ID VARCHAR(256)"`
}

//go test -run TestImpalaConnect -tags=impala ./... -v
func TestImpalaConnect(t *testing.T) {
	engine, _, err := getEngine(configs.DbStdConfig{
		Driver:   "impala",
		Username: "",
		Password: "",
		Protocol: "",
		Address:  "192.168.0.165",
		Port:     "21050",
		Dbname:   "ods",
		Params:   "",
	})
	if err != nil {
		t.Fatal(err)
	}
	xormLog := newXormLog()
	xormLog.ShowSQL(true)
	engine.SetLogger(xormLog)
	var u []*OdsPositionInfo
	//必须指定dbname，暂时没有解决 TODO
	err = engine.AllCols().Table("ods.ods_position_info").Limit(10).Find(&u)
	if err != nil {
		t.Fatal(err)
	}

	if len(u) > 0 {
		t.Log(u[0])
	}
}
