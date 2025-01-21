package documents

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"

	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	// commonService "linkernetworks.com/linker_common_lib/services"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/client/services"
	"strconv"
)

func (p Resource) NetworkWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/network")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of network")
	// number := ws.QueryParameter("number", "Change the nubmer of node for a network")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.CreateNetworkHandler).
		Doc("create a network").
		Operation("CreateNetworkHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID).To(p.DeleteNetworkHandler).
		Doc("delete a network").
		Operation("DeleteNetworkHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/").To(p.CleanNetworksHandler).
		Doc("Clean all network items in cluster").
		Operation("CleanNetworksHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("cluster_id", "The ID of the cluster")))

	ws.Route(ws.DELETE("/" + "ovs").To(p.CleanOvsNetworkHandler).
		Doc("Clean all ovs network items in db").
		Operation("CleanOvsNetworkHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("host_name", "The hostname")).
		Param(ws.QueryParameter("cluster_id", "The ID of the cluster")))

	ws.Route(ws.POST("/" + "ovs").To(p.GetOvsNetworkHandler).
		Doc("get all ovs network items in db").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("cluster_id", "The ID of the cluster")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.ListNetworksHandler).
		Doc("Returns all network items in cluster").
		Operation("ListNetworksHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("cluster_id", "The ID of the cluster")))

	ws.Route(ws.GET("/validate").To(p.NetworkNameCheckHandler).
		Doc("Check network name").
		Operation("NetworkNameCheckHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("username", "name of user")).
		Param(ws.QueryParameter("networkname", "Name of network")))

	ws.Route(ws.GET("/" + paramID).To(p.GetNetworkHandler).
		Doc("Return a network details").
		Operation("GetNetworkHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	return ws

}

func (p *Resource) GetOvsNetworkHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("GetOvsNetworkHandler is called!")
	cluster_id := req.QueryParameter("cluster_id")
	request := entity.HostNames{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	if err != nil {
		logrus.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	ovsnet, errcode, err := services.GetNetworkService().GetOvsNetwork(cluster_id, request, "x_auth_token")
	if err != nil {
		response.WriteStatusError(errcode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: ovsnet}
	resp.WriteEntity(res)
	return

}

func (p *Resource) CleanOvsNetworkHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("CleanOvsNetworkHandler is called!")
	cluster_id := req.QueryParameter("cluster_id")
	host_name := req.QueryParameter("host_name")
	errorCode, err := services.GetNetworkService().CleanOvs(cluster_id, host_name, "x_auth_token")
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) NetworkNameCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("NetworkNameCheckHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	networkName := req.QueryParameter("networkname")
	userName := req.QueryParameter("username")
	errorCode, err := services.GetNetworkService().CheckNetworkName(userName, networkName, "x_auth_token")
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) CreateNetworkHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("CreateNetworkHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	// Stub an repairpolicy to be populated from the body
	request := entity.ClusterNetwork{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)

	logrus.Infof("Request is %v", request)
	logrus.Infof("CulsterId is %v", request.ClusterId)
	logrus.Infof("Network is %v", request.Network)

	if err != nil {
		logrus.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	network, code, err := services.GetNetworkService().CreateNetwork(request, "x_auth_token")
	if network != nil {
		go common.SendCreateLog(err, "create_network", "network", network.Network.Name)
	}
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.Response{Success: true, Data: network}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteNetworkHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DeleteNetworkHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)

	network, code, err := services.GetNetworkService().DeleteById(objectId, "x_auth_token")
	go common.SendCreateLog(err, "delete_network", "network", network.Network.Name)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) GetNetworkHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("GetNetworkHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)
	network, code, err := services.GetNetworkService().QueryById(objectId, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("network is %v", network)

	res := response.QueryStruct{Success: true, Data: network}
	resp.WriteEntity(res)
	return
}

func (p *Resource) CleanNetworksHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("CleanNetworksHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }
	var cluster_id string = req.QueryParameter("cluster_id")

	code, err := services.GetNetworkService().CleanCluster(cluster_id, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return

}

func (p *Resource) ListNetworksHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ListNetworksHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := commonService.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }
	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var cluster_id string = req.QueryParameter("cluster_id")
	var sort string = req.QueryParameter("sort")

	total, networks, code, err := services.GetNetworkService().QueryAllByClusterId(cluster_id, skip, limit, sort, "x_auth_token")

	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: networks}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}
