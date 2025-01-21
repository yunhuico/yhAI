package documents

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	restful "github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) HostMonitorService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/hostrules")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(p.GetHostRulesHandler).
		Doc("Query Prometheus rules of host monitor").
		Operation("UpdateHostRulesHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")))
	ws.Route(ws.PUT("/").To(p.UpdateHostRulesHandler).
		Doc("update Prometheus rules of host monitor").
		Operation("UpdateHostRulesHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))
	return ws
}

func (p *Resource) GetHostRulesHandler(req *restful.Request, resp *restful.Response) {
	hostrules, errcode, err := services.GetHostMonitorService().GetHostRules("x_auth_token")
	if err != nil {
		response.WriteStatusError(errcode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: *hostrules}
	resp.WriteEntity(res)
	return
}

func (p *Resource) UpdateHostRulesHandler(req *restful.Request, resp *restful.Response) {
	// decode request body
	request := entity.ReqPutRules{}
	if err := json.NewDecoder(req.Request.Body).Decode(&request); err != nil {
		logrus.Errorf("decode body to struct failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	hostrules, code, err := services.GetHostMonitorService().UpdateRules(request, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.Response{Success: true, Data: *hostrules}
	resp.WriteEntity(res)
	return
}
