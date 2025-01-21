package documents

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/cluster/services"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

type Resource struct {
}

func (p *Resource) successUpdate(id string, created bool,
	req *restful.Request, resp *restful.Response) {
	// Updated document API location
	docpath := documentLocation(req, id)

	// Content-Location header
	resp.AddHeader("Content-Location", docpath)

	// Information about updated document
	info := response.UpdateStruct{created, docpath}

	if created {
		response.WriteResponseStatus(http.StatusCreated, info, resp)
	} else {
		response.WriteResponse(info, resp)
	}
}

//
// Return document location URL
//
func documentLocation(req *restful.Request, id string) (location string) {
	// Get current location url
	location = strings.TrimRight(req.Request.URL.RequestURI(), "/")

	// Remove id from current location url if any
	if reqId := req.PathParameter(ParamID); reqId != "" {
		idlen := len(reqId)
		// If id is in current location remove it
		if noid := len(location) - idlen; noid > 0 {
			if id := location[noid : noid+idlen]; id == reqId {
				location = location[:noid]
			}
		}
		location = strings.TrimRight(location, "/")
	}

	// Add id of the document
	return location + "/" + id
}

func queryIntParam(req *restful.Request, name string, def int) int {
	num := def
	if strnum := req.QueryParameter(name); len(strnum) > 0 {
		num, _ = strconv.Atoi(strnum)
	}
	return num
}

func createLog(errs error, operation string, queryType string, comments string, x_auth_token string) (err error) {
	var status string
	if errs != nil {
		status = "fail"
	} else {
		status = "success"
	}
	newlog := services.CreatelogCluster(status, operation, queryType, comments)
	log, _, logerr := services.GetLogService().Create(&newlog, x_auth_token)
	logrus.Infof("log is %v", log)
	if logerr != nil {
		logrus.Errorf("save log to db error is [%v]", logerr)
		return
	}
	return
}
