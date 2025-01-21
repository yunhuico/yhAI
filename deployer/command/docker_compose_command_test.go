package command

import (
	"testing"

	"github.com/bmizerany/assert"
)

// replace cases when you test, http request will be actually sent
func TestBuildSwarmEndpoints(t *testing.T) {
	list1 := []string{"linker_hostname_3ce4e36b_9c34_4e15_a430_02dbab06e571_insecure_sysadmin=54.238.158.10",
		"linker_hostname_e2548810_439e_4dc9_abeb_237c3e78ed30_insecure_sysadmin=54.199.246.90",
		"linker_hostname_e64d6086_6f2c_4fcb_8af4_76147b4f8e13_insecure_sysadmin=54.199.232.88"}

	var cases = []struct {
		nodeList  []string
		swarmPort string
		expect    string
	}{
		{list1, "3376", "54.238.158.10:3376"},
	}

	for _, c := range cases {
		//call
		got := buildSwarmEndpoints(c.nodeList, c.swarmPort)
		//assert
		assert.Equal(t, c.expect, got)
	}
}
