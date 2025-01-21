package documents

import (
	"encoding/json"
	"strconv"

	"github.com/Sirupsen/logrus"
	
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/cluster/services"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

func (p Resource) LogtWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/logs")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/").To(p.LogListHandler).
		Doc("List all logs").
		Operation("LogListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_name", "the username of the log")).
		Param(ws.QueryParameter("queryType", "the querytype of the log")).
		Param(ws.QueryParameter("user_id", "the userid of the log")).
		Param(ws.QueryParameter("cluster_id", "The cluster_name of the log")))

	ws.Route(ws.POST("/create").To(p.LogCreateHandler).
		Doc("create log for client").
		Operation("LogCreateHandler").
		Param(ws.BodyParameter("body", "").DataType("string")))

	return ws
}

func (p *Resource) LogCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("LogCreateHandler is called!")
	createRequest := entity.LogMessage{}
	err := json.NewDecoder(req.Request.Body).Decode(&createRequest)
	if err != nil {
		logrus.Errorf("convert body to cluster failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	newLog, code, err := services.GetLogService().Create(&createRequest, "")
	if err != nil {
		logrus.Errorf("create log failed, error is %v", err)
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: newLog}
	resp.WriteEntity(res)
	return
}

func (p *Resource) LogListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("LogListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	skip := queryIntParam(req, "skip", 0)
	limit := queryIntParam(req, "limit", 0)
	sort := req.QueryParameter("sort")
	clusterId := req.QueryParameter("cluster_id")
	usernName := req.QueryParameter("user_name")
	userid := req.QueryParameter("user_id")
	querytype := req.QueryParameter("queryType")

	//query logs
	total, logs, errorCode, err := services.GetLogService().QueryLogs(clusterId, usernName, querytype, userid, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: logs}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}

	resp.WriteEntity(res)
	return
}
