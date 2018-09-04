package {{.Models}}

{{$ilen := len .Imports}}
{{if gt $ilen 0}}
import (
    "database/sql"
    "encoding/gob"
    "fmt"
	{{range .Imports}}"{{.}}"{{end}}

    hfw "github.com/hsyan2008/hfw2"
    "github.com/hsyan2008/hfw2/database"
)
{{else}}
import (
    "encoding/gob"
    hfw "github.com/hsyan2008/hfw2"
)
{{end}}

{{range .Tables}}
var {{Mapper .Name}}Model = &{{Mapper .Name}}{Dao: database.NewNoCacheDao(hfw.Config)}
{{end}}

func init() {
	//gob: type not registered for interface
    {{range .Tables}}gob.Register({{Mapper .Name}}Model){{end}}
}

{{range .Tables}}
type {{Mapper .Name}} struct {
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{Mapper $col.Name}}	{{Type $col}} {{Tag $table $col}}
{{end}}
    Dao         *database.NoCacheDao     `json:"-" xorm:"-"`
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

func (m *{{Mapper .Name}}) Update(params hfw.Cond,
	where hfw.Cond) (err error) {
	return m.Dao.UpdateByWhere(m, params, where)
}

func (m *{{Mapper .Name}}) SearchOne(cond hfw.Cond) (t *{{Mapper .Name}}, err error) {
    if cond == nil {
        cond = hfw.Cond{}
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

func (m *{{Mapper .Name}}) Search(cond hfw.Cond) (t []*{{Mapper .Name}}, err error) {
	err = m.Dao.Search(&t, cond)
	return
}

func (m *{{Mapper .Name}}) Count(cond hfw.Cond) (total int64, err error) {
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

func (m *{{Mapper .Name}}) Replace(cond hfw.Cond) (int64, error) {
    return m.Dao.Replace(fmt.Sprintf("REPLACE `%s` SET ", m.TableName()), cond)
}

func (m *{{Mapper .Name}}) Exec(sqlState string, args ...interface{}) (sql.Result, error) {
    return m.Dao.Exec(sqlState, args...)
}

func (m *{{Mapper .Name}}) Query(args ...interface{}) ([]map[string][]byte, error) {
    return m.Dao.Query(args...)
}

func (m *{{Mapper .Name}}) QueryString(args ...interface{}) ([]map[string]string, error) {
    return m.Dao.QueryString(args...)
}
func (m *{{Mapper .Name}}) QueryInterface(args ...interface{}) ([]map[string]interface{}, error) {
    return m.Dao.QueryInterface(args...)
}
{{end}}
