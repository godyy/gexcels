package parse

import (
	"github.com/ohler55/ojg/sen"
)

var json = &jsonHelper{}

type jsonHelper struct {
}

func (j *jsonHelper) Unmarshal(data []byte, v interface{}) error {
	return sen.Unmarshal(data, v)
}
