package rule

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestConditionsString(t *testing.T) {
	var testCases = []struct {
		Conditions   Conditions
		ExpectString string
	}{
		{
			Conditions{},
			"",
		},
		{
			*NewConditions(Condition{Index: "index_a", CompareSym: ">", Threshold: 1}),
			"index_a > 1",
		},
		{
			*NewConditions(Condition{Index: "index_b", CompareSym: "<=", Threshold: 0.5}),
			"index_b <= 0.5",
		},
		{
			*NewConditions(Condition{Index: "index_a", CompareSym: ">", Threshold: 1}).
				And(Condition{Index: "index_b", CompareSym: "<=", Threshold: 0.5}),
			"(index_a > 1 AND index_b <= 0.5)",
		},
		{
			*NewConditions(Condition{Index: "index_c", CompareSym: "==", Threshold: 0}).
				Or(Condition{Index: "index_d", CompareSym: "<", Threshold: 1}),
			"(index_c == 0 OR index_d < 1)",
		},
		{
			*NewConditions(Condition{Index: "index_a", CompareSym: ">", Threshold: 1}).
				And(Condition{Index: "index_b", CompareSym: "<=", Threshold: 0.5}).
				And(Condition{Index: "index_c", CompareSym: "==", Threshold: 0}),
			"(index_a > 1 AND index_b <= 0.5 AND index_c == 0)",
		},
		{
			*NewConditions(Condition{Index: "index_a", CompareSym: ">", Threshold: 1}).
				And(Condition{Index: "index_b", CompareSym: "<=", Threshold: 0.5}).
				Or(Condition{Index: "index_c", CompareSym: "==", Threshold: 0}),
			"(index_a > 1 AND index_b <= 0.5 OR index_c == 0)", // not recommanded to use
		},
	}

	for _, tc := range testCases {
		gotString := tc.Conditions.String()
		assert.Equal(t, tc.ExpectString, gotString)
	}
}
