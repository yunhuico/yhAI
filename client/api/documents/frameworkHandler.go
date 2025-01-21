package documents

import (
	"encoding/json"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"

	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) FrameworkWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/frameworks")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// ws.Route(ws.POST("/templates").To(p.TemplateInsertHandler).
	// 	Doc("Insert a framework template").
	// 	Operation("TemplateInsertHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
	// 	Param(ws.BodyParameter("body", "Entity FrameworkTemplate")))

	ws.Route(ws.GET("/templates").To(p.TemplateListHandler).
		Doc("List framwork templates").
		Operation("TemplateListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")))

	ws.Route(ws.GET("/templates/{name}").To(p.TemplateDetailHandler).
		Doc("Get detail of a framework template").
		Operation("TemplateDetailHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	// ws.Route(ws.DELETE("/templates/{name}").To(p.TemplateDeleteHandler).
	// 	Doc("Delete a framework template").
	// 	Operation("TemplateDeleteHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	ws.Route(ws.POST("/instances").To(p.InstanceCreateHandler).
		Doc("Create an instance of a framework").
		Operation("InstanceCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "Entity FrameworkInstance")))

	//name:Name of FrameworkInstance
	// ws.Route(ws.PUT("/instances/{name}/deploy").To(p.InstanceDeployHandler).
	// 	Doc("Start to deploy instance of a framework").
	// 	Operation("InstanceDeployHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	//name:Name of FrameworkInstance
	ws.Route(ws.DELETE("/instances/{name}").To(p.InstanceDeleteHandler).
		Doc("Delete an instance of framework").
		Operation("InstanceDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	ws.Route(ws.GET("/instances").To(p.InstanceListHandler).
		Doc("List instances of frameworks").
		Operation("InstanceListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("status", "Filter by status of instance of a framework")))

	ws.Route(ws.GET("/tasks").To(p.FinishedTasksListHandler).
		Doc("List finished tasks of frameworks").
		Operation("FinishedTasksListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("framework_name", "Name of framework")).
		Param(ws.QueryParameter("host_ip", "Host IP,Host detail container info")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")))

	return ws
}

// func (p *Resource) TemplateInsertHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infof("TemplateInsertHandler is called...")

// 	template := entity.FrameworkTemplate{}
// 	err := json.NewDecoder(req.Request.Body).Decode(&template)
// 	if err != nil {
// 		logrus.Errorf("convert body to framework template failed, error is %v", err)
// 		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
// 		return
// 	}

// 	newTemplate, code, err := services.GetFrameworkService().InsertTemplate(template)
// 	if err != nil {
// 		response.WriteStatusError(code, err, resp)
// 		return
// 	}

// 	res := response.QueryStruct{Success: true, Data: newTemplate}
// 	resp.WriteEntity(res)
// 	return
// }

func (p *Resource) TemplateListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TemplateListHandler is called...")

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var sort string = req.QueryParameter("sort")

	total, templates, code, err := services.GetFrameworkService().ListTemplates(skip, limit, sort)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: templates}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

func (p *Resource) TemplateDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TemplateDetailHandler is called...")

	var name = req.PathParameter("name")

	template, code, err := services.GetFrameworkService().GetTemplateDetail(name)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	res := response.QueryStruct{Success: true, Data: template}
	resp.WriteEntity(res)
	return
}

// func (p *Resource) TemplateDeleteHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infof("TemplateDeleteHandler is called...")

// 	name := req.PathParameter("name")

// 	code, err := services.GetFrameworkService().DeleteTemplate(name)
// 	if err != nil {
// 		response.WriteStatusError(code, err, resp)
// 		return
// 	}
// 	// Write success response
// 	res := response.QueryStruct{Success: true}
// 	resp.WriteEntity(res)
// 	return
// }

func (p *Resource) InstanceCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("InstanceCreateHandler is called...")

	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
	//	code, err := services.TokenValidation(x_auth_token)
	//	if err != nil {
	//		logrus.Errorln("token validation error is %v", err)
	//		response.WriteStatusError(code, err, resp)
	//		return
	//	}

	instance := entity.FrameworkInstance{}
	err := json.NewDecoder(req.Request.Body).Decode(&instance)
	if err != nil {
		logrus.Errorf("convert body to framework instance failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newInstance, code, err := services.GetFrameworkService().CreateInstance(instance)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newInstance}
	resp.WriteEntity(res)
	return
}

// func (p *Resource) InstanceDeployHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infof("InstanceDeployHandler is called...")

// 	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
// 	//	code, err := services.TokenValidation(x_auth_token)
// 	//	if err != nil {
// 	//		logrus.Errorln("token validation error is %v", err)
// 	//		response.WriteStatusError(code, err, resp)
// 	//		return
// 	//	}
// 	name := req.PathParameter("name")
// 	newInstance, code, err := services.GetFrameworkService().DeployInstance(name)
// 	if err != nil {
// 		response.WriteStatusError(code, err, resp)
// 		return
// 	}

// 	res := response.QueryStruct{Success: true, Data: newInstance}
// 	resp.WriteEntity(res)
// 	return
// }

func (p *Resource) InstanceDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("InstanceDeleteHandler is called...")

	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
	//	code, err := services.TokenValidation(x_auth_token)
	//	if err != nil {
	//		logrus.Errorln("token validation error is %v", err)
	//		response.WriteStatusError(code, err, resp)
	//		return
	//	}
	name := req.PathParameter("name")

	code, err := services.GetFrameworkService().DeleteInstance(name)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response
	res := response.QueryStruct{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) InstanceListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("InstanceListHandler is called...")

	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
	//	code, err := services.TokenValidation(x_auth_token)
	//	if err != nil {
	//		logrus.Errorln("token validation error is %v", err)
	//		response.WriteStatusError(code, err, resp)
	//		return
	//	}

	skip := queryIntParam(req, "skip", 0)
	limit := queryIntParam(req, "limit", 10)
	sort := req.QueryParameter("sort")

	total, instances, code, err := services.GetFrameworkService().ListInstances(skip, limit, sort)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: instances}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}

func (p *Resource) FinishedTasksListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("FinishedTasksListHandler is called...")

	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
	//	code, err := services.TokenValidation(x_auth_token)
	//	if err != nil {
	//		logrus.Errorln("token validation error is %v", err)
	//		response.WriteStatusError(code, err, resp)
	//		return
	//	}

	skip := queryIntParam(req, "skip", 0)
	limit := queryIntParam(req, "limit", 10)
	sort := req.QueryParameter("sort")
	frameWorkName := req.QueryParameter("framework_name")
	hostIP := req.QueryParameter("host_ip")

	total, finishedTasks, code, err := services.GetFrameworkService().ListFinishedTask(skip, limit, sort, frameWorkName, hostIP)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: finishedTasks}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return
}
