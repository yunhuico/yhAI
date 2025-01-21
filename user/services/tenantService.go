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
)

var tenantService *TenantService = nil
var tenantOnce sync.Once
var TENANT_ERROR_GET = "E10040"
var TENANT_ERROR_CREATE = "E10041"
var TENANT_ERROR_DELETE = "E10042"

type TenantService struct {
	collectionName string
}

func GetTenantService() *TenantService {
	tenantOnce.Do(func() {
		tenantService = &TenantService{"tenant"}
	})

	return tenantService
}

func (p *TenantService) TenantList(token string, limit int, skip int, sort string) (ret []entity.Tenant, count int, errorCode string, err error) {
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, 0, code, err
	}

	query, err := GetAuthService().BuildQueryByAuth("list_tenants", token)
	if err != nil {
		logrus.Errorf("auth failed during query all tenant: %v", err)
		return nil, 0, TENANT_ERROR_GET, err
	}

	ret = []entity.Tenant{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       query,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort}
	count, err = dao.HandleQueryAll(&ret, queryStruct)

	return
}

func (p *TenantService) GetTenantByTenantId(tenantId string) (tenant *entity.Tenant, err error) {
	if !bson.IsObjectIdHex(tenantId) {
		logrus.Errorln("invalid object id for getTenantById: ", tenantId)
		err = errors.New("invalid object id for getTenantById")
		return
	}
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(tenantId)

	tenant = new(entity.Tenant)
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(tenant, queryStruct)
	if err != nil {
		logrus.Warnln("failed to get tenant by id %v", err)
		return
	}

	return
}

func (p *TenantService) TenantDetail(token string, tenantId string) (ret interface{}, errorCode string, err error) {
	if !bson.IsObjectIdHex(tenantId) {
		logrus.Errorln("invalid object id for getTenantDetail: ", tenantId)
		err = errors.New("invalid object id for getTenantDetail")
		return nil, TENANT_ERROR_CREATE, err
	}
	code, err := GetTokenService().TokenValidate(token)
	if err != nil {
		return nil, code, err
	}

	if authorized := GetAuthService().Authorize("get_tenant", token, tenantId, p.collectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return nil, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(tenantId)

	ret = entity.Tenant{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(&ret, queryStruct)
	return
}

type TenantId struct {
	Id string `json:"id"`
}

func (p *TenantService) GetTenantId(token string, userId string) (ret TenantId, errorCode string, err error) {
	if len(userId) <= 0 || !bson.IsObjectIdHex(userId) {
		logrus.Errorln("invalid user id :", userId)
		return ret, COMMON_ERROR_INVALIDATE, errors.New("invalid userId")
	}

	user, err := GetUserService().GetUserByUserId(userId)
	if err != nil {
		logrus.Errorln("get user by id error %v", err)
		return ret, COMMON_ERROR_INTERNAL, err
	}

	tenant, err := p.GetTenantByTenantId(user.TenantId)
	if err != nil {
		return ret, COMMON_ERROR_INTERNAL, err
	}

	ret = TenantId{Id: tenant.ObjectId.Hex()}

	return
}

// CreateAndInsertTenant creat the tenant and insert to collection according
// by tenantname and desc.
func (p *TenantService) createAndInsertTenant(tenantName string, desc string) (tenantId string, err error) {
	tenant, erro := p.getTenantByName(tenantName)
	if erro == nil {
		logrus.Infoln("tenant already exist! tenantname: ", tenantName)
		tenantId = tenant.ObjectId.Hex()
		return
	}

	currentTime := dao.GetCurrentTime()
	objectId := bson.NewObjectId()
	newTenant := entity.Tenant{
		ObjectId:    objectId,
		Tenantname:  tenantName,
		Description: desc,
		TimeCreate:  currentTime,
		TimeUpdate:  currentTime,
	}

	err = dao.HandleInsert(p.collectionName, &newTenant)
	if err != nil {
		logrus.Errorf("create tenant error %v", err)
		return
	}
	tenantId = newTenant.ObjectId.Hex()
	return

}

// GetTenantByName return the tenant by the given tenant name.
func (p *TenantService) getTenantByName(tenantname string) (tenant *entity.Tenant, err error) {
	selector := make(bson.M)
	selector["tenantname"] = tenantname

	tenant = new(entity.Tenant)
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	err = dao.HandleQueryOne(tenant, queryStruct)
	if err != nil {
		logrus.Warnln("get tenant by name error %v", err)
		return
	}

	return
}

func (p *TenantService) deleteTenantByName(tenantname string) (err error) {
	query := strings.Join([]string{"{\"tenantname\": \"", tenantname, "\"}"}, "")

	selector := make(bson.M)
	err = json.Unmarshal([]byte(query), &selector)
	if err != nil {
		return
	}
	selector, err = mejson.Unmarshal(selector)
	if err != nil {
		return
	}

	return dao.HandleDelete(p.collectionName, true, selector)
}

func (p *TenantService) deleteTenantById(tenantid string) (err error) {
	if !bson.IsObjectIdHex(tenantid) {
		err = errors.New("invalide ObjectId.")
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(tenantid)

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		logrus.Errorf("delete tenant [tenantid=%v] error is %v", tenantid, err)
	}
	return

}
