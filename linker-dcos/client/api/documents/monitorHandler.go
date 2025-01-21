package documents

import (
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	// "linkernetworks.com/linker_common_lib/entity"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) MonitorWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/monitors")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// ws.Route(ws.GET("/containers").To(p.MonitorGetAllContainersHandler).
	// 	Doc("Get all containers in this cluster").
	// 	Operation("MonitorGetAllContainersHandler"))

	ws.Route(ws.GET("/{groupId}/containers").To(p.MonitorGetAllContainersByGroupIdHandler).
		Doc("Get all containers with given group id").
		Operation("MonitorGetAllContainersByGroupIdHandler").
		Param(ws.PathParameter("groupId", "Group ID for this service")))

	return ws
}

// func (p *Resource) MonitorGetAllContainersHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infoln("MonitorGetAllContainersHandler is called!")

// 	// get all tasks from marathon
// 	_, tasks, errorCode, err := services.GetContainerService().
// 		GetAllContainers()
// 	if err != nil {
// 		response.WriteStatusError(errorCode, err, resp)
// 		return
// 	}

// 	res := response.Response{Success: true, Data: tasks}
// 	resp.WriteEntity(res)

// 	return
// }

func (p *Resource) MonitorGetAllContainersByGroupIdHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("MonitorGetAllContainersHandler is called!")

	groupId := req.PathParameter("groupId")
	logrus.Debugf("group id is : %s", groupId)

	// query marathon events
	_, tasks, errorCode, err := services.GetContainerService().
		GetContainersByAppSet(groupId)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: tasks}
	resp.WriteEntity(res)

	return
}
