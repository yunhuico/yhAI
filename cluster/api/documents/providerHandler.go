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

func (p Resource) ProviderWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/provider")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of provider")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.ProviderCreateHandler).
		Doc("Store a provider").
		Operation("ProviderCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/validate").To(p.ProviderNameCheckHandler).
		Doc("Check if provider name already exists").
		Operation("ProviderNameCheckHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("provider_name", "Name of provider")))

	ws.Route(ws.GET("/").To(p.ProviderListHandler).
		Doc("Returns all provider items").
		Operation("ProviderListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("type", "IaaS provider type: amazonec2, google")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_id", "The owner ID of the provider")))

	ws.Route(ws.GET("/" + paramID).To(p.ProviderGetHandler).
		Doc("Return a provider").
		Operation("ProviderGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/" + paramID).To(p.ProviderDeleteHandler).
		Doc("Detele a provider by its storage identifier").
		Operation("ProviderDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.PUT("/" + paramID).To(p.ProviderUpdateHandler).
		Doc("Updata a exist Provider by its storage identifier").
		Operation("ProviderUpdateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "Provider update request body in json format").DataType("string")))
	return ws

}

func (p *Resource) ProviderCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderCreateHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	// Stub an acluster to be populated from the body
	provider := entity.IaaSProvider{}

	err := json.NewDecoder(req.Request.Body).Decode(&provider)
	if err != nil {
		logrus.Errorf("convert body to provider failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newProvider, code, err := services.GetProviderService().Create(provider, x_auth_token)
	logrus.Infof("start to create log")
	createLog(err, "create_provider", "provider", newProvider.Name, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newProvider}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ProviderNameCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderNameCheckHandler is called...")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	provider_name := req.QueryParameter("provider_name")
	conflict, errorCode, err := services.GetProviderService().CheckProviderName(provider_name, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	res := response.QueryStruct{Success: conflict}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ProviderGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	// cluster, code, err := services.GetClusterService().QueryById(objectId, x_auth_token)
	provider, errorCode, err := services.GetProviderService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	logrus.Debugf("provider is %v", provider)

	res := response.QueryStruct{Success: true, Data: provider}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ProviderListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	limitnum := queryIntParam(req, "limit", 0)
	skipnum := queryIntParam(req, "skip", 0)
	providerType := req.QueryParameter("type")

	var sort string = req.QueryParameter("sort")
	userId := req.QueryParameter("user_id")

	total, providers, errorCode, err := services.GetProviderService().QueryProvider(providerType, userId, skipnum, limitnum, sort, x_auth_token)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	logrus.Debugf("providers is %v", providers)

	providerinfo := services.GetProviderService().GetProviderInfo(providers, x_auth_token)

	res := response.QueryStruct{Success: true, Data: providerinfo}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ProviderDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderDeleteHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	provider, errorCode, err := services.GetProviderService().DeleteById(objectId, x_auth_token)
	logrus.Infof("start to create delete provider log")
	createLog(err, "delete_provider", "provider", provider.Name, x_auth_token)
	if err != nil {
		logrus.Errorln("terminate hosts error is %v", err)
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	// Write success response
	response.WriteSuccess(resp)
	return
}

func (p *Resource) ProviderUpdateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ProviderUpdateHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")
	objectId := req.PathParameter(ParamID)
	if len(objectId) <= 0 {
		logrus.Warnln("provider id should not be null for update operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("provider id should not be null for update operation"), resp)
		return
	}

	newProvider := entity.IaaSProvider{}

	// Populate the provider data
	err := json.NewDecoder(req.Request.Body).Decode(&newProvider)
	if err != nil {
		logrus.Errorf("convert body to Provider failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	created, id, errorCode, err := services.GetProviderService().ProviderUpdate(token, newProvider, objectId)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successUpdate(id, created, req, resp)
	return
}
