package {{.Model}}

{{$ilen := len .Imports}}
{{if gt $ilen 0}}
import (
    "encoding/gob"
    hfw "github.com/hsyan2008/hfw2"
	{{range .Imports}}"{{.}}"{{end}}
)
{{else}}
import (
    "encoding/gob"
    hfw "github.com/hsyan2008/hfw2"
)
{{end}}

{{range .Tables}}
var {{Mapper .Name}}Model = &{{Mapper .Name}}{Dao: hfw.NewNoCacheDao()}
{{end}}

func init() {
	//gob: type not registered for interface
{{range .Tables}}
	gob.Register({{Mapper .Name}}Model)
{{end}}
}

{{range .Tables}}
type {{Mapper .Name}} struct {
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{Mapper $col.Name}}	{{Type $col}} {{Tag $table $col}}
{{end}}
    Dao         *hfw.NoCacheDao     `json:"-" xorm:"-"`
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

	cond["page"] = 1
	cond["pagesize"] = 1

	rs, err := m.Search(cond)
	if err == nil && len(rs) > 0 {
		t = rs[0]
    }

	return
}

func (m *{{Mapper .Name}}) Search(cond hfw.Cond) (t []*{{Mapper .Name}}, err error) {

	err = m.Dao.Search(&t, cond)

	return
}

func (m *{{Mapper .Name}}) Count(cond hfw.Cond) (total int64, err error) {

	total, err = m.Dao.Count(m, cond)

	return
}

func (m *{{Mapper .Name}}) GetMulti(ids ...interface{}) (t []*{{Mapper .Name}}, err error) {
	err = m.Dao.GetMulti(&t, ids...)

	return
}

//注意，和SearchOne一样，返回的t可能是nil TODO
func (m *{{Mapper .Name}}) GetById(id interface{}) (t *{{Mapper .Name}}, err error) {

	rs, err := m.GetMulti(id)
	if err == nil && len(rs) > 0 {
		t = rs[0]
	}

	return
}

{{end}}
