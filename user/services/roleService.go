package services

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	//	"linkernetworks.com/linker_usermgmt/common"
)

var roleService *RoleService = nil
var roleOnce sync.Once
var ROLE_ERROR_CREATE = "E10030"
var ROLE_ERROR_GET = "E10031"

type RoleService struct {
	collectionName string
}

func GetRoleService() *RoleService {
	roleOnce.Do(func() {
		roleService = &RoleService{"role"}
	})

	return roleService
}

func (p *RoleService) RoleList(token string) (ret []entity.Role, count int, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, 0, code, err
	}

	query, err := GetAuthService().BuildQueryByAuth("list_roles", token)
	if err != nil {
		logrus.Errorf("auth failed during query all role: %v", err)
		return nil, 0, ROLE_ERROR_GET, err
	}

	// ret = interface{}
	ret = []entity.Role{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       query,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	count, err = dao.HandleQueryAll(&ret, queryStruct)

	return
}

// CreateAndInsertRole creat and insert the role items according to the given
// rolename and desc.
func (p *RoleService) createAndInsertRole(roleName string, desc string) (roleId string, err error) {
	role := &entity.Role{}
	role, err = p.getRoleByName(roleName)
	if err == nil {
		logrus.Infoln("role already exist! roleName: ", roleName)
		roleId = role.ObjectId.Hex()
		return
	}

	currentTime := dao.GetCurrentTime()
	objectId := bson.NewObjectId()
	role = &entity.Role{
		ObjectId:    objectId,
		Rolename:    roleName,
		Description: desc,
		TimeCreate:  currentTime,
		TimeUpdate:  currentTime,
	}

	err = dao.HandleInsert(p.collectionName, role)
	if err != nil {
		logrus.Warnln("create role error %v", err)
		return
	}
	roleId = role.ObjectId.Hex()
	return

}

// GetRoleByName return the role by the role name.
func (p *RoleService) getRoleByName(rolename string) (role *entity.Role, err error) {
	query := strings.Join([]string{"{\"rolename\": \"", rolename, "\"}"}, "")

	selector := make(bson.M)
	err = json.Unmarshal([]byte(query), &selector)
	if err != nil {
		return
	}
	selector, err = mejson.Unmarshal(selector)
	if err != nil {
		return
	}

	role = new(entity.Role)
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(role, queryStruct)

	return
}

func (p *RoleService) getRoleByRoleId(roleid string) (role *entity.Role, err error) {
	if !bson.IsObjectIdHex(roleid) {
		err = errors.New("invalide ObjectId.")
		return
	}
	selector := make(bson.M)
	selector["_id"] = bson.ObjectIdHex(roleid)

	role = new(entity.Role)
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(role, queryStruct)
	if err != nil {
		logrus.Warnln("get role by name error %v", err)
		return
	}

	return
}
