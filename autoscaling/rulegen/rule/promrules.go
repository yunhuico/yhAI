package rule

import (
	"bytes"
)

type PromRules []PromRule

func (rs PromRules) Marshal() ([]byte, error) {
	if len(rs) == 0 {
		return []byte{}, nil
	}
	var buf bytes.Buffer
	for i, r := range rs {
		data, err := r.Marshal()
		if err != nil {
			return []byte{}, err
		}
		buf.Write(data)
		if i != len(rs)-1 {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes(), nil
}
