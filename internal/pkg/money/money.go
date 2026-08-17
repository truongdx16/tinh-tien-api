package money

// VND is stored as integer đồng to avoid floating point issues.
type VND int64

func (v VND) Int64() int64 {
	return int64(v)
}
