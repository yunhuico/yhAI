package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"gopkg.in/gomail.v2"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/user/common"
)

var sysadmin_user = "sysadmin"
var sysadmin_pass = "password"
var sys_tenant = "sysadmin"
var sys_admin_role = "sysadmin"
var admin_role = "admin"
var common_role = "common"

var USER_ERROR_REG = "E10000"
var USER_ERROR_EXCEED = "E10001"
var USER_ERROR_CREATE = "E10003"
var USER_ERROR_NOEXIST = "E10004"
var USER_ERROR_WRONGPW = "E10005"
var USER_ERROR_UPDATE = "E10007"
var USER_ERROR_EXIST = "E10009"
var USER_ERROR_DELETE = "E10010"
var USER_ERROR_GET = "E10011"
var USER_ERROR_LOGIN = "E10012"
var USER_ERROR_EXISTCLUSTER = "E10013"
var USER_ERROR_LEGAL = "E10014"
var EMAIL_ERROR_SEND = "E10015"
var USER_ERROR_QUERYUSER = "E10016"
var USER_ERROR_ISNOTONLY = "E10017"
var USER_ERROR_PARSECODE = "E10018"
var USER_ERROR_EXPIRE = "E10019"
var USER_ERROR_CANNOT_CHANGE_SYSADMIN_USER_ROLE = "E10020"
var USER_SRROR_CANNOT_CHANGE_OWN_ROLE = "E10021"

var userService *UserService = nil
var userOnce sync.Once

var subjectTemplate = []byte(`[Linker Cloud Platform] Your account has been registered successfully!`)

var key = "Pa55w0rd"
var activeCode_expire = 60 * 60 * 72

var bodyTemplate = []byte(`
<USER_NAME>, 这封邮件是由领科云发送的。

您账户的初始密码为：<ENDPOINT>

如果有任何问题，请发送邮件到 support@linkernetworks.com 
请勿回复该邮件


此致
领科云管理团队



<USER_NAME>, This email is sent by Linker Cloud Platform.

The initial password for youraccount is :<ENDPOINT>

Any problems, please send mail to support@linkernetworks.com
Please DO NOT reply this mail

Thanks & BestRegards!

Linker Cloud Platform Team
`)

var emailBody = []byte(`<USER_NAME>, 这封邮件是由Linker DCOS发送的。
您收到这封邮件，是由于您申请了用户修改密码。
如果您没有访问过Linker DCOS或没有进行上述操作，请忽略这封邮件。

-------------------------------------------------------------------------------
	               用户修改密码说明
------------------------------------------------------------------------------- 
	         
如果您申请了修改密码，我们需要对您的地址有效性进行验证以避免垃圾邮件或地址被滥用。

您只需点击下面的链接即可进行修改密码，以下链接有效期为3天。过期可以重新请求发送一封新的邮件验证：
<ENDPOINT>

如果有任何问题，请发送邮件到 support@linkernetworks.com 
请勿回复该邮件

感谢您的访问，祝您使用愉快！

此致
Linker DCOS管理团队





USER, This email is sent by Linker DCOS Platform.
You applied for modify password on Liner DCOS Platform,
Please ignore this email if you have not done above operations.

-------------------------------------------------------------------------------
	               usage instruction
------------------------------------------------------------------------------- 
	         
Please click below link to complete your apply. 
To protect your account, the verification link is only valid for 3 days.
NEWURL

Any problems, please send mail to support@linkernetworks.com
Please DO NOT reply this mail

Thanks & BestRegards!

Linker DCOS Platform Team`)

type UserService struct {
	userCollectionName string
}

func GetUserService() *UserService {
	userOnce.Do(func() {
		userService = &UserService{"user"}

		userService.initialize()
	})

	return userService
}

func (p *UserService) initialize() bool {
	logrus.Infoln("UserMgmt initialize...")

	logrus.Infoln("check sysadmin tenant")

	sysTenantId, tenantErr := GetTenantService().createAndInsertTenant(sys_tenant, "system admin tenant")
	if tenantErr != nil {
		logrus.Errorf("create and insert sys admin tenant error,  err is %v", tenantErr)
		return false
	}

	logrus.Infoln("check sysadmin role")
	sysRoleId, roleErr := GetRoleService().createAndInsertRole(sys_admin_role, "sysadmin role")
	if roleErr != nil {
		logrus.Errorf("create and insert sys admin role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check admin role")
	_, roleErr = GetRoleService().createAndInsertRole(admin_role, "admin role")
	if roleErr != nil {
		logrus.Errorf("create and insert admin role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check common role")
	_, roleErr = GetRoleService().createAndInsertRole(common_role, "common role")
	if roleErr != nil {
		logrus.Errorf("create and insert common role error,  err is %v", roleErr)
		return false
	}

	logrus.Infoln("check sysadmin user")
	encryPassword := HashString(sysadmin_pass)
	_, userErr := p.createAndInsertUser(sysadmin_user, encryPassword, sysadmin_user, sysTenantId, sysRoleId, sys_admin_role, "")
	if userErr != nil {
		logrus.Errorf("create and insert sysadmin user error,  err is %v", userErr)
		return false
	}

	return true
}

func (p *UserService) deleteUserById(userId string) (err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for deleteUserById: ", userId)
		err = errors.New("invalid object id for deleteUserById")
		return
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	err = dao.HandleDelete(p.userCollectionName, true, selector)
	return
}

func (p *UserService) Validate(username string, token string) (erorCode string, userid string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return code, "", err
	}

	if authorized := GetAuthService().Authorize("get_user", token, "", p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return COMMON_ERROR_UNAUTHORIZED, "", errors.New("Required opertion is not authorized!")
	}

	currentUser, err := p.getAllUserByName(username)

	if err != nil {
		logrus.Errorf("get all user by name err is %v", err)
		return "", "", err
	}
	if len(currentUser) == 0 {
		return "", "", nil
	} else {
		logrus.Infoln("user already exist! username:", username)
		userId := currentUser[0].ObjectId.Hex()
		return USER_ERROR_EXIST, userId, errors.New("user already exist!")
	}
}

func (p *UserService) Create(userParam UserParam, token string) (errorCode string, userId string, err error) {

	if len(userParam.UserName) == 0 || len(userParam.Email) == 0 {
		logrus.Error("invalid parameter for user create!")
		return "", COMMON_ERROR_INVALIDATE, errors.New("invalid parameter! parameter should not be null")
	}
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return code, "", err
	}

	if authorized := GetAuthService().Authorize("create_user", token, "", p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return "", COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	username := userParam.UserName
	email := userParam.Email
	//	password := userParam.Password
	company := userParam.Company
	password := createpassword(8)
	if len(password) != 8 {
		logrus.Error("invalid parameter for user password!")
		return "", COMMON_ERROR_INVALIDATE, errors.New("password is error")
	}

	if !IsUserNameValid(username) {
		logrus.Errorf("username invalid! username is: %v", username)
		return COMMON_ERROR_INVALIDATE, "", errors.New("invalid username")
	}

	_, errQ := GetTenantService().getTenantByName(username)
	if errQ == nil {
		logrus.Errorln("user already exist!")
		return USER_ERROR_EXIST, "", errors.New("The username has already been registered, please specified another one!")
	}

	encryPassword := HashString(password)

	tenantId, errte := GetTenantService().createAndInsertTenant(username, username)
	if errte != nil {
		logrus.Errorf("create and insert new tenant error,  err is %v", errte)
		return USER_ERROR_REG, "", errte
	}

	role, errrole := GetRoleService().getRoleByName(userParam.RoleType)
	if errrole != nil {
		logrus.Errorf("get role error is %v", errrole)
		return ROLE_ERROR_GET, "", errrole
	}

	userId, err = p.createAndInsertUser(username, encryPassword, email, tenantId, role.ObjectId.Hex(), role.Rolename, company)
	if err != nil {
		logrus.Errorf("create and insert new user error,  err is %v", err)
		return USER_ERROR_REG, "", err
	}

	//start to send email
	body := p.replaceEmailBody(bodyTemplate, username, password)
	subject := subjectTemplate

	logrus.Infof("sending email to %s...  .userId %s", email, userId)

	err = p.sendConfigedEmail(email, string(subject), string(body), token)
	if err != nil {
		logrus.Warnf("fail to send email to %s,reason %v", email, err)
	}

	return "", userId, nil
}

func (p *UserService) replaceEmailBody(bodyTemplate []byte, userName string, password string) (body []byte) {
	//-1 means replace all
	// allendpoints := p.buildArray(endpoints)
	body = bytes.Replace(bodyTemplate, []byte("<USER_NAME>"), []byte(userName), -1)
	body = bytes.Replace(body, []byte("<ENDPOINT>"), []byte(password), -1)
	return body
}

func (p *UserService) sendConfigedEmail(to string, subject string, body string, token string) (err error) {
	Smtps, errq := GetSmtpInfo(token)
	if errq != nil {
		logrus.Errorf("query smtp err is %v", errq)
		return
	}

	if len(Smtps) <= 0 {
		logrus.Warn("does not find smtp server information!")
		err = errors.New("no smtp server! will not send email to user")
		return
	}
	Smtp := Smtps[0]

	emailHost := Smtp.Address
	emailUsername := Smtp.Name
	passwd, errd := common.Base64Decode([]byte(Smtp.PassWd))
	logrus.Infof("smtp password is %v", Smtp.PassWd)
	if errd != nil {
		logrus.Errorf("decode password err is %v", errd)
		return
	}
	emailPasswd := string(passwd)
	logrus.Infof("host is %v, username is %v, password is %v", emailHost, emailUsername, emailPasswd)

	if len(strings.TrimSpace(emailHost)) == 0 {
		return errors.New("get email host  error")
	}
	if len(strings.TrimSpace(emailUsername)) == 0 {
		return errors.New("get email username error")
	}
	if len(strings.TrimSpace(emailPasswd)) == 0 {
		return errors.New("get email password error")
	}

	go p.sendEmail(emailHost, emailUsername, emailPasswd, to, subject, body)
	return
}

//send email
func (p *UserService) sendEmail(host string, from string, password string, to string,
	subject string, body string) (err error) {

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	//port 25
	d := gomail.NewPlainDialer(host, 25, from, password)

	err = d.DialAndSend(m)

	if err != nil {
		logrus.Warnln("send email error %v", err)
	}
	return
}

func (p *UserService) GetUserByUserId(userId string) (user *entity.User, err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for getUseerById: ", userId)
		err = errors.New("invalid object id for getUserById")
		return nil, err
	}
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	user = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(user, queryStruct)

	if err != nil {
		logrus.Warnln("failed to get user by id %v", err)
		return
	}

	return
}

func (p *UserService) UserLogin(username string, password string) (errorCode string, login *entity.LoginResponse, err error) {
	currentAllUser, err := p.getAllUserByName(username)
	if err != nil {
		return "", nil, err
	}
	if len(currentAllUser) == 0 {
		return USER_ERROR_NOEXIST, nil, errors.New("user is not exist")
	}
	currentUser := currentAllUser[0]

	encryPassword := HashString(password)
	if !strings.EqualFold(encryPassword, currentUser.Password) {
		logrus.Errorln("invalid password!")
		return USER_ERROR_WRONGPW, nil, errors.New("Invalid password!")

	}

	tenant, erro := GetTenantService().GetTenantByTenantId(currentUser.TenantId)
	if erro != nil {
		logrus.Errorf("get user's tenant error %v", erro)
		return USER_ERROR_LOGIN, nil, erro
	}
	token, err := GetTokenService().checkAndGenerateToken(username, password, tenant.Tenantname, true)
	if err != nil {
		logrus.Errorf("failed to generate token, error is %s", err)
		return USER_ERROR_LOGIN, nil, err
	}

	var loginRes *entity.LoginResponse
	loginRes = new(entity.LoginResponse)
	loginRes.Id = token
	loginRes.UserId = currentUser.ObjectId.Hex()

	return "", loginRes, nil
}

func (p *UserService) UserUpdate(token string, userParam UserParam, userId string) (created bool, id string, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return false, userId, code, err
	}

	if authorized := GetAuthService().Authorize("update_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return false, userId, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	if !bson.IsObjectIdHex(userId) {
		logrus.Errorf("invalid user id format for user update %v", userId)
		return false, "", COMMON_ERROR_INVALIDATE, errors.New("Invalid object Id for user update")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	user := new(entity.User)
	err = dao.HandleQueryOne(user, queryStruct)
	if err != nil {
		logrus.Errorf("get user by id error %v", err)
		return false, "", USER_ERROR_UPDATE, err
	}

	if len(userParam.Company) >= 0 {
		user.Company = userParam.Company
	}
	if len(userParam.Email) > 0 {
		user.Email = userParam.Email
	}
	Token, errT := tokenService.GetTokenById(token)
	if errT != nil {
		logrus.Errorf("get token by id err is %v", errT)
		return
	}
	operationUser := Token.User.Id
	operationUserRole := Token.Role.Rolename
	
	changeRoletype := userParam.RoleType
	if changeRoletype != "sysadmin" && changeRoletype != "admin" {
		logrus.Errorf("dose not support role type")
		return
	}
	
	oldRoletypeId :=  user.RoleId
	oldrole, errRole := GetRoleService().getRoleByRoleId(oldRoletypeId)
	if errRole != nil {
		logrus.Errorf("get role by id error %v", errRole)
		return
	}
	oldroleType := oldrole.Rolename
	changeRole, errCh := GetRoleService().getRoleByName(changeRoletype)
	if errCh != nil {
		logrus.Errorf("get role by name error %v", errCh)
		return
	}
	
	if oldroleType != changeRoletype {
		if operationUserRole == "sysadmin" {
			if user.Username != "sysadmin" {
				logrus.Infof("start to change user role")
				if user.ObjectId.Hex() != operationUser {
					user.RoleId = changeRole.ObjectId.Hex()
					user.RoleName = changeRole.Rolename
				} else {
					err = errors.New("you can not change you own role")
					errorCode = USER_SRROR_CANNOT_CHANGE_OWN_ROLE
					return
				}
			} else {
				err = errors.New("can not change sysadmin user role")
				errorCode = USER_ERROR_CANNOT_CHANGE_SYSADMIN_USER_ROLE
				return
			}
		} else {
			err = errors.New("you have no auth to change role")
			errorCode = COMMON_ERROR_INVALIDATE
			return
		}
	}
	

	user.TimeUpdate = dao.GetCurrentTime()

	created, err = dao.HandleUpdateOne(user, queryStruct)
	return created, userId, "", nil
}

func (p *UserService) UserDelete(token string, userId string) (User entity.User, errorCode string, err error) {
	if !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid object id for UserDelete: ", userId)
		err = errors.New("invalid object id for UserDelete")
		return User, USER_ERROR_DELETE, err
	}

	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return User, code, err
	}

	if authorized := GetAuthService().Authorize("delete_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return User, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	user, err := p.GetUserById(userId)
	User = *user
	tenantid := user.TenantId
	newtoken, errC, errR := GetTokenService().TokenReGenerate(token, userId, tenantid)
	if errR != nil {
		logrus.Errorf("regenerate token is err is %v", errR)
		return User, errC, errR
	}

	clusters, errquery := GetClusterByUser(userId, newtoken.Id)
	if errquery != nil {
		logrus.Errorf("query cluster err is %v", errquery)
		return User, "", errors.New("query cluster is err")
	}
	if len(clusters) != 0 {
		logrus.Errorf("user has unterminated cluster")
		return User, USER_ERROR_EXISTCLUSTER, errors.New("Please terminated cluster first!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	err = dao.HandleDelete(p.userCollectionName, true, selector)
	if err != nil {
		logrus.Warnln("delete user error %v", err)
		return User, USER_ERROR_DELETE, err
	}

	err = GetTenantService().deleteTenantById(tenantid)
	if err != nil {
		logrus.Warnln("delete tenant error %v", err)
		return User, TENANT_ERROR_DELETE, err
	}

	return User, "", nil
}

func (p *UserService) UserChangePassword(token string, id string, password string, newpassword string, confirm_newpassword string) (User *entity.User, created bool, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return User, false, code, err
	}

	if authorized := GetAuthService().Authorize("change_password", token, id, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return User, false, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	user, err := p.GetUserByUserId(id)
	if err != nil {
		logrus.Errorln("user does exist %v", err)
		return User, false, COMMON_ERROR_INTERNAL, errors.New("User does not exist!")
	}
	User = user

	pwdEncry := HashString(password)
	if !strings.EqualFold(pwdEncry, user.Password) {
		logrus.Errorln("incorrect password!")
		return User, false, USER_ERROR_WRONGPW, errors.New("Incorrect password!")
	}

	if !strings.EqualFold(newpassword, confirm_newpassword) {
		logrus.Errorln("inconsistence new password!")
		return User, false, USER_ERROR_WRONGPW, errors.New("Inconsistent new password!")
	}

	newpasswordEncry := HashString(newpassword)
	user.Password = newpasswordEncry

	user.TimeUpdate = dao.GetCurrentTime()

	// userDoc := ConvertToBson(*user)
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(id)

	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	created, err = dao.HandleUpdateOne(user, queryStruct)
	if err != nil {
		logrus.Errorf("update user password error! %v", err)
		return User, created, USER_ERROR_UPDATE, err
	}

	return User, created, "", nil
}

func (p *UserService) UserList(token string, limit int, skip int, sort string) (ret []entity.User, count int, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, 0, code, err
	}

	query, err := GetAuthService().BuildQueryByAuth("list_users", token)
	if err != nil {
		logrus.Errorf("auth failed during query all user: %v", err)
		return nil, 0, USER_ERROR_GET, err
	}

	result := []entity.User{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       query,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort}
	count, err = dao.HandleQueryAll(&result, queryStruct)

	return result, count, "", err
}

func (p *UserService) UserDetail(token string, userId string) (ret interface{}, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, code, err
	}

	if authorized := GetAuthService().Authorize("get_user", token, userId, p.userCollectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return nil, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userId)

	ret = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(ret, queryStruct)
	logrus.Errorln(ret)
	return
}

func (p *UserService) UserDetailForCluster(token string, userId string) (currentUser *entity.User, err error) {
	tokenCheck := common.UTIL.Props.GetString("cluster.user.token", "")
	logrus.Infof("tokencheck is %v", tokenCheck)
	if token != tokenCheck {
		return nil, errors.New("token is illegal!")
	}
	user, errG := p.GetUserById(userId)
	if errG != nil {
		return nil,errG
	}
	return user, nil
}

func (p *UserService) createAndInsertUser(userName string, password string, email string, tenanId string, roleId string, roleName string, company string) (userId string, err error) {
	// var jsondocument interface{}
	currentUser, erro := p.getAllUserByName(userName)
	if erro != nil {
		logrus.Errorf("get all user by username err is %v", erro)
		return "", erro
	}
	if len(currentUser) != 0 {
		logrus.Infoln("user already exist! username:", userName)
		userId = currentUser[0].ObjectId.Hex()
		return
	}

	currentTime := dao.GetCurrentTime()
	user := new(entity.User)
	user.ObjectId = bson.NewObjectId()
	user.Username = userName
	user.Password = password
	user.TenantId = tenanId
	user.RoleId = roleId
	user.RoleName = roleName
	user.Email = email
	user.Company = company
	user.TimeCreate = currentTime
	user.TimeUpdate = currentTime

	err = dao.HandleInsert(p.userCollectionName, user)
	if err != nil {
		logrus.Warnln("create user error %v", err)
		return
	}
	userId = user.ObjectId.Hex()

	return
}

func (p *UserService) getAllUserByName(username string) (user []entity.User, err error) {
	query := strings.Join([]string{"{\"username\": \"", username, "\"}"}, "")

	selector := make(bson.M)
	err = json.Unmarshal([]byte(query), &selector)
	if err != nil {
		return
	}
	selector, err = mejson.Unmarshal(selector)
	if err != nil {
		return
	}

	user = []entity.User{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	_, err = dao.HandleQueryAll(&user, queryStruct)

	return
}

func (p *UserService) GetUserById(userid string) (currentUser *entity.User, err error) {
	validId := bson.IsObjectIdHex(userid)
	if !validId {
		return nil, errors.New("invalid object id!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(userid)

	currentUser = new(entity.User)
	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(currentUser, queryStruct)
	if err != nil {
		logrus.Infoln("user does not exist! %v", err)
		return nil, err
	}

	return
}

func createpassword(n int) string {
	var letters = []rune("123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().Unix())
	password := make([]rune, n)
	for i := range password {
		password[i] = letters[rand.Intn(len(letters))]
	}
	return string(password)
}

func (p *UserService) Change(username, activecode, newpassword string) (errorcode string, err error) {
	logrus.Infof("start to change passwd")

	Code, expire, err := parseCodeAndExpireTime(activecode)
	logrus.Infof("code is %v, expire is %v", Code, expire)
	if err != nil {
		logrus.Errorf("parse code err is %v", err)
		return USER_ERROR_PARSECODE, err
	}

	logrus.Infof("hashstring username is %v", HashString((username)))
	if Code != HashString(username) {
		logrus.Errorf("user is not legal")
		return USER_ERROR_LEGAL, errors.New("user is not legal")
	}

	currentTime := time.Now().Unix()
	expireTime, erre := strconv.ParseInt(expire, 10, 0)
	if erre != nil {
		logrus.Errorln("convert expire time to string error, expire time is :", expire)
		return COMMON_ERROR_INVALIDATE, erre
	}

	if currentTime >= expireTime {
		logrus.Errorln("expired active code!")
		return USER_ERROR_EXPIRE, errors.New("Active code is expired!")
	}

	user, _, errg := p.GetOnlyUserByName(username)
	if errg != nil {
		logrus.Errorf("get user err is %v", errg)
		return USER_ERROR_QUERYUSER, errg
	}

	newpasswordEncry := HashString(newpassword)
	logrus.Infof("newpassword is %v", newpasswordEncry)
	user.Password = newpasswordEncry
	user.TimeUpdate = dao.GetCurrentTime()

	selector := bson.M{}
	selector["_id"] = user.ObjectId
	logrus.Infof("start to update password in db")

	queryStruct := dao.QueryStruct{
		CollectionName: p.userCollectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	_, err = dao.HandleUpdateOne(user, queryStruct)
	if err != nil {
		logrus.Errorf("update user password error! %v", err)
		return USER_ERROR_UPDATE, err
	}

	return
}

func (p *UserService) GetOnlyUserByName(username string) (user entity.User, errcode string, err error) {
	logrus.Infof("start to get user by username")
	currentUser, erro := p.getAllUserByName(username)
	if erro != nil {
		logrus.Errorf("get all user by username err is %v", erro)
		return user, USER_ERROR_QUERYUSER, erro
	}
	if len(currentUser) == 0 {
		logrus.Infoln("user is not exist! username:", username)
		return user, USER_ERROR_NOEXIST, errors.New("user is not exist")
	}

	if len(currentUser) != 1 {
		logrus.Errorf("user is not only")
		return user, USER_ERROR_ISNOTONLY, errors.New("user is not only")
	}
	user = currentUser[0]
	return
}

func (p *UserService) Forget(username, ip string) (errorcode string, err error) {
	logrus.Infof("start to deal with forget passwd")
	logrus.Infof("start to query user is ixist or not")
	user, _, err := p.GetOnlyUserByName(username)
	if err != nil {
		logrus.Errorf("get user err is %v", err)
		return
	}

	email := user.Email
	logrus.Infof("start to send email to user")
	subject := "[Linker Cloud Platform] Account " + username + " Email Verification"
	code := buildCode(HashString(username))
	url := strings.Join([]string{"http://", ip, "/#/setPassword", "?username=", username, "&activecode=", code}, "")
	body := p.replaceEmailBody(emailBody, username, url)

	err = p.sendConfigedEmail(email, string(subject), string(body), "")
	if err != nil {
		logrus.Warnf("fail to send email to %s,reason %v", email, err)
		return EMAIL_ERROR_SEND, err
	}

	return
}

func buildCode(activeCode string) string {
	t := time.Now().Unix()
	t += int64(activeCode_expire)
	expireTime := strconv.FormatInt(t, 10)

	code := activeCode + "," + expireTime
	result, err := common.DesEncrypt([]byte(code), []byte(key))
	if err != nil {
		logrus.Warnln("encrypt code error %v", err)
		return code
	}

	return base64.StdEncoding.EncodeToString(result)
}

func parseCodeAndExpireTime(code string) (acviceCode string, expireTime string, err error) {
	if len(code) == 0 {
		logrus.Warnln("the active Code is null!")
		err = errors.New("Active code is null")
		return
	}

	//if exist "+"in code, it will because space after getting from url
	code = strings.Replace(code, " ", "+", -1)

	input, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		logrus.Warnln("decode active code error %v", err)
		err = errors.New("Invalid active code! ")
		return
	}
	// activeCode + "," + expireTime
	origData, err := common.DesDecrypt(input, []byte(key))
	if err != nil {
		logrus.Warnln("decrypt error %v", err)
		err = errors.New("invalid active code! ")
		return
	}

	result := string(origData)

	values := strings.Split(result, ",")
	if len(values) < 2 {
		logrus.Warnln("invalid parametr: ", values)
		err = errors.New("Active user failed: invalid active code")
		return
	}

	return values[0], values[1], nil
}
