package documents

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/jmoiron/jsonq"
	"io/ioutil"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"strings"
)

func (p Resource) DeployMarathonService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/marathon")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// id := ws.PathParameter(ParamID, "Storage identifier of cluster")
	// number := ws.QueryParameter("number", "Change the nubmer of node for a cluster")
	// paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/notify").To(p.NotifyMarathonHandler).
		Doc("receive a notification").
		Operation("NotifyMarathonHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	return ws

}

func (p *Resource) NotifyMarathonHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("NotifyMarathonHandler is called!")

	dat, err := ioutil.ReadAll(req.Request.Body)

	if err != nil {
		logrus.Errorf("read notification body failed, error is %v", err)
		return
	}

	s := string(dat[:len(dat)])

	jsondata := map[string]interface{}{}
	result := json.NewDecoder(strings.NewReader(s))
	result.Decode(&jsondata)
	jq := jsonq.NewQuery(jsondata)
	value, _ := jq.String("eventType")
	logrus.Infof("Marathon callback starting ......................")
	logrus.Infof("Notification is %s", s)
	logrus.Infof("eventType is %s", value)

	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}
