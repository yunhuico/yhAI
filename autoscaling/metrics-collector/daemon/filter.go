package daemon

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidConfig returned when filter is not correctly configured
	ErrInvalidFilterConfig = errors.New("invalid filter config")
)

type Filter interface {
	Filter(string) bool
}

// NFVFilter check if line is related with PGW/SGW containers
type NFVFilter struct {
	Indexes []string
}

func NewNFVFilter(indexes []string) (filter *NFVFilter, err error) {
	filter = &NFVFilter{}
	if len(indexes) == 0 {
		err = ErrInvalidFilterConfig
		return
	}
	filter.Indexes = indexes
	return
}

func (f *NFVFilter) Filter(line string) bool {
	for _, index := range f.Indexes {
		if beginWith(line, index) {
			return true
		}
	}
	return false
}

func beginWith(line, prefix string) bool {
	return strings.HasPrefix(line, prefix)
}
