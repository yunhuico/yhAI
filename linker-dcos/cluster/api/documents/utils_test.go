package documents

import (
	"net/http"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/emicklei/go-restful"
)

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

func TestDocumentLocation(t *testing.T) {
	var cases = []struct {
		url    string
		id     string
		expect string
	}{
		{"http://192.168.10.198:10002/v1/cluster", "569c9038e138239fa125d453", "/v1/cluster/569c9038e138239fa125d453"},
		{"http://192.168.10.198:10002/v1/cluster/", "569c9038e138239fa125d453", "/v1/cluster/569c9038e138239fa125d453"},
		{"http://192.168.10.198:10002/v1/cluster/569c9038e138239fa125d453", "569c9038e138239fa125d453", "/v1/cluster/569c9038e138239fa125d453"},
		{"http://192.168.10.198:10002/v1/cluster/569c9038e138239fa125d453/", "569c9038e138239fa125d453", "/v1/cluster/569c9038e138239fa125d453"},
		{"http://192.168.10.198:10002/v1/cluster/569c9038e138239fa125d453/hosts", "569c9038e138239fa125d453", "/v1/cluster/569c9038e138239fa125d453/hosts/569c9038e138239fa125d453"},
	}

	for _, c := range cases {
		//prepare
		req := &restful.Request{}
		req.Request, _ = http.NewRequest("GET", c.url, nil)
		//call
		got := documentLocation(req, c.id)
		//assert
		assert.Equal(t, c.expect, got)
	}

}

func TestSuccessUpdate(t *testing.T) {

}
