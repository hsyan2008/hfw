package {{.Models}}

import (
    "encoding/gob"
    "errors"
    "fmt"
    "database/sql"
{{$ilen := len .Imports}}{{if gt $ilen 0}}{{range .Imports}}"{{.}}"{{end}}{{end}}

    "xorm.io/xorm"
    "github.com/hsyan2008/hfw"
    "github.com/hsyan2008/hfw/configs"
    "github.com/hsyan2008/hfw/db"
    "github.com/hsyan2008/hfw/encoding"
)

{{range .Tables}}
var {{TableMapper .Name}}Model = &{{TableMapper .Name}}{}
func init() {
	//gob: type not registered for interface
    gob.Register({{TableMapper .Name}}Model)
}

type {{TableMapper .Name}} struct {
    tableName string `xorm:"-"`
	Dao *db.XormDao `json:"-" xorm:"-"`
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{ColumnMapper $col.Name}}	{{Type $col}} `json:"{{$col.Name}}" {{Tag $table $col}}`
{{end}}
}

func (m *{{TableMapper .Name}}) GetDao(c ...interface{}) (d *db.XormDao, err error) {
	if m == nil {
		return nil, db.ErrDaoNotInited
	}

	if m.Dao != nil {
		return m.Dao, nil
	}

	var dbConfig = hfw.Config.Db
	if len(c) == 0 { //表示全局的model调用
		if d, ok := db.DefaultDao.(*db.XormDao); ok {
			m.Dao = d
			// m.Dao.EnableCache(m)
			m.Dao.DisableCache(m)
			return m.Dao, nil
		}
	} else if len(c) == 1 {
		switch c[0].(type) {
		case configs.DbConfig:
			dbConfig = c[0].(configs.DbConfig)
		case *db.XormDao:
			m.Dao = c[0].(*db.XormDao)
			if m.Dao == nil {
				return nil, errors.New("nil dao")
			}
			return m.Dao, nil
		default:
			return nil, errors.New("error args")
		}
	} else {
		return nil, errors.New("too many args")
	}

	m.Dao, err = db.NewXormDao(hfw.Config, dbConfig)

	return m.Dao, err
}

{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}
func (m *{{TableMapper $table.Name}}) Get{{ColumnMapper $col.Name}}() (val {{Type $col}}) {
    if m == nil {
        return
    }
    return m.{{ColumnMapper $col.Name}}
}
{{if $col.IsAutoIncrement}}
func (m *{{TableMapper $table.Name}}) AutoIncrColName() string {
    return "{{$col.Name}}"
}

func (m *{{TableMapper $table.Name}}) AutoIncrColValue() (val int64) {
    if m == nil {
        return
    }
    return int64(m.{{ColumnMapper $col.Name}})
}
{{end}}
{{end}}

func (m *{{TableMapper .Name}}) String() string {
    b, _ := encoding.JSON.Marshal(m)
    return string(b)
}

func (m *{{TableMapper .Name}}) GoString() string {
    return m.String()
}

func (m *{{TableMapper .Name}}) SetTableName(tableName string) *{{TableMapper .Name}} {
    if m != nil {
        m.tableName = tableName
    }
    return m
}

func (m *{{TableMapper .Name}}) TableName() string {
    if m.tableName == "" {
	    return "{{.Name}}"
    }
    return m.tableName
}

func (m *{{TableMapper .Name}}) IsTableExist(tableName string) (isExist bool, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.IsTableExist(tableName)
}

func (m *{{TableMapper .Name}}) Save(t *{{TableMapper .Name}}, cols ...string) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    if t.AutoIncrColValue() > 0 {
        return dao.UpdateByIds(m, t, []interface{}{t.AutoIncrColValue()}, cols...)
    } else {
        return dao.Insert(m, t)
    }
}

func (m *{{TableMapper .Name}}) Saves(t []*{{TableMapper .Name}}) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.InsertMulti(m, t)
}

func (m *{{TableMapper .Name}}) Insert(t ...*{{TableMapper .Name}}) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	if len(t) > 1 {
		return dao.InsertMulti(m, t)
	} else {
		var i *{{TableMapper .Name}}
		if len(t) == 0 {
			i = m
		} else if len(t) == 1 {
			i = t[0]
		}
		return dao.Insert(m, i)
	}
}

func (m *{{TableMapper .Name}}) Update(params db.Cond,
	where db.Cond) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.UpdateByWhere(m, params, where)
}

//params可以是Cond，也可以是Model，是Model的时候cols才有效
func (m *{{TableMapper .Name}}) UpdateByIds(params interface{},
	ids []interface{}, cols ...string) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.UpdateByIds(m, params, ids, cols...)
}

func (m *{{TableMapper .Name}}) SearchOne(cond db.Cond) (t *{{TableMapper .Name}}, err error) {
    if cond == nil {
        cond = db.Cond{}
    }
	cond["page"] = 1
	cond["pagesize"] = 1

	rs, err := m.Search(cond)
	if err != nil {
        return
    }
	if len(rs) > 0 {
		t = rs[0]
    }

	return
}

func (m *{{TableMapper .Name}}) Search(cond db.Cond) (t []*{{TableMapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	err = dao.Search(m, &t, cond)
	return
}

func (m *{{TableMapper .Name}}) SearchMap(cond db.Cond) (t map[int64]*{{TableMapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    t = make(map[int64]*{{TableMapper .Name}})
	err = dao.Search(m, &t, cond)
	return
}


func (m *{{TableMapper .Name}}) SearchAndCount(cond db.Cond) (t []*{{TableMapper .Name}}, total int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	total, err = dao.SearchAndCount(m, &t, cond)
	return
}

func (m *{{TableMapper .Name}}) SearchMapAndCount(cond db.Cond) (t map[int64]*{{TableMapper .Name}}, total int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    t = make(map[int64]*{{TableMapper .Name}})
	total, err = dao.SearchAndCount(m, &t, cond)
	return
}

func (m *{{TableMapper .Name}}) Rows(cond db.Cond) (rows *xorm.Rows, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Rows(m, cond)
}

func (m *{{TableMapper .Name}}) Iterate(cond db.Cond, f xorm.IterFunc) (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Iterate(m, cond, f)
}

func (m *{{TableMapper .Name}}) Count(cond db.Cond) (total int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Count(m, cond)
}

func (m *{{TableMapper .Name}}) GetByIds(ids []interface{}, cols ...string) (t []*{{TableMapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	err = dao.GetByIds(m, &t, ids, cols...)
	return
}

func (m *{{TableMapper .Name}}) GetMapByIds(ids []interface{}, cols ...string) (t map[int64]*{{TableMapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    t = make(map[int64]*{{TableMapper .Name}})
	err = dao.GetByIds(m, &t, ids, cols...)
	return
}

func (m *{{TableMapper .Name}}) GetById(id interface{}, cols ...string) (t *{{TableMapper .Name}}, err error) {
	rs, err := m.GetByIds([]interface{}{id}, cols...)
	if err != nil {
        return
    }
	if len(rs) > 0 {
		t = rs[0]
    }
	return
}

func (m *{{TableMapper .Name}}) Replace(cond db.Cond) (i int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	defer dao.ClearCache(m)
	return dao.Replace(fmt.Sprintf("REPLACE `%s` SET ", m.TableName()), cond)
}

func (m *{{TableMapper .Name}}) Exec(sqlState string, args ...interface{}) (rs sql.Result, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	defer dao.ClearCache(m)
	return dao.Exec(sqlState, args...)
}

func (m *{{TableMapper .Name}}) Query(args ...interface{}) (rs []map[string][]byte, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Query(args...)
}

func (m *{{TableMapper .Name}}) QueryString(args ...interface{}) (rs []map[string]string, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.QueryString(args...)
}

func (m *{{TableMapper .Name}}) QueryInterface(args ...interface{}) (rs []map[string]interface{}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.QueryInterface(args...)
}

//ids可以是数字，也可以是数字切片         
func (m *{{TableMapper .Name}}) DeleteByIds(ids interface{}) (i int64, err error) { 
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.DeleteByIds(m, ids)
}

func (m *{{TableMapper .Name}}) Delete(where db.Cond) (i int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.DeleteByWhere(m, where)
}

//以下用于事务，注意同个实例不能在多个goroutine同时使用
//使用完毕需要执行Close()，当Close的时候如果没有commit，会自动rollback
//参数只能是0-1个，可以是
//  configs.DbConfig    重新生成dao
//  *db.XormDao         使用现有的dao
//  空                  使用默认的数据库配置
func New{{TableMapper .Name}}IgnoreErr(c ...interface{}) (m *{{TableMapper .Name}}) {
    var err error
    m, err = New{{TableMapper .Name}}(c...)
    if err != nil {
        panic(err)
    }
    return
}

func New{{TableMapper .Name}}(c ...interface{}) (m *{{TableMapper .Name}}, err error) {
	m = &{{TableMapper .Name}}{}
	if len(c) == 0 {
		c = append(c, hfw.Config.Db)
	}
	m.Dao, err = m.GetDao(c...)
	if err != nil {
		return
	}
	m.Dao.NewSession()

	return
}

func (m *{{TableMapper .Name}}) Close() {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    dao.Close()
}

func (m *{{TableMapper .Name}}) Begin() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Begin()
}

func (m *{{TableMapper .Name}}) Rollback() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Rollback()
}

func (m *{{TableMapper .Name}}) Commit() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Commit()
}
{{end}}
