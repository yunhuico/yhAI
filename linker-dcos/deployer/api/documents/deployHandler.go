package documents

import (
	"encoding/json"
	//	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/deployer/services"
)

func (p Resource) DeployWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/deploy")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// id := ws.PathParameter(ParamID, "Storage identifier of cluster")
	// number := ws.QueryParameter("number", "Change the nubmer of node for a cluster")
	// paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.CreateClusterHandler).
		Doc("create a cluster").
		Operation("CreateClusterHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/").To(p.DeleteClusterHandler).
		Doc("delete a cluster").
		Operation("DeleteClusterHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/nodes").To(p.AddNodesHandler).
		Doc("add nodes").
		Operation("AddNodesHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/nodes").To(p.DeleteNodesHandler).
		Doc("delete nodes").
		Operation("DeleteNodesHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/addpubkeys").To(p.AddPubkeysHandler).
		Doc("Add a pubkey").
		Operation("AddPubkeysHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/deletepubkeys").To(p.DeletePubkeyHandler).
		Doc("delete a cluster").
		Operation("DeletePubkeyHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/addregistry").To(p.AddRegistryHandler).
		Doc("Add a registry").
		Operation("AddRegistryHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/deleteregistry").To(p.DeleteregistryHandler).
		Doc("delete a registry").
		Operation("DeleteregistryHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.POST("/components/healthcheck").To(p.ComponentCheckHandler).
		Doc("check component").
		Operation("ComponentCheckHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/nodes/healthcheck").To(p.NodeCheckHandler).
		Doc("check nodes").
		Operation("NodeCheckHandler").
		Param(ws.QueryParameter("username", "cluster's username").DataType("string")).
		Param(ws.QueryParameter("clustername", "cluster's name").DataType("string")))

	return ws

}

func (p *Resource) ComponentCheckHandler(req *restful.Request, resp *restful.Response) {
	request := entity.Components{}
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("ComponentCheckHandler is called!")
	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	
	componentInfo, errCode, err := services.GetDeployService().ComponentsCheck(request)
	if err != nil {
		response.WriteStatusError(errCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: componentInfo}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteregistryHandler(req *restful.Request, resp *restful.Response) {
	request := entity.DeleteRegistryRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("DeleteregistryHandler is called!")
	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	erra := services.GetDeployService().DeleteRegistry(request)
	if erra != nil {
		creLog.Errorf("delete pubkey err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, erra, resp)
		return
	}
	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) AddRegistryHandler(req *restful.Request, resp *restful.Response) {
	request := entity.AddRegistryRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("AddRegistryHandler is called!")
	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	erra := services.GetDeployService().AddRegistry(request)
	if erra != nil {
		creLog.Errorf("add pubkey err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, erra, resp)
		return
	}
	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return

}

func (p *Resource) DeletePubkeyHandler(req *restful.Request, resp *restful.Response) {
	request := entity.DeletePubkeysRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("DeletePubkeyHandler is called!")
	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	erra := services.GetDeployService().DeletePubkeys(request)
	if erra != nil {
		creLog.Errorf("delete pubkey err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, erra, resp)
		return
	}
	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) AddPubkeysHandler(req *restful.Request, resp *restful.Response) {
	request := entity.AddPubkeysRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("AddPubkeysHandler is called!")
	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	erra := services.GetDeployService().AddPubkeys(request)
	if erra != nil {
		creLog.Errorf("add pubkey err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, erra, resp)
		return
	}
	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return

}

func (p *Resource) CreateClusterHandler(req *restful.Request, resp *restful.Response) {
	// Stub an repairpolicy to be populated from the body
	request := entity.Request{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("CreateClusterHandler is called!")
	creLog.Infoln("cluster request token is %v", request.XAuthToken)

	creLog.Infof("create cluster request is %v", request)

	if err != nil {
		creLog.Errorf("convert body to request failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	servers := []entity.Server{}
	go services.GetDeployService().CreateCluster(request)

	res := response.Response{Success: true, Data: servers}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteClusterHandler(req *restful.Request, resp *restful.Response) {
	// Stub an repairpolicy to be populated from the body
	request := entity.DeleteAllRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	delLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	delLog.Infof("DeleteClusterHandler is called!")

	delLog.Infof("delete cluster request is %v", request)

	if err != nil {
		delLog.Errorf("convert body to DeleteClusterRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	go services.GetDeployService().DeleteCluster(request.UserName, request.ClusterName, request.ClusterId, request.LogId, request.XAuthToken, request.ClusterMgmtIp)

	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) AddNodesHandler(req *restful.Request, resp *restful.Response) {
	// Stub an repairpolicy to be populated from the body
	request := entity.AddNodeRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	adLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	adLog.Infof("AddNodesHandler is called!")
	adLog.Infof("add node request is %v", request)

	if err != nil {
		adLog.Errorf("convert body to AddNodesRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	go services.GetDeployService().CreateNode(request)

	var res response.Response
	servers := []entity.Server{}
	res = response.Response{Success: true, Data: servers}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DeleteNodesHandler(req *restful.Request, resp *restful.Response) {
	// Stub an repairpolicy to be populated from the body
	request := entity.DeleteRequest{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&request)
	deLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	deLog.Infof("DeleteNodesHandler is called!")
	deLog.Infof("delete node request is %v", request)

	if err != nil {
		deLog.Errorf("convert body to DeleteClusterRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	go services.GetDeployService().DeleteNode(request)
	slaves := []entity.Server{}
	res := response.Response{Success: true, Data: slaves}

	resp.WriteEntity(res)
	return
}

func (p *Resource) NodeCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("NodeCheckHandler is called!")

	username := req.QueryParameter("username")
	clustername := req.QueryParameter("clustername")
	nodes, code, err := services.GetDeployService().GetNodesCheck(username, clustername)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: nodes}
	resp.WriteEntity(res)
	return

}
