package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-xorm/xorm"
	logger "github.com/hsyan2008/go-logger"
	"github.com/hsyan2008/hfw/common"
	"github.com/hsyan2008/hfw/configs"
)

var _ Dao = &XormDao{}

//config用于缓存，dbConfig用于数据库配置
func NewXormDao(config configs.AllConfig, dbConfig configs.DbConfig) (instance *XormDao, err error) {

	if dbConfig.Driver == "" {
		return nil, errors.New("nil db config")
	}

	instance = &XormDao{}

	instance.engine, err = InitDb(config, dbConfig)
	if err != nil {
		return nil, err
	}

	//开启缓存
	if len(dbConfig.CacheType) > 0 {
		instance.cacher, err = GetCacher(config, dbConfig)
	}

	return instance, err
}

type XormDao struct {
	engine  xorm.EngineInterface
	isCache bool
	cacher  *xorm.LRUCacher
	sess    *xorm.Session
}

func (d *XormDao) IsTableExist(beanOrTableName interface{}) (bool, error) {
	return d.engine.IsTableExist(beanOrTableName)
}

func (d *XormDao) UpdateById(m, t Model) (affected int64, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	affected, err = sess.Table(m).Id(t.AutoIncrColValue()).AllCols().Update(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) UpdateByIds(t Model, params Cond,
	ids []interface{}) (affected int64, err error) {

	if len(ids) == 0 {
		return 0, errors.New("ids parameters error")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	affected, err = sess.Table(t).In(t.AutoIncrColName(), ids).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) UpdateByWhere(t Model, params Cond,
	where Cond) (affected int64, err error) {
	if len(where) == 0 {
		return 0, errors.New("where paramters error")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, where, false, false)
	if err != nil {
		return
	}

	affected, err = sess.Table(t).Update(params)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) Insert(m, t Model) (affected int64, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	affected, err = sess.Table(m).Insert(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) InsertMulti(m Model, t interface{}) (affected int64, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	affected, err = sess.Table(m).InsertMulti(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) SearchOne(t Model, cond Cond) (has bool, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, true, false)
	if err != nil {
		return
	}

	has, err = sess.Get(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) buildCond(t Model, sess *xorm.Session, cond Cond, isOrder, isPaging bool) (session *xorm.Session, err error) {
	var (
		str      []string
		orderby  = fmt.Sprintf("%s desc", t.AutoIncrColName())
		page     = 1
		pageSize = DefaultPageSize
		where    string
		args     []interface{}
	)
	sess = sess.Table(t)
FOR:
	for k, v := range cond {
		k = strings.ToLower(k)
		switch k {
		//select all
		case "nolimit":
			isPaging = false
			continue FOR
		case "orderby":
			if isOrder {
				if s, ok := v.(string); ok && len(s) > 0 {
					orderby = s
				}
			}
			continue FOR
		case "groupby":
			if s, ok := v.(string); ok && len(s) > 0 {
				sess.GroupBy(s)
			}
			continue FOR
		case "having":
			if s, ok := v.(string); ok && len(s) > 0 {
				sess.Having(s)
			}
			continue FOR
		case "page":
			if isPaging {
				page = common.Max(common.ConvertToInt(v), page)
			}
			continue FOR
		case "pagesize":
			if isPaging {
				ps := common.ConvertToInt(v)
				if ps > 0 {
					pageSize = ps
				}
			}
			continue FOR
		case "where":
			if s, ok := v.(string); ok && len(s) > 0 {
				where = s
			}
			continue FOR
		case "sql":
			sess.SQL(v)
			continue FOR
		case "select":
			if s, ok := v.(string); ok && len(s) > 0 {
				sess.Select(s)
			}
			continue FOR
		case "distinct":
			if s, ok := v.(string); ok && len(s) > 0 {
				sess.Distinct(s)
			}
			continue FOR
		case "forupdate":
			sess.ForUpdate()
			continue FOR
		}

		keys := strings.Fields(strings.TrimSpace(k))
		switch len(keys) {
		case 1:
			str = append(str, fmt.Sprintf("`%s` = ?", k))
			args = append(args, v)
			continue FOR
		case 2:
			switch keys[1] {
			case "in":
				sess.In(keys[0], v)
			case "notin":
				sess.NotIn(keys[0], v)
			case "like":
				//key: name like
				//val: %h%
				str = append(str, fmt.Sprintf("`%s` like ?", keys[0]))
				args = append(args, v)
			default: // > >= < <=等
				str = append(str, fmt.Sprintf("%s ?", k))
				args = append(args, v)
				// return nil, errors.New("error cond key")
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

	sess.Where(where, args...)
	if isOrder {
		sess.OrderBy(orderby)
	}
	if isPaging {
		sess.Limit(pageSize, (page-1)*pageSize)
	}

	return sess, nil
}

func (d *XormDao) Search(t Model, ts interface{}, cond Cond) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, true, true)
	if err != nil {
		return
	}

	err = sess.Find(ts)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) SearchAndCount(t Model, ts interface{}, cond Cond) (total int64, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, true, true)
	if err != nil {
		return
	}

	total, err = sess.FindAndCount(ts)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Rows(t Model, cond Cond) (rows *xorm.Rows, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, true, true)
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

func (d *XormDao) Iterate(t Model, cond Cond, f xorm.IterFunc) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, true, true)
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

func (d *XormDao) GetMulti(t Model, ts interface{}, ids ...interface{}) (err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	err = sess.Table(t).In(t.AutoIncrColName(), ids...).Find(ts)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Count(t Model, cond Cond) (total int64, err error) {
	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, cond, false, false)
	if err != nil {
		return
	}

	total, err = sess.Count(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) Replace(sql string, cond Cond) (id int64, err error) {
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
	tmp := append([]interface{}{sqlStr}, args...)

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	rs, err = sess.Exec(tmp...)
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

func (d *XormDao) DeleteByIds(t Model, ids interface{}) (affected int64, err error) {

	if ids == nil {
		return 0, errors.New("ids parameters error")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}

	affected, err = sess.Table(t).In(t.AutoIncrColName(), ids).Unscoped().Delete(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}

	return
}

func (d *XormDao) DeleteByWhere(t Model, where Cond) (affected int64, err error) {
	if len(where) == 0 {
		return 0, errors.New("where paramters error")
	}

	sess := d.sess
	if sess == nil {
		sess = d.engine.NewSession()
		defer sess.Close()
	}
	sess, err = d.buildCond(t, sess, where, false, false)
	if err != nil {
		return
	}

	affected, err = sess.Unscoped().Delete(t)
	if err != nil {
		lastSQL, lastSQLArgs := sess.LastSQL()
		logger.Error(err, lastSQL, lastSQLArgs)
	}
	return
}

func (d *XormDao) EnableCache(t Model) {
	if d.cacher != nil {
		d.isCache = true
		_ = d.engine.MapCacher(t, d.cacher)
	}
}

func (d *XormDao) DisableCache(t Model) {
	d.isCache = false
	_ = d.engine.MapCacher(t, nil)
}

//用于清理缓存
func (d *XormDao) ClearCache(t Model) {
	if d.isCache {
		_ = d.engine.ClearCache(t)
	}
}

//以下主要用于事务
//用法
//首先NewSession，然后defer Close
//然后Begin，如果不Commit，会自动在Close里Rollback掉
//Notice: 注意并发不安全，请勿在全局上使用
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
	if d.sess == nil {
		return errors.New("please NewSession at first")
	}

	return d.sess.Rollback()
}

func (d *XormDao) Commit() error {
	if d.sess == nil {
		return errors.New("please NewSession at first")
	}

	return d.sess.Commit()
}
