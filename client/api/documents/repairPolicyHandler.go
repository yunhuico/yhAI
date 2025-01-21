package documents

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"strconv"
)

func (p Resource) RepairPolicyWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/repairPolices")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of the repairpolicy")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.GET("/").To(p.RepairPolicyListHandler).
		Doc("Returns all repairpolicy items").
		Operation("RepairPolicyListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")))

	ws.Route(ws.GET("/" + paramID).To(p.RepairPolicyGetHandler).
		Doc("Return an repairpolicy by its storage identifier").
		Operation("RepairPolicyListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.QueryParameter("fields", "Comma separated list of field names")))

	ws.Route(ws.GET("/operations/" + paramID).To(p.RepairPolicyAuthOperationHandler).
		Doc("Return authorized operations of an repairpolicy by its storage identifier").
		Operation("RepairPolicyAuthOperationHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.POST("/").To(p.RepairPolicyCreateHandler).
		Doc("Store an repairpolicy").
		Operation("RepairPolicyCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "RepairPolicy body in json format,for example {\"cmd\":\"...\",   \"id\":\"...\",  \"cpus\":\"...\",   \"mem\":\"...\",  \"instances\":\"...\",   \"env\":\"...\",  \"excutor\":\"...\",  \"constraints\":[],  \"container\":{\"docker\":{}}}").DataType("string")))

	ws.Route(ws.PUT("/" + paramID).To(p.RepairPolicyUpdateHandler).
		Doc("Update an repairpolicy by storage identifier").
		Operation("RepairPolicyUpdateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "RepairPolicy body in json format,for example {\"cmd\":\"...\",   \"id\":\"...\",   \"cpus\":\"...\",   \"mem\":\"...\",   \"instances\":\"...\",   \"env\":\"...\",   \"excutor\":\"...\",   \"constraints\":[],  \"container\":{\"docker\":{}}}").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID).To(p.RepairPolicyDeleteHandler).
		Doc("Delete an repairpolicy by its storage identifier").
		Operation("RepairPolicyDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	return ws
}

// RepairPolicyListHandler parses the http request and return the repairpolicy items.
// Usage :
//		GET /v1/repairPolicies
// Params :
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyListHandler(req *restful.Request,
	resp *restful.Response) {
	// logrus.Infof("RepairPolicyListHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	total, repairpolicies, code, err := services.GetRepairPolicyService().QueryAll(skip, limit, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: repairpolicies}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

// RepairPolicyGetHandler parses the http request and return the repairpolicy item by objectid.
// Usage :
//		GET /v1/repairPolicies/{ParamID}
// Params :
//		ParamID : Storage identifier of the repairpolicy
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyGetHandler(req *restful.Request,
	resp *restful.Response) {
	// logrus.Infof("RepairPolicyGetHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)

	repairpolicy, code, err := services.GetRepairPolicyService().QueryById(objectId, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("repairpolicy is %v", repairpolicy)

	res := response.QueryStruct{Success: true, Data: repairpolicy}
	resp.WriteEntity(res)
	return
}

// RepairPolicyAuthOperationHandler parses the http request and return authorized
//	operations of an repairpolicy.
// Usage :
//		GET /v1/repairPolicies/operations/{ParamID}
// Params :
//		ParamID : Storage identifier of the repairpolicy
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyAuthOperationHandler(req *restful.Request,
	resp *restful.Response) {
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)

	operations, code, err := services.GetRepairPolicyService().GetOperationById(objectId, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: operations}
	resp.WriteEntity(res)
	return
}

// RepairPolicyDeleteHandler parses the http request and delete the repairpolicy items.
// Usage :
//		DELETE /v1/repairpolicies/{ParamID}
// Params :
//		ParamID : Storage identifier of the repairpolicy
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyDeleteHandler(req *restful.Request, resp *restful.Response) {
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)

	code, err := services.GetRepairPolicyService().DeleteById(objectId, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	response.WriteSuccess(resp)
	return
}

// RepairPolicyCreateHandler parses the http request and store an repairpolicy.
// Usage :
//		POST /v1/repairPolicies
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyCreateHandler(req *restful.Request, resp *restful.Response) {
	// logrus.Infof("RepairPolicyCreateHandler is called!")
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	// Stub an repairpolicy to be populated from the body
	repairpolicy := entity.RepairPolicy{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&repairpolicy)
	if err != nil {
		logrus.Errorf("convert body to repairpolicy failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newRepairPolicy, code, err := services.GetRepairPolicyService().Create(repairpolicy, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.Response{Success: true, Data: newRepairPolicy}
	resp.WriteEntity(res)
	return
}

// RepairPolicyUpdateHandler parses the http request and update an repairpolicy.
// Usage :
//		PUT /v1/repairPolicies/{ParamID}
// Params :
//		ParamID : Storage identifier of the repairpolicy
// If successful,response code will be set to 201.
func (p *Resource) RepairPolicyUpdateHandler(req *restful.Request, resp *restful.Response) {
	// x_auth_token := req.HeaderParameter("X-Auth-Token")
	// code, err := services.TokenValidation(x_auth_token)
	// if err != nil {
	// 	response.WriteStatusError(code, err, resp)
	// 	return
	// }

	objectId := req.PathParameter(ParamID)

	// Stub an repairpolicy to be populated from the body
	repairpolicy := entity.RepairPolicy{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&repairpolicy)
	if err != nil {
		logrus.Errorf("convert body to repairpolicy failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	created, code, err := services.GetRepairPolicyService().UpdateById(objectId, repairpolicy, "x_auth_token")
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	p.successUpdate(objectId, created, req, resp)
}
