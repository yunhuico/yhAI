package usermgmt

import (
	"encoding/json"
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"github.com/emicklei/go-restful"
	"gopkg.in/mgo.v2/bson"
//	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/user/services"
)

func (p Resource) UserService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/v1/user")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON)

	id := ws.PathParameter(ParamID, "Storage identifier of user")
	paramID := "{" + ParamID + "}"

	// user
	ws.Route(ws.POST("/").To(p.UserCreateUserHandler).
		Doc("Registry a new user").
		Operation("UserCreateUserHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.BodyParameter("body", "User registry request body in json format,for example {\"username\":\"...\", \"password\":\"...\", \"email\":\"...\"}").DataType("string")))

	ws.Route(ws.POST("/login").To(p.UserLoginHandler).
		Doc("Login with an exist user").
		Operation("UserLoginHandler").
		Param(ws.BodyParameter("body", "User login request body in json format,for example {\"username\":\"...\", \"password\":\"...\"}").DataType("string")))

	ws.Route(ws.DELETE("/" + paramID).To(p.UserDeleteHandler).
		Doc("Delete a user by its storage identifier").
		Operation("UserDeleteHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.GET("/").To(p.UserListHandler).
		Doc("Return all user items").
		Operation("UserListHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("count", "Count total items and return the result in X-Object-count header").DataType("boolean")).
		Param(ws.QueryParameter("skip", "Number of items to skip in the result set, default=0")).
		Param(ws.QueryParameter("limit", "Maximum number of items in the result set, default=0")).
		Param(ws.QueryParameter("sort", "Comma separated list of field names to sort")))

	ws.Route(ws.GET("/" + paramID).To(p.UserDetailHandler).
		Doc("Return a user by its storage identifier").
		Operation("UserDetailHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))
		
	ws.Route(ws.GET("/forcluster/" + paramID).To(p.UserDetailForClusterHandler).
		Doc("Return a user by its storage identifier").
		Operation("UserDetailForClusterHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id))

	ws.Route(ws.GET("/validate").To(p.UserValidateHandler).
		Doc("Return is the user is exit.").
		Operation("UserValidateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(ws.QueryParameter("username", "")))

	ws.Route(ws.PUT("/" + paramID).To(p.UserUpdateHandler).
		Doc("Updata a exist user by its storage identifier").
		Operation("UserUpdateHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "User update request body in json format,for example {\"company\":\"...\", \"email\":\"...\"}").DataType("string")))

	ws.Route(ws.PUT("/forget").To(p.ForgetPasswdHandler).
		Doc("forget password create new passwd and send email").
		Operation("ForgetPasswdHandler").
		Param(ws.QueryParameter("ip", "")).
		Param(ws.QueryParameter("username", "")))

	ws.Route(ws.PUT("/forget/change").To(p.ForgetChangeHandler).
		Doc("forget password create new passwd and send email").
		Operation("ForgetChangeHandler").
		Param(ws.QueryParameter("activecode", "")).
		Param(ws.QueryParameter("newpassword", "")).
		Param(ws.QueryParameter("username", "")))

	ws.Route(ws.PUT("/changepassword/" + paramID).To(p.UserChangePasswdHandler).
		Doc("change password of an exist user by its storage identifier").
		Operation("UserChangePasswdHandler").
		Param(ws.HeaderParameter("X-Auth-Token", "A valid authentication token")).
		Param(id).
		Param(ws.BodyParameter("body", "User login request body in json format,for example {\"password\":\"...\", \"newpassword\":\"...\",\"confirm_newpassword\":\"...\"}").DataType("string")))

	return ws
}

func (p *Resource) ForgetChangeHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ForgetChangeHandler is called!")
	username := req.QueryParameter("username")
	activecode := req.QueryParameter("activecode")
	newpassword := req.QueryParameter("newpassword")

	if len(username) == 0 || len(newpassword) == 0 || len(activecode) == 0 {
		logrus.Warnln("this parameter should not be null")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("parameter should not be null"), resp)
		return
	}

	errorCode, err := services.GetUserService().Change(username, activecode, newpassword)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)

}

func (p *Resource) ForgetPasswdHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("ForgetPasswdHandler is called!")
	username := req.QueryParameter("username")
	ip := req.QueryParameter("ip")
	if len(username) == 0 || len(ip) == 0 {
		logrus.Warnln("username or ip should not be null")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("username should not be null"), resp)
		return
	}
	logrus.Infof("username is %v", username)
	logrus.Infof("ip is %v", ip)

	errorCode, err := services.GetUserService().Forget(username, ip)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
}

func (p *Resource) UserValidateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserValidateHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")
	username := req.QueryParameter("username")
	if len(username) == 0 {
		logrus.Warnln("username should not be null")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("username should not be null"), resp)
		return
	}
	logrus.Infof("username is %v", username)
	logrus.Infof("start to test username")

	errorCode, _, err := services.GetUserService().Validate(username, token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)

}

// CheckAndGenerateToken parses the http request and registry a new user.
// Usage :
//		POST /v1/user/registry
// If successful,response code will be set to 201.
func (p *Resource) UserCreateUserHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserCreateUserHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	doc := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&doc)
	if err != nil {
		logrus.Errorf("decode user err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	username, email, company, roletype, paraErr := userRegistryParamCheck(doc)
	if paraErr != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, paraErr, resp)
		return
	}

	if len(email) == 0 || len(username) == 0 {
		logrus.Errorln("parameter can not be null!")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("Invalid parameter"), resp)
		return
	}
	if len(username) > 10 {
		logrus.Errorln("username can not longer than 10")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("username is too long, please less than 10"), resp)
		return
	}

	userParam := services.UserParam{
		UserName: username,
		Email:    email,
		//		Password: password,
		Company: company,
		RoleType: roletype}
	errorCode, userId, err := services.GetUserService().Create(userParam, token)
	services.SendLog(err, "create_user", "user", username, token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successUpdate(userId, true, req, resp)

}

func userRegistryParamCheck(doc interface{}) (username string, email string, company string, roletype string, paraErr error) {
	var document interface{}
	document, paraErr = mejson.Marshal(doc)
	if paraErr != nil {
		logrus.Errorf("marshal user err is %v", paraErr)
		return
	}

	docJson := document.(map[string]interface{})
	emailDoc := docJson["email"]
	if emailDoc == nil {
		logrus.Errorln("invalid parameter ! email can not be null")
		paraErr = errors.New("Invalid parameter!")
		return
	} else {
		email = emailDoc.(string)
	}

	usernameDoc := docJson["username"]
	if usernameDoc == nil {
		logrus.Errorln("invalid parameter ! username can not be null")
		paraErr = errors.New("Invalid parameter!")
		return
	} else {
		username = usernameDoc.(string)
	}

	//	password = "password"

	companyDoc := docJson["company"]
	if companyDoc != nil {
		company = companyDoc.(string)
	}
	
	roleType := docJson["roleType"]
	if roleType == nil {
		logrus.Errorln("invalid parameter ! role can not be null")
		paraErr = errors.New("Invalid parameter!")
		return
	} else {
		roletype = roleType.(string)
	}

	return
}

// UserLoginHandler parses the http request and login with an exist user.
// Usage :
//		POST v1/user/login
// If successful,response code will be set to 201.
func (p *Resource) UserLoginHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserLoginHandler is called!")

	doc := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&doc)
	if err != nil {
		logrus.Errorf("decode user err is %v", err)
		response.WriteStatusError(services.USER_ERROR_LOGIN, err, resp)
		return
	}
	username, password, paraErr := userLoginParamCheck(doc)
	if paraErr != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, paraErr, resp)
		return
	}

	if len(username) == 0 || len(password) == 0 {
		logrus.Errorf("username and password can not be null!")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("Username or password can not be null"), resp)
		return
	}

	errorCode, loginRes, err := services.GetUserService().UserLogin(username, password)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(loginRes, resp)

	return

}

func userLoginParamCheck(doc interface{}) (username string, password string, paraErr error) {
	var document interface{}
	document, paraErr = mejson.Marshal(doc)
	if paraErr != nil {
		logrus.Errorf("marshal user err is %v", paraErr)
		return
	}

	docJson := document.(map[string]interface{})
	usernameDoc := docJson["username"]
	if usernameDoc == nil {
		logrus.Errorln("invalid parameter ! username can not be null")
		paraErr = errors.New("Invalid parameter!")
		return
	} else {
		username = usernameDoc.(string)
	}

	passwordDoc := docJson["password"]
	if passwordDoc == nil {
		logrus.Errorln("invalid parameter ! password can not be null")
		paraErr = errors.New("Invalid parameter!")
		return
	} else {
		password = passwordDoc.(string)
	}
	return
}

// UserUpdateHandler parses the http request and updata a exist user.
// Usage :
//		PUT /v1/user/{ParamID}
// Params :
//		ParamID : storage identifier of user
// If successful,response code will be set to 201.
func (p *Resource) UserUpdateHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserUpdateHanlder is called!")
	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("user id should not be null for update operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("user id should not be null for update operation"), resp)
		return
	}
	
	doc := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&doc)
	if err != nil {
		logrus.Errorf("decode user err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}
	
	username, email, company, roletype, paraErr := userRegistryParamCheck(doc)
	if paraErr != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, paraErr, resp)
		return
	}
	
	userParam := services.UserParam{
		UserName: username,
		Email:    email,
		//		Password: password,
		Company: company,
		RoleType: roletype}
		
	created, id, errorCode, err := services.GetUserService().UserUpdate(token, userParam, id)
	services.SendLog(err, "update_user", "user", username, token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}
	
	p.successUpdate(id, created, req, resp)

/*	newuser := entity.User{}

	// Populate the user data
	err := json.NewDecoder(req.Request.Body).Decode(&newuser)
	if err != nil {
		logrus.Errorf("convert body to user failed, error is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	created, id, errorCode, err := services.GetUserService().UserUpdate(token, newuser, id)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successUpdate(id, created, req, resp)*/
}

// UserChangePasswdHandler parses the http request and change
// password of an exist user.
// Usage :
//		PUT v1/user/changepassword/{ParamID}
// Params :
//		ParamID : storage identifier of user
// If successful,response code will be set to 201.
func (p *Resource) UserChangePasswdHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserChangePasswdHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("user id should not be null for change password operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("user id should not be null for update operation"), resp)
		return
	}

	document := bson.M{}
	decoder := json.NewDecoder(req.Request.Body)
	err := decoder.Decode(&document)
	if err != nil {
		logrus.Errorf("decode change password object err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INTERNAL, err, resp)
		return
	}

	document, err = mejson.Unmarshal(document)
	if err != nil {
		logrus.Errorf("unmarshal change password obejct err is %v", err)
		response.WriteStatusError(services.COMMON_ERROR_INTERNAL, err, resp)
		return
	}

	password := document["password"]
	newpwd1 := document["newpassword"]
	newpwd2 := document["confirm_newpassword"]
	if password == nil || newpwd1 == nil || newpwd2 == nil {
		logrus.Errorln("invalid parameter! password and newpassword field should not be null")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("invalid parameter!password, newpassword and confirm_newpassword should not be null!"), resp)
		return
	}

	user, created, errorCode, err := services.GetUserService().UserChangePassword(token, id, password.(string), newpwd1.(string), newpwd2.(string))
	services.SendLog(err, "change_password", "user", user.Username, token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successUpdate(id, created, req, resp)

}

// UserDeleteHandler parses the http request and delete a user.
// Usage :
//		DELETE /v1/user/{ParamID}
// Params :
//		ParamID : storage identifier of user
// If successful,response code will be set to 201.
func (p *Resource) UserDeleteHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserDeleteHandler is called!")
	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("user id should not be null for delete operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("user id should not be null for delete operation"), resp)
		return
	}

	user, errorCode, err := services.GetUserService().UserDelete(token, id)
	services.SendLog(err, "delete_user", "user", user.Username, token)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteSuccess(resp)
}

// UserListHandler parses the http request and return the user items.
// Usage :
//		GET /v1/user
//		GET /v1/user/{ParamID}
// Params :
//		ParamID : storage identifier of user
// If successful,response code will be set to 201.
func (p *Resource) UserListHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserListHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	limitnum := queryIntParam(req, "limit", 10)
	skipnum := queryIntParam(req, "skip", 0)
	sort := req.QueryParameter("sort")

	ret, count, errorCode, err := services.GetUserService().UserList(token, limitnum, skipnum, sort)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	p.successList(ret, limitnum, count, req, resp)
}

func (p *Resource) UserDetailHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserDetailHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("user id should not be null for user detail operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("user id should not be null for get user operation"), resp)
		return
	}

	ret, errorCode, err := services.GetUserService().UserDetail(token, id)
	if err != nil {
		response.WriteStatusError(errorCode, err, resp)
		return
	}

	response.WriteResponse(ret, resp)
}

func (p *Resource) UserDetailForClusterHandler(req *restful.Request, resp *restful.Response) {
	logrus.Infof("UserDetailForClusterHandler is called!")

	token := req.HeaderParameter("X-Auth-Token")
	id := req.PathParameter(ParamID)
	if len(id) <= 0 {
		logrus.Warnln("user id should not be null for user detail operation")
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, errors.New("user id should not be null for get user operation"), resp)
		return
	}

	user, err := services.GetUserService().UserDetailForCluster(token, id)
	if err != nil {
		response.WriteStatusError(services.COMMON_ERROR_INVALIDATE, err, resp)
		return
	}

	response.WriteResponse(user, resp)
}

