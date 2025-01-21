package response

import (
	"github.com/emicklei/go-restful"
)

type Error struct {
	Code     string `json:"code"`
	ErrorMsg string `json:"errormsg"`
}

func WriteError(err error, resp *restful.Response) {
	// Set error code to "internal error"
	errorCode := "E12003"

	// Write error response
	WriteStatusError(errorCode, err, resp)
}

func WriteStatusError(errorCode string, err error, resp *restful.Response) {
	success := NewResponse(false)
	success.SetError(err, errorCode)
	success.WriteStatus(500, resp)
}
