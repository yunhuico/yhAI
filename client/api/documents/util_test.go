package documents

import (
	"net/http"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/emicklei/go-restful"
)

//测试用例中URL不需要能够真实访问到
func TestQueryIntParam(t *testing.T) {
	var cases = []struct {
		url    string
		param  string
		def    int
		expect int
	}{
		{"https://www.baidu.com/s?n=1", "n", 0, 1},
		{"https://www.baidu.com/s?wd=go%20test&rsv_spt=1&rsv_iqid=0xfd0cbaae00090dd7&issp=1", "rsv_spt", 0, 1},
		{"https://www.baidu.com/s?wd=go%20test&rsv_spt=1&rsv_iqid=0xfd0cbaae00090dd7&issp=1", "issp", 0, 1},
		{"http://192.168.10.101:8080/v1/cluster?number=3", "number", 0, 3},
		{"http://192.168.10.101:8080/v1/cluster?skip=10&count=true&limit=20", "skip", 0, 10},
		{"http://192.168.10.101:8080/v1/cluster?skip=10&count=true&limit=20", "limit", 0, 20},
		{"https://www.baidu.com/s?n=1", "number", 0, 0},
		{"https://www.baidu.com/s?n=1", "number", 5, 5},
	}

	for _, c := range cases {
		//prepare
		req := &restful.Request{}
		req.Request, _ = http.NewRequest("GET", c.url, nil)
		//call
		got := queryIntParam(req, c.param, c.def)
		//assert
		assert.Equal(t, got, c.expect)
	}
}

//测试用例中URL不需要能够真实访问到
func TestQueryBoolParam(t *testing.T) {
	var cases = []struct {
		url      string
		param    string
		def      bool
		expect   bool
		errOccur bool
	}{
		{"http://192.168.10.101:8080/v1/cluster?count=true", "count", false, true, false},
		{"http://192.168.10.101:8080/v1/cluster?count=false", "count", false, false, false},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=true", "skip_group", false, true, false},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=false", "skip_group", false, false, false},

		//0,1 shoud be recongnized
		{"http://192.168.10.101:8080/v1/cluster?count=0", "count", false, false, false},
		{"http://192.168.10.101:8080/v1/cluster?count=1", "count", false, true, false},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=0", "skip_group", false, false, false},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=1", "skip_group", false, true, false},

		//error should occur
		{"http://192.168.10.101:8080/v1/cluster?count=ok", "count", false, false, true},
		{"http://192.168.10.101:8080/v1/cluster?count=ok", "count", true, true, true},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=no", "skip_group", false, false, true},
		{"http://192.168.10.101:8080/v1/appsets?count=true&skip_group=no", "skip_group", true, true, true},
	}

	for _, c := range cases {
		//prepare
		req := &restful.Request{}
		req.Request, _ = http.NewRequest("GET", c.url, nil)
		//call
		got, err := queryBoolParam(req, c.param, c.def)
		if err != nil {
			assert.Equal(t, c.errOccur, true)
			continue
		}
		//assert
		assert.Equal(t, c.expect, got)
	}
}
