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

const (
	PROVIDER_ERROR_CREATE              string = "E51000"
	PROVIDER_ERROR_UPDATE              string = "E51001"
	PROVIDER_ERROR_DELETE              string = "E51002"
	PROVIDER_ERROR_QUERY               string = "E51003"
	PROVIDER_ERROR_EMPTY_NAME          string = "E51004"
	PROVIDER_NAME_CONFILCT             string = "E51005"
	PROVIDER_ERROR_DELETE_EXISTCLUSTER string = "E51006"

	PROVIDER_EC2_TYPE       = "amazonec2"
	PROVIDER_GOOGLE_TYPE    = "google"
	PROVIDER_OPENSTACK_TYPE = "openstack"
	PROVIDER_GENERIC_TYPE   = "generic"
)

var (
	providerService *ProviderService = nil
	onceProvider    sync.Once
)

type ProviderService struct {
	collectionName string
}

func GetProviderService() *ProviderService {
	onceProvider.Do(func() {
		logrus.Debugf("Once called from providerService ......................................")
		providerService = &ProviderService{"provider"}
	})
	return providerService
}

func (p *ProviderService) Create(provider entity.IaaSProvider, token string) (newProvider entity.IaaSProvider, errorCode string, err error) {
	logrus.Infof("create provider [%v]", provider)
	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for provider creation [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_provider", token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("create provider [%v] error is %v", provider, err)
		return
	}

	userId := provider.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		logrus.Errorf("create provider [%v] error is %v", provider, err)
		return
	}

	user, err := GetUserById(userId, token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = COMMON_ERROR_INTERNAL
		return newProvider, errorCode, err
	}

	provider.ObjectId = bson.NewObjectId()
	provider.TenantId = user.TenantId
	// set created_time and updated_time
	provider.TimeCreate = dao.GetCurrentTime()
	provider.TimeUpdate = provider.TimeCreate

	err = dao.HandleInsert(p.collectionName, provider)
	if err != nil {
		errorCode = PROVIDER_ERROR_CREATE
		logrus.Errorf("create provider [%v] to bson error is %v", provider, err)
		return
	}

	newProvider = provider

	return

}

func (p *ProviderService) CheckProviderName(provider_name string, token string) (conflict bool, errorCode string, err error) {
	if len(strings.TrimSpace(provider_name)) == 0 {
		return false, PROVIDER_ERROR_EMPTY_NAME, errors.New("invalid provider_name")
	}

	logrus.Infof("provider_name: %s", provider_name)

	return p.isProviderNameConflict(provider_name, token)
}
func (p *ProviderService) isProviderNameConflict(providerName string, token string) (conflict bool, errorCode string, err error) {
	selector := bson.M{}
	selector["name"] = providerName
	n, _, _, err := p.queryByQuery(selector, 0, 0, "", token, false)
	if err != nil {
		logrus.Errorf("query db for providers failed, %v", err)
		return false, PROVIDER_ERROR_QUERY, err
	}
	//found conflict
	if n > 0 {
		return false, PROVIDER_NAME_CONFILCT, errors.New("provider name already exists")
	}
	return true, "", nil
}
func (p *ProviderService) QueryProvider(providerType string, userId string, skip int, limit int, sort string, token string) (total int, providers []entity.IaaSProvider,
	errorCode string, err error) {
	query := bson.M{}
	if len(providerType) > 0 {
		query["type"] = providerType
	}

	if len(userId) > 0 && bson.IsObjectIdHex(userId) {
		query["user_id"] = userId
	} else {
		logrus.Warnf("not a valid object id for query provider operation! userId: %s", userId)
	}

	return p.queryByQuery(query, skip, limit, sort, token, false)
}

// func (p *ProviderService) QueryProviderByUserId(userId string, skip int, limit int, sort string, token string) (total int, providers []entity.IaaSProvider,
// 	errorCode string, err error) {
// 	query := bson.M{}
// 	if len(userId) > 0 {
// 		logrus.Infof("userId  is : %s", userId)
// 		query["user_id"] = userId
// 	}
// 	return p.queryByQuery(query, skip, limit, sort, token, false)
// }
func (p *ProviderService) QueryById(objectId string, token string) (provider entity.IaaSProvider, errorCode string, err error) {
	logrus.Infof("query provider by id[%s]", objectId)
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for provider get [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("get_provider", token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get provider with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	provider = entity.IaaSProvider{}
	err = dao.HandleQueryOne(&provider, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query provider [objectId=%v] error is %v", objectId, err)
		errorCode = PROVIDER_ERROR_QUERY
	}
	return

}

func (p *ProviderService) queryByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, providers []entity.IaaSProvider,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_provider", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	providers = []entity.IaaSProvider{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&providers, queryStruct)
	if err != nil {
		logrus.Errorf("query providers by query [%v] error is %v", query, err)
		errorCode = PROVIDER_ERROR_QUERY
		return
	}
	return
}

func (p *ProviderService) DeleteById(objectId string, token string) (provider entity.IaaSProvider, errorCode string, err error) {
	logrus.Infof("start to delete provider with id [%v]", objectId)

	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for provider creation [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("delete_provider", token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting provider with objectId [%v] error is %v", objectId, err)
		return
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	providerQ, _, errQ := p.QueryById(objectId, token)
	if errQ != nil {
		logrus.Errorf("query provicer err is %v", errQ)
	} else {
		provider = providerQ
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["providerId"] = objectId
	query2["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["$and"] = []bson.M{query1, query2}
	total, _, errCodeQ, errQ := GetClusterService().queryByQuery(query, 0, 0, "", token, false)
	if errQ != nil {
		logrus.Errorf("query cluster err is %v", errQ)
		return provider, errCodeQ, errQ
	}
	if total != 0 {
		logrus.Infof("cluster total is %v", total)
		logrus.Errorf("some cluster using this provide cannot delete this provider")
		return provider, PROVIDER_ERROR_DELETE_EXISTCLUSTER, errors.New("some cluster using this provider,cannot delete this provider")
	}

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		errorCode = PROVIDER_ERROR_DELETE
		logrus.Errorf("delete provider failure with id [%v] , error is %v", objectId, err)
		return provider, errorCode, err
	}
	return
}

func (p *ProviderService) ProviderUpdate(token string, newProvider entity.IaaSProvider, objectId string) (created bool, id string, errorCode string, err error) {
	logrus.Infof("update provider: %v", newProvider)
	code, err := TokenValidation(token)
	if err != nil {
		return false, objectId, code, err
	}

	if authorized := GetAuthService().Authorize("update_provider", token, objectId, p.collectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return false, objectId, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["providerId"] = objectId
	query2["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["$and"] = []bson.M{query1, query2}
	total, _, errCodeQ, errQ := GetClusterService().queryByQuery(query, 0, 0, "", token, false)
	if errQ != nil {
		logrus.Errorf("query cluster err is %v", errQ)
		return created, objectId, errCodeQ, errQ
	}
	if total != 0 {
		logrus.Infof("cluster total is %v", total)
		logrus.Errorf("some cluster using this provide cannot update this provider")
		return created, objectId, PROVIDER_ERROR_DELETE_EXISTCLUSTER, errors.New("some cluster using this provider,cannot update this provider")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	// provider.TimeUpdate = dao.GetCurrentTime()
	created, err = dao.HandleUpdateOne(newProvider, queryStruct)
	if err != nil {
		logrus.Errorf("update provider by id error %v", err)
		return false, "", PROVIDER_ERROR_UPDATE, err
	}
	return created, objectId, "", nil
}

func (p *ProviderService) GetProviderInfo(providers []entity.IaaSProvider, token string) (providerinfos []entity.ProviderListInfo) {
	logrus.Infof("start to get providerinfo")
	providerinfos = []entity.ProviderListInfo{}
	if len(providers) <= 0 {
		logrus.Infof("provider lenght is 0!")
		return providerinfos
	}
	for _, oneprovider := range providers {
		providerinfo := convertToProviderInfo(oneprovider, token)
		providerinfos = append(providerinfos, providerinfo)
	}

	return providerinfos
}

func convertToProviderInfo(provider entity.IaaSProvider, token string) (providerinfo entity.ProviderListInfo) {
	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["providerId"] = provider.ObjectId.Hex()
	query2["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["$and"] = []bson.M{query1, query2}
	total, _, _, errQ := GetClusterService().queryByQuery(query, 0, 0, "", token, false)
	if errQ != nil {
		logrus.Errorf("query cluster err is %v", errQ)
		return
	}
	isuse := false
	if total != 0 {
		isuse = true
	}

	providerinfo = entity.ProviderListInfo{
		ObjectId:      provider.ObjectId,
		Name:          provider.Name,
		Type:          provider.Type,
		SshUser:       provider.SshUser,
		OpenstackInfo: provider.OpenstackInfo,
		AwsEC2Info:    provider.AwsEC2Info,
		GoogleInfo:    provider.GoogleInfo,
		UserId:        provider.UserId,
		TenantId:      provider.TenantId,
		TimeCreate:    provider.TimeCreate,
		TimeUpdate:    provider.TimeUpdate,
		IsUse:         isuse,
	}
	return providerinfo
}
