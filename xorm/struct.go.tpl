package {{.Models}}

import (
    "encoding/gob"
    "fmt"
{{$ilen := len .Imports}}
{{if gt $ilen 0}}
	{{range .Imports}}{{if ne . "time"}}"{{.}}"{{end}}{{end}}
{{end}}
    "github.com/go-xorm/xorm"
    hfw "github.com/hsyan2008/hfw2"
    "github.com/hsyan2008/hfw2/db"
)

{{range .Tables}}
var {{Mapper .Name}}Model = &{{Mapper .Name}}{}
{{end}}

func init() {
    {{range .Tables}}{{Mapper .Name}}Model.Dao = db.NewNoCacheDao(hfw.Config)
	//gob: type not registered for interface
    gob.Register({{Mapper .Name}}Model){{end}}
}

{{range .Tables}}
type {{Mapper .Name}} struct {
    db.Models `xorm:"extends"`
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{if eq $col.Name "id" "is_deleted" "updated_at" "created_at"}}{{else}}{{Mapper $col.Name}}	{{Type $col}} {{Tag $table $col}}{{end}}
{{end}}
}

func (m *{{Mapper .Name}}) TableName() string {
	return "{{.Name}}"
}

func (m *{{Mapper .Name}}) Save(t *{{Mapper .Name}}) (err error) {
	if t.Id > 0 {
		err = m.Dao.UpdateById(t)
	} else {
		err = m.Dao.Insert(t)
	}
	return
}

func (m *{{Mapper .Name}}) Saves(t []*{{Mapper .Name}}) (err error) {
    return m.Dao.Insert(t)
}

func (m *{{Mapper .Name}}) Update(params db.Cond,
	where db.Cond) (err error) {
	return m.Dao.UpdateByWhere(m, params, where)
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
    } else {
        t = new({{Mapper .Name}})
    }
	return
}

func (m *{{Mapper .Name}}) Search(cond db.Cond) (t []*{{Mapper .Name}}, err error) {
	err = m.Dao.Search(&t, cond)
	return
}

func (m *{{Mapper .Name}}) Rows(cond db.Cond) (rows *xorm.Rows, err error) {
	return m.Dao.Rows(m, cond)
}

func (m *{{Mapper .Name}}) Iterate(cond db.Cond, f xorm.IterFunc) (err error) {
	return m.Dao.Iterate(m, cond, f)
}

func (m *{{Mapper .Name}}) Count(cond db.Cond) (total int64, err error) {
	return m.Dao.Count(m, cond)
}

func (m *{{Mapper .Name}}) GetMulti(ids ...interface{}) (t []*{{Mapper .Name}}, err error) {
	err = m.Dao.GetMulti(&t, ids...)
	return
}

func (m *{{Mapper .Name}}) GetById(id interface{}) (t *{{Mapper .Name}}, err error) {
	rs, err := m.GetMulti(id)
	if err != nil {
        return
    }
	if len(rs) > 0 {
		t = rs[0]
    } else {
        t = new({{Mapper .Name}})
    }
	return
}

func (m *{{Mapper .Name}}) Replace(cond db.Cond) (int64, error) {
    return m.Dao.Replace(fmt.Sprintf("REPLACE `%s` SET ", m.TableName()), cond)
}
{{end}}
