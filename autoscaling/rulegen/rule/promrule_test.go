package rule

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestPromRuleMarshal(t *testing.T) {
	// ALERT HighMemoryAlert
	//   IF container_memory_usage_high_result > 1
	//   FOR 30s
	//   ANNOTATIONS {
	//     summary = "High Memory usage alert for container",
	//     description = "High Memory usage for linker container on {{$labels.image}} for container {{$labels.name}} (current value: {{$value}})",
	//   }
	rule1 := PromRule{
		AlertName:  "HighMemoryAlert",
		Conditions: *NewConditions(Condition{Index: "container_memory_usage_high_result", CompareSym: ">", Threshold: 1}),
		Duration:   "30s",
		Annotations: map[string]string{
			"summary":     "High Memory usage alert for container",
			"description": "High Memory usage for linker container on {{$labels.image}} for container {{$labels.name}} (current value: {{$value}})",
		},
	}
	expectData01 := readFile(t, "expect-data-01.rule")

	// ALERT LowMemoryAlert
	//   IF (container_memory_usage_low_result < 1 AND container_memory_usage_low_result > 0)
	//   FOR 1m
	//   ANNOTATIONS {
	//     summary = "Low Memory usage alert for container",
	//     description = "Low Memory usage for linker container on {{$labels.image}} for container {{$labels.name}} (current value: {{$value}})",
	//   }
	rule2 := PromRule{
		AlertName: "LowMemoryAlert",
		Conditions: *NewConditions(Condition{Index: "container_memory_usage_low_result", CompareSym: "<", Threshold: 1}).
			And(Condition{Index: "container_memory_usage_low_result", CompareSym: ">", Threshold: 0}),
		Duration: "1m",
		Annotations: map[string]string{
			"summary":     "Low Memory usage alert for container",
			"description": "Low Memory usage for linker container on {{$labels.image}} for container {{$labels.name}} (current value: {{$value}})",
		},
	}
	expectData02 := readFile(t, "expect-data-02.rule")
	var cases = []struct {
		PromRule   PromRule
		ExpectData []byte
		ExpectErr  error
	}{
		{PromRule{}, []byte{}, nil},
		{rule1, expectData01, nil},
		{rule2, expectData02, nil},
	}
	for _, c := range cases {
		gotData, gotErr := c.PromRule.Marshal()
		// fmt.Println(string(gotData))
		// fmt.Println(string(c.ExpectData))
		assert.Equal(t, c.ExpectErr, gotErr)
		assert.Equal(t, c.ExpectData, gotData)
	}
}
