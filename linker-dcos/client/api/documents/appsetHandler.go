package documents

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/client/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) AppsetWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/appsets")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	// create appset
	ws.Route(ws.POST("/").To(p.AppsetCreateHandler).
		Doc("Create appset").
		Operation("AppsetCreateHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.BodyParameter("body", "Entity appset containing marathon json")))

	// list appsets
	ws.Route(ws.GET("/").To(p.AppsetListHandler).
		Doc("List appsets").
		Operation("AppsetListHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")))

	// delete appset
	ws.Route(ws.DELETE("/{name}").To(p.AppsetDeleteHandler).
		Doc("Delete appset").
		Operation("AppsetDeleteHandler").
		Param(ws.PathParameter("name", "The name of the appset")))
	// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	// start appset, update all instance number to its expected number in appset to marathon
	ws.Route(ws.PUT("/{name}/start").To(p.AppsetStartHandler).
		Doc("Start appset").
		Operation("AppsetStartHandler").
		Param(ws.PathParameter("name", "The name of the appset")))
	// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	// stop appset, suspend all apps under this group in marathon
	ws.Route(ws.PUT("/{name}/stop").To(p.AppsetStopHandler).
		Doc("Stop appset").
		Operation("AppsetStopHandler").
		Param(ws.PathParameter("name", "The name of the appset")))
	// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")))

	// get appset detail, including components and tasks
	ws.Route(ws.GET("/{name}").To(p.AppsetDetailHandler).
		Doc("Get appset detail").
		Operation("AppsetDetailHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter("name", "The name of the appset")).
		Param(ws.QueryParameter("skip_group", "Do not return field marathon_group if set true").DataType("boolean")).
		Param(ws.QueryParameter("monitor", "Show monitor URL map if set true").DataType("boolean")))

	ws.Route(ws.GET("/{name}/apps").To(p.AppsetAppGetHandler).
		Doc("Get appset detail").
		Operation("AppsetAppGetHandler").
		Param(ws.PathParameter("name", "The name of the appset")))

	// update appset
	ws.Route(ws.PUT("/{name}").To(p.AppsetUpdateHandler).
		Doc("Update appset").
		Operation("AppsetUpdateHandler").
		// Param(ws.HeaderParameter("X-Auth-Token", "Authentication token")).
		Param(ws.PathParameter("name", "Name of the appset you want to update")).
		Param(ws.BodyParameter("body", "Entity component")))

	return ws
}

func (p *Resource) AppsetAppGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetAppGetHandler is called...")
	name := req.PathParameter("name")

	appname, errorCode, err := services.GetAppsetService().GetAppName(name)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: appname}
	resp.WriteEntity(res)
	return

}

func (p *Resource) AppsetCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetCreateHandler is called...")
	appset := entity.Appset{}
	err := json.NewDecoder(req.Request.Body).Decode(&appset)
	if err != nil {
		logrus.Errorf("convert body to entity failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newAppset, errorCode, err := services.GetAppsetService().Create(appset)
	if newAppset != nil {
		go common.SendCreateLog(err, "create_service", "service", newAppset.Name)
	}

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.Response{Success: true, Data: newAppset}
	resp.WriteEntity(res)
	return
}

//List appsets with basic info
func (p *Resource) AppsetListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetListHandler is called...")

	skip := queryIntParam(req, "skip", 0)
	limit := queryIntParam(req, "limit", 0)
	sort := req.QueryParameter("sort")

	//query appsets
	total, appsets, errorCode, err := services.GetAppsetService().List(skip, limit, sort)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: appsets}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}

	resp.WriteEntity(res)
	return
}

func (p *Resource) AppsetDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetDeleteHandler is called...")
	name := req.PathParameter("name")
	errorCode, err := services.GetAppsetService().Delete(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "delete_service", "service", name)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	response.WriteSuccess(resp)
	return
}

func (p *Resource) AppsetStartHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetStartHandler is called...")
	name := req.PathParameter("name")
	errorCode, err := services.GetAppsetService().Start(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "start_service", "service", name)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	response.WriteSuccess(resp)
	return
}

func (p *Resource) AppsetStopHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetStopHandler is called...")
	name := req.PathParameter("name")
	errorCode, err := services.GetAppsetService().Stop(name)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "stop_service", "service", name)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	response.WriteSuccess(resp)
	return
}

//Get appset
func (p *Resource) AppsetDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Debugf("AppsetDetailHandler is called...")

	name := req.PathParameter("name")
	skip_group, err := queryBoolParam(req, "skip_group", false)
	if err != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	displayMonitor, err := queryBoolParam(req, "monitor", false)
	if err != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	appset, errorCode, err := services.GetAppsetService().GetDetail(name, skip_group, displayMonitor)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	res := response.QueryStruct{Success: true, Data: appset}
	resp.WriteEntity(res)
	return
}

func (p *Resource) AppsetUpdateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("AppsetUpdateHandler is called...")

	appsetName := req.PathParameter("name")
	//decode body
	appset := entity.Appset{}
	err := json.NewDecoder(req.Request.Body).Decode(&appset)
	if err != nil {
		logrus.Warningf("udpate appset convert body to entity failed, error is %v", err)
		response.WriteStatusError(services.APPSET_ERR_DECODE_JSON, err, resp)
		return
	}

	if appsetName != appset.Name {
		err := errors.New("name is not equal to appset.Name")
		logrus.Warningf("udpate appset failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newAppset, errorCode, err := services.GetAppsetService().Update(appset)

	logrus.Infof("start to create log")
	go common.SendCreateLog(err, "update_service", "service", appsetName)

	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newAppset}
	resp.WriteEntity(res)
	return
}
