package documents

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) ComponentWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/components")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/").To(p.ComponentCreateHandler).
		Doc("Create a component").
		Operation("ComponentCreateHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "Entity component")))

	ws.Route(ws.GET("/").To(p.ComponentDetailHandler).
		Doc("Get a component").
		Operation("ComponentDetailHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("name", "Name of the component you want to get")))

	ws.Route(ws.PUT("/scale").To(p.ComponentScaleHandler).
		Doc("Scale instances of containers in this component").
		Operation("ComponentScaleHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("name", "Name of the component you want to scale")).
		Param(ws.QueryParameter("scaleto", "Integer number of new instances count")))

	ws.Route(ws.PUT("/stop").To(p.ComponentStopHandler).
		Doc("Stop a component").
		Operation("ComponentStopHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("name", "Name of the component you want to stop")))

	ws.Route(ws.PUT("/start").To(p.ComponentStartHandler).
		Doc("Start a component").
		Operation("ComponentStartHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("name", "Name of the component you want to start")))

	ws.Route(ws.DELETE("/").To(p.ComponentDeleteHandler).
		Doc("Delete a component").
		Operation("ComponentDeleteHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("name", "Name of the component you want to delete")))

	ws.Route(ws.PUT("/").To(p.ComponentUpdateHandler).
		Doc("Update a component").
		Operation("ComponentUpdateHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "The component entity")))
	return ws
}

func (p *Resource) ComponentCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentCreateHandler is called...")
	//decode
	component := entity.ComponentViewObj{}
	err := json.NewDecoder(req.Request.Body).Decode(&component)
	if err != nil {
		logrus.Errorf("convert body to entity failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newComponent, errorCode, err := services.GetComponentService().
		Create(component)

	logrus.Infof("start to create log")
	if newComponent != nil {
		go common.SendCreateLog(err, "create_component", "service", newComponent.App.ID)
	}

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: newComponent}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ComponentDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentDetailHandler is called...")
	name := req.QueryParameter("name")
	component, errorCode, err := services.GetComponentService().
		Detail(name)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: component}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ComponentUpdateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentUpdateHandler is called...")
	//decode body
	component := entity.ComponentViewObj{}
	err := json.NewDecoder(req.Request.Body).Decode(&component)
	if err != nil {
		logrus.Errorf("convert body to entity failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newComponent, errorCode, err := services.GetComponentService().
		Update(component)

	logrus.Infof("start to create log")
	if newComponent != nil {
		go common.SendCreateLog(err, "update_component", "service", newComponent.App.ID)
	}

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: newComponent}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ComponentScaleHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentScaleHandler is called...")
	name := req.QueryParameter("name")
	scaleTo := req.QueryParameter("scaleto")

	_, errorCode, err := services.GetComponentService().
		Scale(name, scaleTo)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "scale_component", "service", "scale "+name+" to "+string(scaleTo))

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ComponentStopHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentStopHandler is called...")
	name := req.QueryParameter("name")

	errorCode, err := services.GetComponentService().
		Stop(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "stop_component", "service", name)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}

func (p *Resource) ComponentStartHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentStartHandler is called...")
	name := req.QueryParameter("name")
	newComponent, errorCode, err := services.GetComponentService().
		Start(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "start_component", "service", name)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: newComponent}
	resp.WriteEntity(res)
	return
}

func (p *Resource) ComponentDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("ComponentDeleteHandler is called...")
	name := req.QueryParameter("name")
	errorCode, err := services.GetComponentService().
		Delete(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "delete_component", "service", name)

	logrus.Infof("start to send create log to clustermgmt")

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
	return
}
