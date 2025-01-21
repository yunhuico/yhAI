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

func (p Resource) DockerRegistryWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/dockerregistries")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON, restful.MIME_OCTET)

	id := ws.PathParameter(ParamID, "Docker Registry related apis")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.DockerRegistryCreateHandler).
		Doc("Create a docker registry").
		Operation("DockerRegistryCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.DockerRegistryListHandler).
		Doc("Returns all docker registries by user id").
		Operation("DockerRegistryListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_id", "The owner ID of the pubkey")))

	ws.Route(ws.GET("/" + paramID).To(p.DockerRegistrySingleHandler).
		Doc("Return a docker registry by id").
		Operation("DockerRegistrySingleHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.GET("/registryValidate").To(p.DockerRegistryNameCheckHandler).
		Doc("Check the name of registry if there is the same one.").
		Operation("DockerRegistryNameCheckHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("type", "name or used")).
		Param(ws.QueryParameter("name", "Registry name to be checked")).
		Param(ws.QueryParameter("user_id", "User ID")))

	ws.Route(ws.DELETE("/" + paramID).To(p.DockerRegistryDeleteHandler).
		Doc("Detele a DockerRegistry").
		Operation("DockerRegistryDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	return ws

}

func (p *Resource) DockerRegistryDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("DockerRegistryDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)

	registry, code, err := services.GetDockerRegistryService().DeleteById(objectId, x_auth_token)
	logrus.Infof("start to create delete registry log")
	createLog(err, "delete_registry", "registry", registry.Name, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response

	res := response.QueryStruct{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) DockerRegistrySingleHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DockerRegistrySingleHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	registry, code, err := services.GetDockerRegistryService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("registry is %v", registry)

	res := response.QueryStruct{Success: true, Data: registry}
	resp.WriteEntity(res)
	return

}

func (p *Resource) DockerRegistryNameCheckHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DockerRegistryNameCheckHandler is called!")
	//	x_auth_token := req.HeaderParameter("X-Auth-Token")
	name := req.QueryParameter("name")
	userId := req.QueryParameter("user_id")
	typestring := req.QueryParameter("type")

	if typestring == "name" {
		ok, _, _ := services.GetDockerRegistryService().IsRegistryNameExist(name, userId)
		if ok {
			response.WriteStatusError("E57001", errors.New("Registry name exists."), resp)
			return
		}

	} else if typestring == "used" {
		used, _, _ := services.GetDockerRegistryService().IsRegistryUsed(name, userId)
		if used {
			response.WriteStatusError("E57008", errors.New("Registry used."), resp)
			return
		}

	}

	res := response.QueryStruct{Success: true}
	resp.WriteEntity(res)
	return

}

func (p *Resource) DockerRegistryListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DockerRegistryListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	//var name string = req.QueryParameter("name")
	var user_id string = req.QueryParameter("user_id")

	var sort string = req.QueryParameter("sort")

	total, pubkeys, code, err := services.GetDockerRegistryService().QueryDockerRegistries("", user_id, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	
	dockerregistryinfo := services.GetDockerRegistryService().GetRegistryInfo(pubkeys, x_auth_token)
	
	res := response.QueryStruct{Success: true, Data: dockerregistryinfo}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}

func (p *Resource) DockerRegistryCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("DockerRegistryCreateHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")

	// Stub an pubkey to be populated from the body
	registry := entity.DockerRegistry{}

	err := json.NewDecoder(req.Request.Body).Decode(&registry)
	if err != nil {
		logrus.Errorf("convert body to docker registry failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	newRegistry, code, err := services.GetDockerRegistryService().Save(registry, x_auth_token)	
	createLog(err, "create_registry", "registry", registry.Name, x_auth_token)	
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newRegistry}
	resp.WriteEntity(res)
	return

}
