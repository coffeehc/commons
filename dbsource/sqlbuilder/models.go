package sqlbuilder

type TableCount struct {
	Count int64 `json:"count,omitempty"`
}

type TableId struct {
	Id int64 `json:"id,omitempty" db:"id"`
}

type SqlContext struct {
	Sql    string
	Params []interface{}
}
