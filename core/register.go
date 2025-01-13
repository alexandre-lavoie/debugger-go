package core

type Register struct {
	Name    string
	Index   uint
	BitSize uint
	Type    string
}

type RegisterValue struct {
	Definition *Register
	Value      []byte
}

func (r *RegisterValue) ToUint() uint {
	o := uint(0)

	for i := len(r.Value) - 1; i >= 0; i-- {
		o <<= 8
		o |= uint(r.Value[i])
	}

	return o
}
