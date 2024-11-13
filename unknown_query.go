package tarantool

type UnknownQuery struct {
	cmd  uint
	data []byte
}

var _ Query = (*UnknownQuery)(nil)

func NewUnknownQuery(cmd uint) *UnknownQuery {
	return &UnknownQuery{cmd: cmd}
}

func (q *UnknownQuery) GetCommandID() uint {
	return q.cmd
}

func (q *UnknownQuery) MarshalMsg(b []byte) ([]byte, error) {
	return append(b, q.data...), nil
}

func (q *UnknownQuery) UnmarshalMsg(data []byte) (buf []byte, err error) {
	q.data = make([]byte, len(data))
	copy(q.data, data)

	return nil, nil
}

// IsKnownQuery returns true if passed query is known and supported.
func IsKnownQuery(q Query) bool {
	_, unknown := q.(*UnknownQuery)

	return q != nil && !unknown
}
