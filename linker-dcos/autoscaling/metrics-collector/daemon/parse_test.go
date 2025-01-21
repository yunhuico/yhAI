package daemon

import (
	"testing"

	"github.com/bmizerany/assert"
	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

func TestParseLine(t *testing.T) {
	var tests = []struct {
		Line   string
		Expect *types.Line
	}{
		{
			Line: string([]byte(`some_index{name="tom",age="10"} 3.14`)),
			Expect: &types.Line{
				Index: "some_index",
				Map: map[string]string{
					"name": "tom",
					"age":  "10",
				},
				FloatVar: 3.14,
			},
		},
	}

	for _, test := range tests {
		got, err := parseLine(test.Line)
		assert.Equal(t, err, nil)
		assert.Equal(t, got, test.Expect)
	}
}

func TestParseFakeJSON(t *testing.T) {
	var tests = []struct {
		Data string
		Map  map[string]string
	}{
		{
			string([]byte(`{name="tom",age="10"}`)),
			map[string]string{
				"name": "tom",
				"age":  "10",
			},
		},
		{
			string([]byte(`{app_container_id="108d77780df9",app_id="/iot/pgw1",group_id="/iot",id="/docker/778fa9d279b4f35fee2e28dac650646bfdaa3da7d502311f7dbfd1f11179644f",image="nginx",mesos_task_id="",name="mesos_mytest",service_group_id="",service_group_instance_id="",service_order_id=""}`)),
			map[string]string{
				"app_container_id":          "108d77780df9",
				"app_id":                    "/iot/pgw1",
				"group_id":                  "/iot",
				"id":                        "/docker/778fa9d279b4f35fee2e28dac650646bfdaa3da7d502311f7dbfd1f11179644f",
				"image":                     "nginx",
				"mesos_task_id":             "",
				"name":                      "mesos_mytest",
				"service_group_id":          "",
				"service_group_instance_id": "",
				"service_order_id":          "",
			},
		},
	}

	for _, test := range tests {
		got, err := parseFakeJSON(test.Data)
		assert.Equal(t, err, nil)
		assert.Equal(t, got, test.Map)
	}
}
