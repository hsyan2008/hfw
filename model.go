package hfw

type Cond map[string]interface{}

//暂时没用
type Model interface {
	TableName() string
	Save(*Model) error
	Update(Cond, Cond) error
	SearchOne(Cond) (*Model, error)
	Search(Cond) ([]*Model, error)
	Count(Cond) (int64, error)
	GetMulti(...interface{}) ([]*Model, error)
	GetById(interface{}) (*Model, error)
}
