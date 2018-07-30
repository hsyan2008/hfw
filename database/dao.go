package database

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-xorm/xorm"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

const DefaultPageSize = 1000

type Dao interface {
	UpdateById(interface{}) error
	UpdateByIds(interface{}, map[string]interface{}, []interface{}) error
	UpdateByWhere(interface{}, map[string]interface{}, map[string]interface{}) error
	Insert(interface{}) error
	SearchOne(interface{}) error
	Search(interface{}, map[string]interface{}) error
	GetMulti(interface{}, ...interface{}) error
	Count(interface{}, map[string]interface{}) (int64, error)
}

var _ Dao = &NoCacheDao{}

func NewNoCacheDao(config configs.AllConfig, dbConfigs ...configs.DbConfig) *NoCacheDao {
	dbConfig := config.Db

	instance := &NoCacheDao{}

	//允许默认配置为空
	if dbConfig.Driver == "" && len(dbConfigs) == 0 {
		return instance
	}

	if len(dbConfigs) > 0 {
		dbConfig = dbConfigs[0]
	}
	instance.engine = ConnectDb(config, dbConfig)

	return instance
}

type NoCacheDao struct {
	engine *xorm.Engine
}

func (d *NoCacheDao) UpdateById(t interface{}) (err error) {
	sess := d.engine.NewSession()
	defer sess.Close()

	id := reflect.ValueOf(t).Elem().FieldByName("Id").Int()
	_, err = sess.Id(id).AllCols().Update(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *NoCacheDao) UpdateByIds(t interface{}, params map[string]interface{},
	ids []interface{}) (err error) {

	if len(ids) == 0 {
		return errors.New("ids parameters error")
	}

	sess := d.engine.NewSession()
	defer sess.Close()
	_, err = sess.Table(t).In("id", ids).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *NoCacheDao) UpdateByWhere(t interface{}, params map[string]interface{},
	where map[string]interface{}) (err error) {
	if len(where) == 0 {
		return errors.New("where paramters error")
	}
	sess := d.engine.NewSession()
	defer sess.Close()

	var (
		str  []string
		args []interface{}
	)
	for k, v := range where {
		str = append(str, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}

	_, err = sess.Table(t).Where(strings.Join(str, " "), args...).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return err
}

func (d *NoCacheDao) Insert(t interface{}) (err error) {
	sess := d.engine.NewSession()
	defer sess.Close()
	_, err = sess.Insert(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *NoCacheDao) SearchOne(t interface{}) (err error) {
	sess := d.engine.NewSession()
	defer sess.Close()
	_, err = sess.Get(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *NoCacheDao) Search(t interface{}, cond map[string]interface{}) (err error) {
	sess := d.engine.NewSession()
	defer sess.Close()
	var (
		str      []string
		args     []interface{}
		orderby  = "id desc"
		page     = 1
		pageSize = DefaultPageSize
		where    string
	)
	for k, v := range cond {
		k = strings.ToLower(k)
		if k == "orderby" {
			orderby = v.(string)
			continue
		}
		if k == "page" {
			page = common.Max(v.(int), page)
			continue
		}
		if k == "pagesize" {
			if v.(int) > 0 {
				pageSize = v.(int)
			}
			continue
		}
		if k == "where" {
			where = v.(string)
			continue
		}
		str = append(str, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}

	if where != "" {
		where = fmt.Sprintf("(%s) AND %s", where, strings.Join(str, " AND "))
	} else {
		where = strings.Join(str, " AND ")
	}
	err = sess.Where(where, args...).OrderBy(orderby).
		Limit(pageSize, (page-1)*pageSize).Find(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return err
}

func (d *NoCacheDao) GetMulti(t interface{}, ids ...interface{}) (err error) {
	sess := d.engine.NewSession()
	defer sess.Close()

	err = sess.In("id", ids...).Find(t)

	return
}

func (d *NoCacheDao) Count(t interface{}, cond map[string]interface{}) (total int64, err error) {
	sess := d.engine.NewSession()
	defer sess.Close()
	var (
		str   []string
		args  []interface{}
		where string
	)
	for k, v := range cond {
		k = strings.ToLower(k)
		if k == "orderby" || k == "page" || k == "pagesize" {
			continue
		}
		if k == "where" {
			where = v.(string)
			continue
		}
		str = append(str, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}

	if where != "" {
		where = fmt.Sprintf("(%s) AND %s", where, strings.Join(str, " AND "))
	} else {
		where = strings.Join(str, " AND ")
	}
	total, err = sess.Where(where, args...).Count(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}
