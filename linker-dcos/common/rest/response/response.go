package response

import (
	"github.com/emicklei/go-restful"
)

type Response struct {
	Success bool        `json:"success"`
	Error   *Error      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func WriteSuccess(resp *restful.Response) {
	success := NewResponse(true)
	success.Write(resp)
}

func WriteResponse(data interface{}, resp *restful.Response) {
	WriteResponseStatus(200, data, resp)
}

func WriteResponseStatus(status int, data interface{}, resp *restful.Response) {
	success := NewResponse(true)
	success.Data = data
	success.WriteStatus(status, resp)
}

func NewResponse(success bool) *Response {
	return &Response{Success: success}
}

func NewErrorResponse(err error) *Response {
	res := &Response{Success: false, Error: &Error{}}
	res.SetError(err, "E12003")
	return res
}

func (r *Response) SetError(err error, errorCode string) {
	if err != nil {
		if r.Error == nil {
			r.Error = &Error{}
		}

		r.Error.ErrorMsg = err.Error()
		r.Error.Code = errorCode
	}
}

func (r *Response) WriteStatus(status int, resp *restful.Response) {
	resp.WriteHeader(status)
	r.Write(resp)
}

func (r *Response) Write(resp *restful.Response) {
	resp.WriteEntity(r)
}
