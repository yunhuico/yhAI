package usermgmt

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/user/services"
)

var TENANTCOLLECTION = "tenant"

func (p Resource) TenantService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/tenant")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of tenant")
	paramID := "{" + ParamID + "}"

	// ws.Route(ws.POST("/").To(p.TenantCreateHandler).
	// 	Doc("Create a new Tenant").
	// 	Operation("TenantCreateHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
	// 	Reads(""))

	ws.Route(ws.GET("/").To(p.TenantListHandler).
		Doc("Return all exist tenant items").
		Operation("TenantListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		// Param(ws.QueryParameter("fields", "Comma separated list of field names")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")))

	ws.Route(ws.GET("/" + paramID).To(p.TenantDetailHandler).
		Doc("Return a tenant by its storage identifier").
		Operation("TenantDetailHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.GET("/tenantId/").To(p.TenantIdHandler).
		Doc("Return a tenantId by userId").
		Operation("TenantIdHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("userId", "Storage identifier of tenant")))

	return ws
}

// TenantListHandler parses the http request and return a tenant.
// Usage :
//		GET /v1/tenant
//		GET /v1/tenant/{ParamID}
// Params :
//		ParamID : Storage identifier of the tenant
// If successful,response code will be set to 201.
func (p *Resource) TenantListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TenantListHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	limitnum := queryIntParam(req, "limit", 10)
	skipnum := queryIntParam(req, "skip", 0)
	sort := req.QueryParameter("sort")

	ret, count, errorCode, err := services.GetTenantService().TenantList(token, limitnum, skipnum, sort)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successList(ret, limitnum, count, req, resp)
}

func (p *Resource) TenantDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TenantDetailHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("tenant id should not be null for tenant detail operation")
		response.WriteStatusError(services.TENANT_ERROR_GET, errors.New("tenant id should not be null for get tenant operation"), resp)
		return
	}

	ret, errorCode, err := services.GetTenantService().TenantDetail(token, id)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(ret, resp)
}

func (p *Resource) TenantIdHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("TenantIdHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	userId := req.QueryParameter("userId")

	ret, errorCode, err := services.GetTenantService().GetTenantId(token, userId)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(ret, resp)
}

///////////////////reserved code/////////////////////////
// TenantCreateHandler parses the http request and create a new tenant.
// Usage :
//		POST /v1/tenant
// If successful,response code will be set to 201.
// func (p *Resource) TenantCreateHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infof("TenantCreateHandler is called!")
// 	token := req.HeaderParameter("X-Auth-Token")
// 	err, code := p.TokenValidate(token)
// 	if err != nil {
// 		response.WriteStatusError(code, err, resp)
// 		return
// 	}

// 	if authorized := p.Authorize("create_tenant", token, req); !authorized {
// 		logrus.Errorln("required opertion is not allowed!")
// 		response.WriteStatusError("E12004",
// 			errors.New("required opertion is not authorized!"), resp)
// 		return
// 	}

// 	document := bson.M{}
// 	decoder := json.NewDecoder(req.Request.Body)
// 	err = decoder.Decode(&document)
// 	if err != nil {
// 		logrus.Errorf("decode tenant err is %v", err)
// 		response.WriteStatusError("E10008", err, resp)
// 		return
// 	}

// 	name, desc, paraErr := tenantCreateParamCheck(document)
// 	if paraErr != nil {
// 		response.WriteStatusError("E12002", paraErr, resp)
// 		return
// 	}

// 	if len(name) == 0 {
// 		logrus.Errorln("parameter can not be null!")
// 		response.WriteStatusError("E12002",
// 			errors.New("invalid parameter! tenant name should not be null!"), resp)
// 		return
// 	}

// 	id, err := p.CreateAndInsertTenant(name, desc)
// 	if err != nil {
// 		logrus.Errorln("create tenant error %v", err)
// 		response.WriteStatusError("E10008", err, resp)
// 		return
// 	}

// 	p.successUpdate(id, true, req, resp)
// }

// func tenantCreateParamCheck(doc interface{}) (name string, desc string, paraErr error) {
// 	var document interface{}
// 	document, paraErr = mejson.Marshal(doc)
// 	if paraErr != nil {
// 		logrus.Errorf("marshal tenant err is %v", paraErr)
// 		return
// 	}

// 	docJson := document.(map[string]interface{})
// 	nameDoc := docJson["tenantname"]
// 	if nameDoc == nil {
// 		logrus.Errorln("invalid parameter ! tenantname can not be null")
// 		paraErr = errors.New("invalid parameter!")
// 		return
// 	} else {
// 		name = nameDoc.(string)
// 	}

// 	descDoc := docJson["description"]
// 	if descDoc != nil {
// 		desc = descDoc.(string)
// 	}

// 	return
// }
///////////////////reserved code/////////////////////////
