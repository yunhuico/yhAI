package usermgmt

import (
	"encoding/json"
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"github.com/emicklei/go-restful"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/user/services"
)

func (p Resource) TokenService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/token")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of user")
	paramID := "{" + ParamID + "}"

	ws.Route(ws.POST("/").To(p.TokenCreateHandler).
		Doc("Create a new user token").
		Operation("TokenCreateHandler").
		Param(ws.BodyParameter("body", "Token create body in json format,for example {\"username\":\"...\", \"password\":\"...\", \"tenantname\":\"...\"}").DataType("string")))

	ws.Route(ws.GET("/").To(p.TokenValidateHandler).
		Doc("Return valid user token").
		Operation("TokenValidateHandler").
		Param(ws.QueryParameter("token", "User token field")))

	ws.Route(ws.GET("/" + paramID).To(p.TokenDetailHandler).
		Doc("Return a token by its storage identifier").
		Operation("TokenDetailHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	// ws.Route(ws.GET("/ids/").To(p.GetIdsFromTokenHandler).
	// 	Doc("Return  user id and tenant id from token").
	// 	Operation("GetIdsFromTokenHandler").
	// 	Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")))

	ws.Route(ws.POST("/regenerate/").To(p.TokenReGenerateHandler).
		Doc("Generate another user's token from current token").
		Operation("TokenReGenerateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "Token generate body in json format,for example {\"user_id\":\"...\", \"tenant_id\":\"...\"}").DataType("string")))

	return ws
}

// TokenReGenerateHandler parses the http request and generate
// another user's token from current token.
// Usage :
//		POST /v1/token/regenerate
// If successful,response code will be set to 201.
func (p *Resource) TokenReGenerateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infoln("TokenReGenerateHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")

	document := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&document)
	if err != nil {
		logrus.Errorf("decode token generate err is %v", err)
		response.WriteStatusError(services.TOKEN_ERROR_CREATE, err, resp)
		return
	}

	document, err = mejson.Unmarshal(document)
	if err != nil {
		logrus.Errorf("unmarshal token generate err is %v", err)
		response.WriteStatusError(services.TOKEN_ERROR_CREATE, err, resp)
		return
	}

	userid := document["user_id"]
	tenantid := document["tenant_id"]
	if userid == nil || tenantid == nil {
		logrus.Errorln("invalid parameter! user and tenant should not be null!")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("invalid parameter! user and tenant should not be null!"), resp)
		return
	}

	tokenId, errorCode, err := services.GetTokenService().TokenReGenerate(token, userid.(string), tenantid.(string))
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(tokenId, resp)

	return
}

func (p *Resource) TokenDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TokenDetailHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("token id should not be null for token detail operation")
		response.WriteStatusError(services.TOKEN_ERROR_GET, errors.New("token id should not be null for get user operation"), resp)
		return
	}

	ret, errorCode, err := services.GetTokenService().TokenDetail(token, id)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(ret, resp)
}

// GetIdsFromTokenHandler parses the http request and return
// user id and tenant id from token.
// Usage :
//		GET /v1/token/ids/
// If successful,response code will be set to 201.
// func (p *Resource) GetIdsFromTokenHandler(req *restful.Request, resp *restful.Response) {
// 	logrus.Infof("GetIdsFromTokenHandler is called!")
// 	tokenid := req.HeaderParameter("X-Auth-Token")

// 	ids, errorCode, err := services.GetTokenService().GetIdsFromToken(tokenid)
// 	if err != nil {
// 		response.WriteStatusError(errorCode, err, resp)
// 		return
// 	}

// 	response.WriteResponse(ids, resp)

// 	return
// }

// TokenCreateHandler parses the http request and create a new user token.
// Usage :
//		POST /v1/token
// If successful,response code will be set to 201.
func (p *Resource) TokenCreateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TokenCreateHandler is called!")

	doc := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&doc)
	if err != nil {
		logrus.Errorf("decode credential err is %v", err)
		response.WriteStatusError(services.TOKEN_ERROR_CREATE, err, resp)
		return
	}

	username, passwd, tenant, paraErr := tokenCreateParamCheck(doc)
	if paraErr != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, paraErr, resp)
		return
	}

	ret, errorCode, err := services.GetTokenService().TokenCreate(username, passwd, tenant)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(ret, resp)

	return
}

func tokenCreateParamCheck(doc interface{}) (email string, passwd string, tenant string, paraErr error) {
	var document interface{}
	document, paraErr = mejson.Marshal(doc)
	if paraErr != nil {
		logrus.Errorf("marshal user credential err is %v", paraErr)
		return
	}

	docJson := document.(map[string]interface{})
	usernameDoc := docJson["username"]
	if usernameDoc == nil {
		logrus.Errorln("invalid parameter ! username can not be null")
		paraErr = errors.New("invalid parameter!")
		return
	} else {
		email = usernameDoc.(string)
	}

	passwordDoc := docJson["password"]
	if passwordDoc == nil {
		logrus.Errorln("invalid parameter ! password can not be null")
		paraErr = errors.New("invalid parameter!")
		return
	} else {
		passwd = passwordDoc.(string)
	}

	tenantDoc := docJson["tenantname"]
	if tenantDoc == nil {
		logrus.Errorln("invalid parameter ! tenantname can not be null")
		paraErr = errors.New("invalid parameter!")
		return
	} else {
		tenant = tenantDoc.(string)
	}

	return
}

// TokenValidateHandler parses the http request and return valid user token.
// Usage :
//		GET /v1/token
// If successful,response code will be set to 201.
func (p *Resource) TokenValidateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("TokenValidateHandler is called!")

	var tokenId = req.QueryParameter("token")
	if len(tokenId) <= 0 {
		logrus.Errorln("invalie parameter! tokenId can not be null for token validation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("invalie parameter! tokenId can not be null for token validation"), resp)
		return
	}

	errorCode, err := services.GetTokenService().TokenValidate(tokenId)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	response.WriteSuccess(resp)

}
