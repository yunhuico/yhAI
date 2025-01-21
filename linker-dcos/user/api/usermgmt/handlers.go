package usermgmt

import (
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Resource struct {
}

func (p *Resource) successList(ret interface{}, limitnum int, lenth int, req *restful.Request, resp *restful.Response) {
	res := response.QueryStruct{Success: true, Data: ret}
	// Get limit amount
	limit := 10
	if limitnum > 0 {
		limit = limitnum
	}

	// If got full limit set next link
	if lenth == limit {
		res.Prev, res.Next = prevnexturl(req)
	}

	// Count documents if count parameter is included in query
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = lenth
		resp.AddHeader("X-Object-Count", strconv.Itoa(lenth))
	}

	// Write result back to client
	resp.WriteEntity(res)
}

func (p *Resource) successUpdate(id string, created bool, req *restful.Request, resp *restful.Response) {
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

func prevnexturl(req *restful.Request) (prev string, next string) {
	return collectionurl(false, req), collectionurl(true, req)
}

func collectionurl(next bool, req *restful.Request) string {
	// Get current location url
	uri, _ := url.Parse(req.Request.URL.RequestURI())
	q := uri.Query()

	// Skip/limit values
	limitnum := queryIntParam(req, "limit", 10)
	skipnum := queryIntParam(req, "skip", 0)

	// Number of documents to skip
	if next {
		q.Set("skip", strconv.Itoa(skipnum+limitnum))
	} else {
		// prev
		prevskip := skipnum - limitnum
		if prevskip < 0 {
			prevskip = 0
		}
		if prevskip == skipnum {
			return ""
		}
		q.Set("skip", strconv.Itoa(prevskip))
	}

	// URL query
	uri.RawQuery = q.Encode()

	return uri.String()
}

func queryIntParam(req *restful.Request, name string, def int) int {
	num := def
	if strnum := req.QueryParameter(name); len(strnum) > 0 {
		num, _ = strconv.Atoi(strnum)
	}
	return num
}
