package usermgmt

import (
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/user/services"
)

var ROLECOLLECTION = "role"

func (p Resource) RoleService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/role")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(p.RoleListHandler).
		Doc("Return exist roles").
		Operation("RoleListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")))

	return ws
}

// RoleListHandler parses the http request and return the exist roles.
// Usage :
//		GET /v1/role
// If successful,response code will be set to 201.
func (p *Resource) RoleListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("RoleListHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")

	ret, count, errorCode, err := services.GetRoleService().RoleList(token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successList(ret, 0, count, req, resp)

}
