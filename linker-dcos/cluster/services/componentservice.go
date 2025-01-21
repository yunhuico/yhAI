package services

import (
	"errors"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)


var (
	componentService *ComponentService = nil
	onceComponent    sync.Once
	
	COMPONENT_ERROR_CREATE = "E54100"
	COMPONENT_ERROR_QUERY = "E54101"
	COMPONENT_ERROR_UPDATE = "E54102"
)

type ComponentService struct {
	collectionName string
}

func GetComponentService() *ComponentService {
	onceComponent.Do(func() {
		logrus.Debugf("Once called from clusterService ......................................")
		componentService = &ComponentService{"component"}
	})
	return componentService
}

func (p *ComponentService) Create (component entity.Components, x_auth_token string) (newcomponent entity.Components, errorCode string, err error) {
	logrus.Infof("start to create new component")
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}
	
	if authorized := GetAuthService().Authorize("create_component", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("save docker component [%v] error is %v", component, err)
		return
	}
	
	component.ObjectId = bson.NewObjectId()

	token, err := GetTokenById(x_auth_token)
	if err != nil {
		errorCode = HOST_ERROR_CREATE
		logrus.Errorf("get token failed when create component [%v], error is %v", component, err)
		return
	}

	// set token_id and user_id from token
	component.TenantId = token.Tenant.Id
	component.UserId = token.User.Id

	// set created_time and updated_time
	component.TimeCreate = dao.GetCurrentTime()
	component.TimeUpdate = dao.GetCurrentTime()

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, component)
	if err != nil {
		errorCode = COMPONENT_ERROR_CREATE
		logrus.Errorf("insert component [%v] to db error is %v", component, err)
		return
	}

	newcomponent = component
	return
}

func (p *ComponentService) QuerycomponentByClusterid(clusterId string, skip int, limit int, x_auth_token string) (total int,
	component []entity.Components, errorCode string, err error) {
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return total, component, errorCode, err
	}	
	
	query := bson.M{}
	if len(strings.TrimSpace(clusterId)) > 0 {
		query["clusterId"] = clusterId
	}
	return p.queryByQuery(query, skip, limit, x_auth_token, false)	
}	

func (p *ComponentService) queryByQuery(query bson.M, skip int, limit int,
	x_auth_token string, skipAuth bool) (total int, component []entity.Components,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_component", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	component = []entity.Components{}
	// fix : "...." sort by time_create
	queryStruct := dao.QueryStruct{p.collectionName, selector, skip, limit, "time_create"}
	total, err = dao.HandleQueryAll(&component, queryStruct)
	if err != nil {
		logrus.Errorf("query hosts by query [%v] error is %v", query, err)
		errorCode = COMPONENT_ERROR_QUERY
	}
	return
}

func (p *ComponentService) UpdateById(objectId string, component entity.Components, x_auth_token string) (created bool,
	errorCode string, err error) {
	clustername := component.ClusterName
	upLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	upLog.Infof("start to update component [%v]", component)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		upLog.Errorf("token validate err is %v", err)
		return created, errorCode, err
	}
	// do authorize first
	if authorized := GetAuthService().Authorize("update_component", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		upLog.Errorf("update component with objectId [%v] error is %v", objectId, err)
		return
	}

	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// FIXING
	//	hostquery, _, _  := p.QueryById(objectId, x_auth_token)
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	component.ObjectId = bson.ObjectIdHex(objectId)
	component.TimeUpdate = dao.GetCurrentTime()

	upLog.Infof("start to change component")
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, &component)
	//	created, err = dao.HandleUpdateOne(&host, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		upLog.Errorf("update component [%v] error is %v", component, err)
		errorCode = COMPONENT_ERROR_UPDATE
	}
	created = true
	return
}

func (p *ComponentService) DeleteByClusterid(clusterid string, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to delete component by clusterid")
	errCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return errCode, err
	}
	if authorized := GetAuthService().Authorize("delete_component", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting pubkey with id [%v] , error is %v", clusterid, err)
		return errCode, err
	}
	if !bson.IsObjectIdHex(clusterid) {
		err = errors.New("Invalid cluster id.")
		errCode = COMMON_ERROR_INVALIDATE
		return errCode, err
	}
	var selector = bson.M{}
	selector["clusterId"] = clusterid
	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		errCode = PUBKEY_ERROR_DELETE
		logrus.Errorf("delete component failure with cluster id [%v] , error is %v", clusterid, err)
		return errCode, err
	}
	return
}
