package documents

import (
	"encoding/json"
	"net/http"
	"os"

	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"linkernetworks.com/dcos-backend/cluster/services"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

func (p Resource) PubKeyWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/pubkey")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON, restful.MIME_OCTET)

	id := ws.PathParameter(ParamID, "Storage identifier of pubkey")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.PubKeyCreateHandler).
		Doc("Store a pubkey").
		Operation("PubKeyCreateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))

	ws.Route(ws.GET("/").To(p.PubKeyListHandler).
		Doc("Returns all pubkey items").
		Operation("PubKeyListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-Count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("name", "The name of pubkey wanted to query")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")).
		Param(ws.QueryParameter("user_id", "The owner ID of the pubkey")))

	ws.Route(ws.GET("/" + paramID).To(p.PubKeyGetHandler).
		Doc("Return a pubkey").
		Operation("PubKeyGetHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.DELETE("/" + paramID).To(p.PubKeyDeleteHandler).
		Doc("Detele a PubKey by its storage identifier").
		Operation("PubKeyDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))
	ws.Route(ws.POST("/userSelfCreate").To(p.SelfCreatePubKeyHandler).
		Doc("userself create a pubkey").
		Operation("SelfCreatePubKeyHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "").DataType("string")))
	ws.Route(ws.GET("/downLoadKey/{userId}").To(p.PubKeyDowmLoadHandler).
		Doc("user download pubkey").
		Operation("PubKeyDowmLoadHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.PathParameter("userId", "user identifier")))

	return ws

}

func (p *Resource) PubKeyDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("PubKeyDeleteHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")
	objectId := req.PathParameter(ParamID)
	pubkey, code, err := services.GetPubKeyService().DeleteById(objectId, x_auth_token)
	
	createLog(err, "delete_pubkey", "pubkey", pubkey.Name, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	// Write success response

	res := response.QueryStruct{Success: true}
	resp.WriteEntity(res)
	return
}
func (p *Resource) PubKeyDowmLoadHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("PubKeyDowmLoadHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	code, err := services.TokenValidation(x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	userId := req.PathParameter("userId")
	privateKeyPath := services.PUBKEY_STORAGEPATH_USER_SELF + userId + ""
	privateKeyFile := privateKeyPath + "/" + "user_key"
	_, err = os.Stat(privateKeyFile)
	if err == nil || os.IsExist(err) {
		resp.AddHeader("Content-Disposition:", "attachment;filename="+"id_rsa.txt")
		resp.AddHeader("Content-Type", "application/octet-stream")
		http.ServeFile(resp.ResponseWriter, req.Request, privateKeyFile)
		//to download once
		err := os.RemoveAll(privateKeyPath)
		logrus.Infof("start to save pubkey %v", err)
	}
	return
}
func (p *Resource) SelfCreatePubKeyHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("SelfCreatePubKeyHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")
	// Stub an pubkey to be populated from the body
	pubkey := entity.PubKey{}

	err := json.NewDecoder(req.Request.Body).Decode(&pubkey)
	if err != nil {
		logrus.Errorf("convert body to pubkey failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	userId := pubkey.UserId
	//to create and save pubkey ssh-keygen
	privateKeyFile, code, err := services.GetPubKeyService().CreateAndSave(userId, pubkey, x_auth_token)
	createLog(err, "create_pubkey", "pubkey", pubkey.Name + " " + privateKeyFile, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	//to return
	res := response.QueryStruct{Success: true, Data: userId}
	resp.WriteEntity(res)

	return
}
func (p *Resource) PubKeyGetHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("PubKeyGetHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	objectId := req.PathParameter(ParamID)
	pubkey, code, err := services.GetPubKeyService().QueryById(objectId, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}
	logrus.Debugf("pubkey is %v", pubkey)

	res := response.QueryStruct{Success: true, Data: pubkey}
	resp.WriteEntity(res)
	return

}

func (p *Resource) PubKeyListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("PubKeyListHandler is called!")
	x_auth_token := req.HeaderParameter("X-Auth-Token")

	var skip int = queryIntParam(req, "skip", 0)
	var limit int = queryIntParam(req, "limit", 0)

	var name string = req.QueryParameter("name")
	var user_id string = req.QueryParameter("user_id")

	var sort string = req.QueryParameter("sort")

	total, pubkeys, code, err := services.GetPubKeyService().QueryPubKey(name, user_id, skip, limit, sort, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	pubkeyInfo := services.GetPubKeyService().GetPubkeyInfo(pubkeys, x_auth_token)

	res := response.QueryStruct{Success: true, Data: pubkeyInfo}
	if c, _ := strconv.ParseBool(req.QueryParameter("count")); c {
		res.Count = total
		resp.AddHeader("X-Object-Count", strconv.Itoa(total))
	}
	resp.WriteEntity(res)
	return

}

func (p *Resource) PubKeyCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("PubKeyCreateHandler is called!")

	x_auth_token := req.HeaderParameter("X-Auth-Token")

	// Stub an pubkey to be populated from the body
	pubkey := entity.PubKey{}

	err := json.NewDecoder(req.Request.Body).Decode(&pubkey)
	if err != nil {
		logrus.Errorf("convert body to pubkey failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	newPubKey, code, err := services.GetPubKeyService().Save(pubkey, x_auth_token)
	createLog(err, "save_pubkey", "pubkey", newPubKey.Name, x_auth_token)
	if err != nil {
		response.WriteStatusError(code, err, resp)
		return
	}

	res := response.QueryStruct{Success: true, Data: newPubKey}
	resp.WriteEntity(res)
	return

}
