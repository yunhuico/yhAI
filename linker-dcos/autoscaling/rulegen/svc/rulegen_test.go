package svc

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestGenRuleFile(t *testing.T) {
	rule1 := readFile(t, "expect-rulefile.01")
	rule2 := readFile(t, "expect-rulefile.02")
	var cases = []struct {
		CPUEnable         bool
		MemEnable         bool
		CPUHigh           float32
		CPULow            float32
		MemHigh           float32
		MemLow            float32
		Duration          string
		ExpectFileContent []byte
		ExpectErr         error
	}{
		{false, false, 0, 0, 0, 0, "", []byte{}, nil},
		{false, false, 80, 20, 60, 40, "", []byte{}, nil},
		{true, false, 80, 20, 0, 0, "10m", rule1, nil},
		{false, true, 0, 0, 81, 21, "15m", rule2, nil},
		{false, false, 0, 0, 0, 0, "", []byte{}, nil},
		{false, false, 1, 2, 3, 4, "anything", []byte{}, nil},
	}
	for _, c := range cases {
		gotFileContent, gotErr := genRuleFile(c.CPUEnable, c.MemEnable, c.CPUHigh, c.CPULow, c.MemHigh, c.MemLow, c.Duration)
		// fmt.Println(string(c.ExpectFileContent))
		// fmt.Println(string(gotFileContent))
		assert.Equal(t, c.ExpectErr, gotErr)
		assert.Equal(t, c.ExpectFileContent, gotFileContent)
	}
}
