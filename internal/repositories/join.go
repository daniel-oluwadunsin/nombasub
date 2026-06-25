package repositories

type Join struct {
	column string
	args   []interface{}
}

func NewJoin(column string, args ...interface{}) *Join {
	return &Join{column: column, args: args}
}
