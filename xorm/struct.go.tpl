package {{.Models}}

import (
    "encoding/gob"
    "errors"
    "fmt"
    "database/sql"
{{$ilen := len .Imports}}{{if gt $ilen 0}}{{range .Imports}}"{{.}}"{{end}}{{end}}

    "github.com/go-xorm/xorm"
    "github.com/hsyan2008/hfw"
    "github.com/hsyan2008/hfw/configs"
    "github.com/hsyan2008/hfw/db"
)

{{range .Tables}}
var {{Mapper .Name}}Model = &{{Mapper .Name}}{}
func init() {
	//gob: type not registered for interface
    gob.Register({{Mapper .Name}}Model)
}

type {{Mapper .Name}} struct {
	Dao *db.XormDao `json:"-" xorm:"-"`
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{Mapper $col.Name}}	{{Type $col}} {{Tag $table $col}}
{{end}}
}

func (m *{{Mapper .Name}}) GetDao(c ...interface{}) (d *db.XormDao, err error) {
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
func (m *{{Mapper $table.Name}}) Get{{Mapper $col.Name}}() (val {{Type $col}}) {
    if m == nil {
        return
    }
    return m.{{Mapper $col.Name}}
}
{{if $col.IsAutoIncrement}}
func (m *{{Mapper $table.Name}}) AutoIncrColName() string {
    return "{{$col.Name}}"
}

func (m *{{Mapper $table.Name}}) AutoIncrColValue() (val int64) {
    if m == nil {
        return
    }
    return int64(m.{{Mapper $col.Name}})
}
{{end}}
{{end}}

func (m *{{Mapper .Name}}) String() string {
    return fmt.Sprintf("%#v", m)
}

func (m *{{Mapper .Name}}) TableName() string {
	return "{{.Name}}"
}

func (m *{{Mapper .Name}}) Save(t ...*{{Mapper .Name}}) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    if len(t) > 1 {
        return dao.InsertMulti(t)
    } else {
        var i *{{Mapper .Name}}
        if len(t) == 0 {
            i = m
        } else if len(t) == 1 {
            i = t[0]
        }
	    if i.AutoIncrColValue() > 0 {
		    return dao.UpdateById(i)
    	} else {
            return dao.Insert(i)
    	}
    }
}

func (m *{{Mapper .Name}}) Saves(t []*{{Mapper .Name}}) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.InsertMulti(t)
}

func (m *{{Mapper .Name}}) Insert(t ...*{{Mapper .Name}}) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	if len(t) > 1 {
		return dao.InsertMulti(t)
	} else {
		var i *{{Mapper .Name}}
		if len(t) == 0 {
			i = m
		} else if len(t) == 1 {
			i = t[0]
		}
		return dao.Insert(i)
	}
}

func (m *{{Mapper .Name}}) Update(params db.Cond,
	where db.Cond) (affected int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.UpdateByWhere(m, params, where)
}

func (m *{{Mapper .Name}}) SearchOne(cond db.Cond) (t *{{Mapper .Name}}, err error) {
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

func (m *{{Mapper .Name}}) Search(cond db.Cond) (t []*{{Mapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	err = dao.Search(m, &t, cond)
	return
}

func (m *{{Mapper .Name}}) SearchAndCount(cond db.Cond) (t []*{{Mapper .Name}}, total int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	total, err = dao.SearchAndCount(m, &t, cond)
	return
}

func (m *{{Mapper .Name}}) Rows(cond db.Cond) (rows *xorm.Rows, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Rows(m, cond)
}

func (m *{{Mapper .Name}}) Iterate(cond db.Cond, f xorm.IterFunc) (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Iterate(m, cond, f)
}

func (m *{{Mapper .Name}}) Count(cond db.Cond) (total int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Count(m, cond)
}

func (m *{{Mapper .Name}}) GetMulti(ids ...interface{}) (t []*{{Mapper .Name}}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	err = dao.GetMulti(m, &t, ids...)
	return
}

func (m *{{Mapper .Name}}) GetByIds(ids ...interface{}) (t []*{{Mapper .Name}}, err error) {
	return m.GetMulti(ids...)
}

func (m *{{Mapper .Name}}) GetById(id interface{}) (t *{{Mapper .Name}}, err error) {
	rs, err := m.GetMulti(id)
	if err != nil {
        return
    }
	if len(rs) > 0 {
		t = rs[0]
    }
	return
}

func (m *{{Mapper .Name}}) Replace(cond db.Cond) (i int64, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	defer dao.ClearCache(m)
	return dao.Replace(fmt.Sprintf("REPLACE `%s` SET ", m.TableName()), cond)
}

func (m *{{Mapper .Name}}) Exec(sqlState string, args ...interface{}) (rs sql.Result, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	defer dao.ClearCache(m)
	return dao.Exec(sqlState, args...)
}

func (m *{{Mapper .Name}}) Query(args ...interface{}) (rs []map[string][]byte, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.Query(args...)
}

func (m *{{Mapper .Name}}) QueryString(args ...interface{}) (rs []map[string]string, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.QueryString(args...)
}

func (m *{{Mapper .Name}}) QueryInterface(args ...interface{}) (rs []map[string]interface{}, err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.QueryInterface(args...)
}

//ids可以是数字，也可以是数字切片         
func (m *{{Mapper .Name}}) DeleteByIds(ids interface{}) (i int64, err error) { 
	dao, err := m.GetDao()
	if err != nil {
		return
	}
	return dao.DeleteByIds(m, ids)
}

func (m *{{Mapper .Name}}) Delete(where db.Cond) (i int64, err error) {
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
func New{{Mapper .Name}}IgnoreErr(c ...interface{}) (m *{{Mapper .Name}}) {
    m, _ = New{{Mapper .Name}}(c...)
    return
}

func New{{Mapper .Name}}(c ...interface{}) (m *{{Mapper .Name}}, err error) {
	m = &{{Mapper .Name}}{}
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

func (m *{{Mapper .Name}}) Close() {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    dao.Close()
}

func (m *{{Mapper .Name}}) Begin() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Begin()
}

func (m *{{Mapper .Name}}) Rollback() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Rollback()
}

func (m *{{Mapper .Name}}) Commit() (err error) {
	dao, err := m.GetDao()
	if err != nil {
		return
	}
    return dao.Commit()
}
{{end}}
