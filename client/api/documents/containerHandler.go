package documents

import (
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"strconv"
)

func (p Resource) ContainerWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/containers")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(p.ListContainerByHostHandler).
		Doc("Get containers by host").
		Operation("ListContainerByHostHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("host_ip", "The ip of the host you want to get containers from")))

	ws.Route(ws.PUT("/{taskId}/redeploy").To(p.ContainerRedeployHandler).
		Doc("Redeploy a container").
		Operation("ContainerRedeployHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter("taskId", "The Id of the task you want to redeploy")))

	ws.Route(ws.PUT("/{taskId}/kill").To(p.ContainerKillHandler).
		Doc("Redeploy a container").
		Operation("ContainerKillHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter("taskId", "The Id of the task you want to redeploy")))

	return ws
}

func (p *Resource) ListContainerByHostHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ListContainerByHostHandler is called...")

	hostIP := req.QueryParameter("host_ip")
	total, tasks, errorCode, err := services.GetContainerService().
		GetContainersByHost(hostIP)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: tasks}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}

	resp.WriteEntity(res)
	return
}

func (p *Resource) ContainerRedeployHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ContainerRedeployHandler is called...")

	taskId := req.PathParameter("taskId")
	errorCode, err := services.GetContainerService().
		Kill(taskId, false)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}

func (p *Resource) ContainerKillHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ContainerKillHandler is called...")

	taskId := req.PathParameter("taskId")
	errorCode, err := services.GetContainerService().
		Kill(taskId, true)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}
