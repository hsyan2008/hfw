package {{.Model}}

{{$ilen := len .Imports}}
{{if gt $ilen 0}}
import (
	"time"
	{{range .Imports}}"{{.}}"{{end}}
)
{{else}}
import (
	"time"
)
{{end}}

{{range .Tables}}
type {{Mapper .Name}} struct {
{{$table := .}}
{{range .ColumnsSeq}}{{$col := $table.GetColumn .}}	{{Mapper $col.Name}}	{{Type $col}} {{Tag $table $col}}
{{end}}
}

func (m *{{Mapper .Name}}) TableName() string {

	return "{{.Name}}"
}

func (m *{{Mapper .Name}}) Save(t *{{Mapper .Name}}) (err error) {

	t.UpdatedAt = int(time.Now().Unix())
	if t.Id > 0 {
		err = dao.UpdateById(t)
	} else {
		t.CreatedAt = int(time.Now().Unix())
		err = dao.Insert(t)
	}

	return
}

func (m *{{Mapper .Name}}) Update(params Cond,
	where Cond) (err error) {

	params["updated_at"] = time.Now().Unix()
	return dao.UpdateByWhere(m, params, where)
}

func (m *{{Mapper .Name}}) SearchOne(cond Cond) (t *{{Mapper .Name}}, err error) {

	cond["page"] = 1
	cond["pagesize"] = 1

	rs, err := m.Search(cond)
	if err == nil && len(rs) > 0 {
		t = rs[0]
    }

	return
}

func (m *{{Mapper .Name}}) Search(cond Cond) (t []*{{Mapper .Name}}, err error) {

	err = dao.Search(&t, cond)

	return
}

func (m *{{Mapper .Name}}) Count(cond Cond) (total int64, err error) {

	total, err = dao.Count(m, cond)

	return
}

func (m *{{Mapper .Name}}) GetMulti(ids ...interface{}) (t []*{{Mapper .Name}}, err error) {
	err = dao.GetMulti(&t, ids...)

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
