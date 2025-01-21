package compare

import (
	"fmt"
	"time"

	"github.com/araddon/dateparse"
)

type compareTime struct {
	time.Time
	err error
}

func toTime(datetime string) *compareTime {
	t, err := dateparse.ParseAny(datetime)
	if err != nil {
		return &compareTime{err: fmt.Errorf("transform %s to time: %w", datetime, err)}
	}
	return &compareTime{Time: t}
}

func (t *compareTime) before(other time.Time) (bool, error) {
	if t.err != nil {
		return false, t.err
	}

	return t.Before(other), nil
}
