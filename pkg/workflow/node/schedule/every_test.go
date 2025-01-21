package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_newOutput(t *testing.T) {
	assert := require.New(t)

	parsed, err := time.Parse(time.RFC3339, "2023-01-31T15:46:25+08:00")
	assert.NoError(err)

	got := newOutput(parsed)
	want := everyOutput{
		DateTime:      "2023-01-31T15:46:25+08:00",
		Date:          "2023-01-31",
		Time:          "15:46:25",
		DayOfWeek:     2,
		DayOfWeekName: "Tuesday",
		DateYear:      2023,
		DateMonth:     1,
		DateDay:       31,
		TimeHour:      15,
		TimeMinute:    46,
		TimeSecond:    25,
	}

	assert.Equal(want, got)

	got = newOutput(parsed.UTC())
	want = everyOutput{
		DateTime:      "2023-01-31T07:46:25+00:00",
		Date:          "2023-01-31",
		Time:          "07:46:25",
		DayOfWeek:     2,
		DayOfWeekName: "Tuesday",
		DateYear:      2023,
		DateMonth:     1,
		DateDay:       31,
		TimeHour:      7,
		TimeMinute:    46,
		TimeSecond:    25,
	}

	assert.Equal(want, got)
}
