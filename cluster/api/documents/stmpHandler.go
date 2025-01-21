package documents

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/cluster/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) SmtpWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/smtp")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of smtp")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.SmtpCreateHandler).
		Doc("Store a stmp").
		Operation("SmtpCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.SmtpListHandler).
		Doc("Returns all smtp items").
		Operation("SmtpListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("name", "The name of smtp wanted to query")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("address", "The address of the smtp")))

	ws.Route(ws.GET("/" + paramID).To(p.SmtpGetHandler).
		Doc("Return a smtp").
		Operation("SmtpGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/" + paramID).To(p.SmtpDeleteHandler).
		Doc("Detele a smtp by its storage identifier").
		Operation("SmtpDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.PUT("/" + paramID).To(p.SmtpUpdateHandler).
		Doc("Updata a exist stmp by its storage identifier").
		Operation("SmtpUpdateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "").DataType("string")))

	return ws

}

func (p *Resource) SmtpUpdateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("SmtpUpdateHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("smtp id should not be null for update operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("stmp id should not be null for update operation"), resp)
		return
	}

	newSmtp := entity.Smtp{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&newSmtp)
	if err != nil {
		logrus.Errorf("convert body to smtp failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	created, id, errorCode, err := services.GetSmtpService().SmtpUpdate(token, newSmtp, id)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successUpdate(id, created, req, resp)
}

func (p *Resource) SmtpDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("SmtpDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	objectId := req.PathParameter(ParamID)

	smtp, code, err := services.GetSmtpService().DeleteById(objectId, x_auth_token)
	createLog(err, "delete_smtp", "smtp", smtp.Name, x_auth_token)
	
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	res := response.QueryStruct{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) SmtpGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("SmtpGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	smtp, code, err := services.GetSmtpService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("stmp is %v", smtp)

	res := response.QueryStruct{Success: true, Data: smtp}
	resp.WriteEntity(res)
	return

}

func (p *Resource) SmtpListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("SmtpListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var name string = req.QueryParameter("name")
	var address string = req.QueryParameter("address")

	var sort string = req.QueryParameter("sort")

	total, stmps, code, err := services.GetSmtpService().QueryStmp(name, address, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: stmps}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}

func (p *Resource) SmtpCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("SmtpCreateHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	smtp := entity.Smtp{}

	err := json.NewDecoder(req.Request.Body).Decode(&smtp)
	if err != nil {
		logrus.Errorf("convert body to stmp failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newSmtp, code, err := services.GetSmtpService().Save(smtp, x_auth_token)
	createLog(err, "create_smtp", "smtp", newSmtp.Name, x_auth_token)
	
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newSmtp}
	resp.WriteEntity(res)
	return

}
