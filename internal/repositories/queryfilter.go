package repositories

type QueryFilter struct {
	query string
	args  []interface{}
}

func NewQueryFilter() *QueryFilter {
	return &QueryFilter{}
}

func (q *QueryFilter) Where(query string, args ...interface{}) *QueryFilter {
	q.query = query
	q.args = args

	return q
}
