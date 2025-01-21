package documents

import (
	"errors"
	"encoding/json"
	"strconv"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/cluster/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

const (
	ParamHostID string = "host_id"
)

func (p Resource) ClusterWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/cluster")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of cluster")
	paramID := "{" + ParamID + "}"
	paramHostID := "{" + ParamHostID + "}"

	ws.Route(ws.POST("/").To(p.ClusterCreateHandler).
		Doc("Store a cluster").
		Operation("ClusterCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.ClustersListHandler).
		Doc("Returns all cluster items").
		Operation("ClustersListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("name", "The name of cluster wanted to query")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_id", "The owner ID of the cluster")).
		Param(ws.QueryParameter("username", "The owner of the cluster")).
		Param(ws.QueryParameter("status", "DEPLOYED,RUNNING,FAILED,TERMINATED,unterminated. Query all clusters by default if not provided")))

	ws.Route(ws.GET("/" + paramID).To(p.ClusterGetHandler).
		Doc("Return a cluster").
		Operation("ClusterGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))
		
	ws.Route(ws.GET("/" + paramID + "/components").To(p.GetClusterComponentsHandler).
		Doc("Return a cluster's components").
		Operation("GetClusterComponentsHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/" + paramID).To(p.ClusterDeleteHandler).
		Doc("Detele a Cluster by its storage identifier").
		Operation("ClusterDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.POST("/" + paramID + "/email").To(p.EmailSendHandler).
		Doc("Send cluster owner an email of endpoint.").
		Operation("EmailSendHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))
		
	ws.Route(ws.POST("/alertEmail").To(p.AlertEmailSendHandler).
		Doc("send alert email to user").
		Operation("AlertEmailSendHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/" + paramID + "/hosts").To(p.HostsAddHandler).
		Doc("Add hosts for a cluster").
		Operation("HostsAddHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID + "/hosts").To(p.HostsDeleteHandler).
		Doc("Terminate hosts of a cluster").
		Operation("HostsDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")).
		Param(id))

	ws.Route(ws.GET("/" + paramID + "/hosts").To(p.HostsListHandler).
		Doc("List hosts of a cluster").
		Operation("HostsListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("status", "DEPLOYED,RUNNING,FAILED,TERMINATED,unterminated. Query all hosts by default if not provided")).
		Param(ws.QueryParameter("tag", "Host tag. Query all hosts by default if not provided")).
		Param(id))

	ws.Route(ws.GET("/" + paramID + "/hosts/" + paramHostID).To(p.HostGetHandler).
		Doc("List detail of a host").
		Operation("HostGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter(ParamHostID, "Storage identifier of host")).
		Param(id))

	ws.Route(ws.POST("/notify").To(p.ClusterNotifyHandler).
		Doc("Notify cluster").
		Operation("ClusterNotifyHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/hosts/notify").To(p.HostsNotifyHandler).
		Doc("Notify hosts").
		Operation("HostsNotifyHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/" + paramID + "/pubkey").To(p.PubkeyAddHandler).
		Doc("Add pubkey for a cluster").
		Operation("PubkeyAddHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID + "/pubkey").To(p.PubkeyDeleteHandler).
		Doc("Delete pubkey for a cluster").
		Operation("PubkeyDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.POST("/" + paramID + "/registry").To(p.RegistryAddHandler).
		Doc("Add registry for a cluster").
		Operation("RegistryAddHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID + "/registry").To(p.RegistryDeleteHandler).
		Doc("Delete registry for a cluster").
		Operation("RegistryDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.POST("/" + paramID + "/setting").To(p.SettingHandler).
		Doc("set cmi for a cluster").
		Operation("SettingHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))
		
	ws.Route(ws.GET("/check").To(p.ClusterCheckHandler).
		Doc("Check cluster name").
		Operation("ClusterCheckHandler"))
		

	//pre-check if cluster name matches with regex, and if it is conflict
	ws.Route(ws.GET("/validate").To(p.ClusterNameCheckHandler).
		Doc("Check cluster name").
		Operation("ClusterNameCheckHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("userid", "Storage identifier of user")).
		Param(ws.QueryParameter("clustername", "Name of cluster")))

	return ws

}

type project struct {
	Cmi string `json:"cmi"`
}

func (p *Resource) ClusterCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("just check clustermgmt")
	response.WriteSuccess(resp)
	return
}

func (p *Resource) GetClusterComponentsHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("GetClusterComponentsHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	components, code, err := services.GetClusterService().GetComponents(clusterId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("components is %v", components)

	res := response.QueryStruct{Success: true, Data: components}
	resp.WriteEntity(res)
	return
	
}

func (p *Resource) SettingHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("SettingHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	body := project{}
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	if err != nil {
		logrus.Errorf("convert body to RequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	cmi := body.Cmi
	
	if cmi == "" {
		logrus.Errorf("cmi value can not be null")
		err := errors.New("cmi value can not be null")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	errorCode, errt := services.GetClusterService().SettingProject(clusterId, cmi, x_auth_token)
	if errt != nil {
		logrus.Errorln("add pubkey error is %v", errt)
		response.WriteStatusError(errorCode, errt, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
	
}

func (p *Resource) RegistryAddHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("RegistryAddHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	body := RequestBody{}
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	if err != nil {
		logrus.Errorf("convert body to RequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	errorCode, errt := services.GetClusterService().AddRegistry(clusterId, body.Ids, x_auth_token)
	if errt != nil {
		logrus.Errorln("add pubkey error is %v", errt)
		response.WriteStatusError(errorCode, errt, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return

}

func (p *Resource) RegistryDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("RegistryDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	body := RequestBody{}
	clusterId := req.PathParameter(ParamID)
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	logrus.Infof("body is %v", body)
	if err != nil {
		logrus.Errorf("convert body to RequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	errorCode, err := services.GetClusterService().DeleteRegistry(clusterId, body.Ids, x_auth_token)
	if err != nil {
		logrus.Errorln("add pubkey error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}

func (p *Resource) PubkeyDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("PubkeyDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	body := RequestBody{}
	clusterId := req.PathParameter(ParamID)
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	logrus.Infof("body is %v", body)
	if err != nil {
		logrus.Errorf("convert body to RequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	errorCode, err := services.GetClusterService().DeletePubkeys(clusterId, body.Ids, x_auth_token)
	if err != nil {
		logrus.Errorln("add pubkey error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}

func (p *Resource) PubkeyAddHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("PubkeyAddHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	body := RequestBody{}
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	if err != nil {
		logrus.Errorf("convert body to RequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	errorCode, errt := services.GetClusterService().AddPubkeys(clusterId, body.Ids, x_auth_token)
	if errt != nil {
		logrus.Errorln("add pubkey error is %v", errt)
		response.WriteStatusError(errorCode, errt, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return

}

type RequestBody struct {
	Ids []string `json:"ids"`
}

func (p *Resource) HostsNotifyHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("HostsNotifyHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	hostsNotify := entity.NotifyHost{}

	err := json.NewDecoder(req.Request.Body).Decode(&hostsNotify)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	code, errn := services.GetClusterService().NotifyHosts(hostsNotify, x_auth_token)
	if errn != nil {
		response.WriteStatusError(code, errn, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return

}

func (p *Resource) ClusterNotifyHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("ClusterNotifyHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	logrus.Infoln("callback token is %v", x_auth_token)

	clusterNotify := entity.NotifyCluster{}
	err := json.NewDecoder(req.Request.Body).Decode(&clusterNotify)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	_, clusters, _, err := services.GetClusterService().QueryCluster(clusterNotify.ClusterName, "", clusterNotify.UserName, "unterminated", 0, 0, "", x_auth_token)
	if err != nil {
		logrus.Warnf("can not found cluster with name %s, error is %v", clusterNotify.ClusterName, err)
	}

	var clusterId string
	if len(clusters) <= 0 {
		logrus.Warnf("no cluster with name %s", clusterNotify.ClusterName)
	} else {
		clusterId = clusters[0].ObjectId.Hex()
	}

	code, errn := services.GetClusterService().NotifyCluster(clusterNotify, x_auth_token)
	if errn != nil {
		_, erru := services.GetLogService().UpdateLogStatus(clusterNotify.LogId, services.LOG_OPERATE_STATUS_FAIL, x_auth_token)
		if erru != nil {
			logrus.Errorf("change log status err is %v", erru)
		}
		response.WriteStatusError(code, errn, resp)
		return
	}

	//save operat log
	//logerr name does not affect the program to err
	_, _, logerr := services.GetLogService().CreateByClusterNotifyLog(clusterNotify, clusterId, x_auth_token)
	if logerr != nil {
		logrus.Errorf("save log to db error is [%v]", logerr)
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

//check username and clustername
func (p *Resource) ClusterNameCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("ClusterNameCheckHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	userId := req.QueryParameter("userid")
	clusterName := req.QueryParameter("clustername")
	errorCode, err := services.GetClusterService().CheckClusterName(userId, clusterName, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

//Send cluster owner an email of endpoint
func (p *Resource) EmailSendHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("EmailSendHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	clusterId := req.PathParameter(ParamID)
	errorCode, err := services.GetEmailService().SendClusterDeployedEmail(clusterId, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource)AlertEmailSendHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("AlertEmailSendHandler is called!")
	sendEmail := entity.SendHostAlertReq{}
	err := json.NewDecoder(req.Request.Body).Decode(&sendEmail)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	
	errorCode, err := services.GetEmailService().SendAlertEmail(sendEmail)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
	
}

func (p *Resource) ClusterDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("ClusterDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	objectId := req.PathParameter(ParamID)

	// start to change log status
	newlog, _, logerr := services.GetLogService().CreateByLogParam("cluster", objectId, "", services.LOG_OPERATE_TYPE_DELETE_CLUSTER, services.LOG_OPERATE_STATUS_START, x_auth_token)
	if logerr != nil {
		logrus.Errorf("save log to db error is [%v]", logerr)
	}

	code, err := services.GetClusterService().DeleteById(objectId, newlog.ObjectId.Hex(), x_auth_token)
	if err != nil {
		// delete cluster is err change log status failed
		_, erru := services.GetLogService().UpdateLogStatus(newlog.ObjectId.Hex(), services.LOG_OPERATE_STATUS_FAIL, x_auth_token)
		if erru != nil {
			logrus.Errorf("change log status err is %v", erru)
		}

		response.WriteStatusError(code, err, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) ClusterCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClusterCreateHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	// Stub an acluster to be populated from the body
	clusterRequest := entity.CreateRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&clusterRequest)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	// start to save operation log
	newlog, _, logerr := services.GetLogService().CreateByLogParam("cluster", "", clusterRequest.Name, services.LOG_OPERATE_TYPE_CREATE_CLUSTER, services.LOG_OPERATE_STATUS_START, x_auth_token)
	var logId string
	if logerr != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterRequest.Name}).Errorf("save log to db error is [%v]", logerr)
		logId = bson.NewObjectId().Hex()
	} else {
		logId = newlog.ObjectId.Hex()
	}

	newCluster, code, err := services.GetClusterService().Create(clusterRequest, logId, x_auth_token)
	if err != nil {
		// create newcluster is err change log status failed
		_, erru := services.GetLogService().UpdateLogStatus(logId, services.LOG_OPERATE_STATUS_FAIL, x_auth_token)
		if erru != nil {
			logrus.WithFields(logrus.Fields{"clustername": clusterRequest.Name}).Errorf("change log status err is %v", erru)
		}
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newCluster}
	resp.WriteEntity(res)
	return

}

func (p *Resource) ClustersListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClustersListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var name string = req.QueryParameter("name")
	var user_id string = req.QueryParameter("user_id")
	var status string = req.QueryParameter("status")
	var sort string = req.QueryParameter("sort")
	var username string = req.QueryParameter("username")

	total, clusters, code, err := services.GetClusterService().QueryCluster(name, user_id, username, status, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: clusters}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}

func (p *Resource) ClusterGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ClusterGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	cluster, code, err := services.GetClusterService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("cluster is %v", cluster)

	res := response.QueryStruct{Success: true, Data: cluster}
	resp.WriteEntity(res)
	return

}

func (p *Resource) HostsAddHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsAddHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	addrequest := entity.AddRequest{}
	err := json.NewDecoder(req.Request.Body).Decode(&addrequest)
	if err != nil {
		logrus.Errorf("convert body to AddRequest failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	// start to save operation log
	newlog, _, logerr := services.GetLogService().CreateByLogParam("host", addrequest.ClusterId, "", services.LOG_OPERATE_TYPE_ADD_HOSTS, services.LOG_OPERATE_STATUS_START, x_auth_token)
	if logerr != nil {
		logrus.Errorf("save log to db error is [%v]", logerr)
	}

	cluster, errorCode, err := services.GetClusterService().AddHosts(addrequest, newlog.ObjectId.Hex(), x_auth_token)
	if err != nil {
		// add host is err start to change log status to failed
		_, erru := services.GetLogService().UpdateLogStatus(newlog.ObjectId.Hex(), services.LOG_OPERATE_STATUS_FAIL, x_auth_token)
		if erru != nil {
			logrus.Errorf("change log status err is %v", erru)
		}
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: cluster}
	resp.WriteEntity(res)
	return
}

type TerminateHostsRequestBody struct {
	HostIds []string `json:"host_ids"`
}

//terminate specified hosts of a cluster
// Request
// URL:
// 	PUT /v1/cluster/<CLUSTER_ID>/hosts
// Header:
// 	X-Auth-Token
// Except Body:
//{
//    "host_ids":["568e23655d5c3d173019f1ba","568e2be45d5c3d173019f1bb","568e2bfd5d5c3d173019f1bc","568e2c335d5c3d173019f1bd"]
//}
//
// Response:
//{
//  "success": true
//}
//
func (p *Resource) HostsDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	body := TerminateHostsRequestBody{}
	err := json.NewDecoder(req.Request.Body).Decode(&body)
	if err != nil {
		logrus.Errorf("convert body to TerminateHostsRequestBody failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	// start to save operation log
	newlog, _, logerr := services.GetLogService().CreateByLogParam("host", clusterId, "", services.LOG_OPERATE_TYPE_DELETE_HOSTS, services.LOG_OPERATE_STATUS_START, x_auth_token)
	if logerr != nil {
		logrus.Errorf("save log to db error is [%v]", logerr)
	}

	errorCode, errt := services.GetClusterService().TerminateHosts(clusterId, body.HostIds, newlog.ObjectId.Hex(), x_auth_token)
	if errt != nil {
		logrus.Errorln("terminate hosts error is %v", errt)
		// terminated hosts err start to change log status failed
		_, erru := services.GetLogService().UpdateLogStatus(newlog.ObjectId.Hex(), services.LOG_OPERATE_STATUS_FAIL, x_auth_token)
		if erru != nil {
			logrus.Errorf("change log status err is %v", erru)
		}

		response.WriteStatusError(errorCode, errt, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) HostsListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostsListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	clusterId := req.PathParameter(ParamID)
	var err error
	var skip, limit int64
	if param_skip := req.QueryParameter("skip"); len(param_skip) > 0 {
		skip, err = strconv.ParseInt(param_skip, 10, 0)
		if err != nil {
			response.WriteStatusError("E12002", err, resp)
			return
		}
	}
	if param_limit := req.QueryParameter("limit"); len(param_limit) > 0 {
		limit, err = strconv.ParseInt(req.QueryParameter("limit"), 10, 0)
		if err != nil {
			response.WriteStatusError("E12002", err, resp)
			return
		}
	}

	var status string = req.QueryParameter("status")
	total, hosts, errorCode, err := services.GetHostService().QueryHosts(clusterId, int(skip), int(limit), status, x_auth_token)
	if err != nil {
		logrus.Errorln("list hosts error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	tag := req.QueryParameter("tag")
	hostnames := services.GetHostNameByTag(hosts, tag)
	logrus.Debugf("get hostname by tag %s, hostnames %v", tag, hostnames)

	hostinfo := services.GetHostAdditionalInfos(hosts, hostnames, x_auth_token)

	res := response.QueryStruct{Success: true, Data: hostinfo}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

func (p *Resource) HostGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("HostGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	hostId := req.PathParameter(ParamHostID)

	host, code, err := services.GetHostService().QueryById(hostId, x_auth_token)
	if err != nil {
		logrus.Errorln("get host error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: host}
	resp.WriteEntity(res)
	return
}
