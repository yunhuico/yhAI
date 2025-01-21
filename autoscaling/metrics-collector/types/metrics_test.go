package types

import (
	"github.com/bmizerany/assert"
	"testing"
)

func TestLineString(t *testing.T) {
	var tests = []struct {
		Line   *Line
		Expect string
	}{
		{
			&Line{
				Index: "some_index",
				Map: map[string]string{
					"name": "tom",
					"age":  "10",
				},
				FloatVar: 3.14,
			},
			"some_index{age=\"10\",name=\"tom\"} 3.140000e+00\n",
		},
	}

	for _, test := range tests {
		// got := fmt.Sprintf("%s", test.Line)
		got := test.Line.String()
		assert.Equal(t, got, test.Expect)
	}
}
