package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-xorm/xorm"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw2/common"
	"github.com/hsyan2008/hfw2/configs"
)

var _ Dao = &XormDao{}

func NewXormDao(config configs.AllConfig, dbConfigs ...configs.DbConfig) *XormDao {
	dbConfig := config.Db

	instance := &XormDao{}

	//允许默认配置为空
	if dbConfig.Driver == "" && len(dbConfigs) == 0 {
		return instance
	}

	if len(dbConfigs) > 0 {
		dbConfig = dbConfigs[0]
	}
	instance.engine = InitDb(config, dbConfig)

	//开启缓存
	instance.cacher = GetCacher(config)

	return instance
}

type XormDao struct {
	engine  xorm.EngineInterface
	isCache bool
	cacher  *xorm.LRUCacher
	sess    *xorm.Session
}

func (d *XormDao) UpdateById(t interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	id := reflect.ValueOf(t).Elem().FieldByName("Id").Int()
	_, err = sess.Id(id).AllCols().Update(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) UpdateByIds(t interface{}, params map[string]interface{},
	ids []interface{}) (err error) {

	if len(ids) == 0 {
		return errors.New("ids parameters error")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	_, err = sess.Table(t).In("id", ids).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) UpdateByWhere(t interface{}, params map[string]interface{},
	where map[string]interface{}) (err error) {
	if len(where) == 0 {
		return errors.New("where paramters error")
	}

	var (
		str  []string
		args []interface{}
	)
	for k, v := range where {
		str = append(str, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	_, err = sess.Table(t).Where(strings.Join(str, " AND "), args...).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return err
}

func (d *XormDao) Insert(t interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	_, err = sess.Insert(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) InsertMulti(t interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	_, err = sess.InsertMulti(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) SearchOne(t interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	_, err = sess.Get(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) buildCond(sess *xorm.Session, cond map[string]interface{}) (session *xorm.Session, err error) {
	var (
		str      []string
		orderby  = "id desc"
		page     = 1
		pageSize = DefaultPageSize
		where    string
		args     []interface{}
	)
FOR:
	for k, v := range cond {
		k = strings.ToLower(k)
		switch k {
		case "orderby":
			orderby = v.(string)
			continue FOR
		case "page":
			page = common.Max(v.(int), page)
			continue FOR
		case "pagesize":
			if v.(int) > 0 {
				pageSize = v.(int)
			}
			continue FOR
		case "where":
			where = v.(string)
			continue FOR
		case "sql":
			sess.SQL(v)
			continue FOR
		case "select":
			sess.Select(v.(string))
			continue FOR
		case "distinct":
			sess.Distinct(v.(string))
			continue FOR
		}

		keys := strings.Fields(k)
		switch len(keys) {
		case 1:
			str = append(str, fmt.Sprintf("`%s` = ?", k))
			args = append(args, v)
			continue FOR
		case 2:
			switch keys[1] {
			case "in":
				sess.In(keys[0], v.([]interface{}))
			default:
				return nil, errors.New("error cond key")
			}
			continue FOR
		default:
			return nil, errors.New("error cond")
		}
	}

	var strs string
	if len(str) > 0 {
		strs = strings.Join(str, " AND ")
	}

	if len(where) > 0 && len(strs) > 0 {
		where = fmt.Sprintf("(%s) AND %s", where, strs)
	} else if len(strs) > 0 {
		where = strs
	}

	sess.Where(where, args...).OrderBy(orderby).
		Limit(pageSize, (page-1)*pageSize)

	return sess, nil
}

func (d *XormDao) Search(t interface{}, cond map[string]interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(sess, cond)
	if err != nil {
		return
	}

	err = sess.Find(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Rows(t interface{}, cond map[string]interface{}) (rows *xorm.Rows, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(sess, cond)
	if err != nil {
		return
	}

	rows, err = sess.Rows(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Iterate(t interface{}, cond map[string]interface{}, f xorm.IterFunc) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(sess, cond)
	if err != nil {
		return
	}

	err = sess.Iterate(t, f)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) GetMulti(t interface{}, ids ...interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	err = sess.In("id", ids...).Find(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Count(t interface{}, cond map[string]interface{}) (total int64, err error) {
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
		str = append(str, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	if where != "" {
		where = fmt.Sprintf("(%s) AND %s", where, strings.Join(str, " AND "))
	} else {
		where = strings.Join(str, " AND ")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	total, err = sess.Where(where, args...).Count(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Replace(sql string, cond map[string]interface{}) (id int64, err error) {
	var (
		str  []string
		args []interface{}
	)
	for k, v := range cond {
		k = strings.ToLower(k)
		if k == "orderby" || k == "page" || k == "pagesize" || k == "where" {
			continue
		}
		str = append(str, fmt.Sprintf("`%s` = ?", k))
		args = append(args, v)
	}

	rs, err := d.Exec(sql+strings.Join(str, ", "), args...)
	if err != nil {
		return
	}

	return rs.LastInsertId()
}

//调用方必须确保执行Exec后，再执行ClearCache
func (d *XormDao) Exec(sqlStr string, args ...interface{}) (rs sql.Result, err error) {
	tmp := make([]interface{}, 0)
	tmp = append(tmp, sqlStr)
	tmp = append(tmp, args...)

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	rs, err = sess.Exec(args...)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Query(args ...interface{}) (rs []map[string][]byte, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	rs, err = sess.Query(args...)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) QueryString(args ...interface{}) (rs []map[string]string, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	rs, err = sess.QueryString(args...)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) QueryInterface(args ...interface{}) (rs []map[string]interface{}, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	rs, err = sess.QueryInterface(args...)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) EnableCache(t interface{}) {
	if d.cacher != nil {
		d.isCache = true
		_ = d.engine.MapCacher(t, d.cacher)
	}
}

func (d *XormDao) DisableCache(t interface{}) {
	d.isCache = false
	_ = d.engine.MapCacher(t, nil)
}

//用于清理缓存
func (d *XormDao) ClearCache(t interface{}) {
	if d.isCache {
		_ = d.engine.ClearCache(t)
	}
}

//以下主要用于事务
//用法
//首先NewSession，然后defer Close
//然后Begin，如果不Commit，会自动在Close里Rollback掉
func (d *XormDao) NewSession() {
	d.sess = d.engine.NewSession()
}

func (d *XormDao) Close() {
	if d.sess != nil {
		d.sess.Close()
		d.sess = nil
	}
}

func (d *XormDao) Begin() error {
	if d.sess == nil {
		return errors.New("please NewSession at first")
	}

	return d.sess.Begin()
}

func (d *XormDao) Rollback() error {
	return d.sess.Rollback()
}

func (d *XormDao) Commit() error {
	return d.sess.Commit()
}
