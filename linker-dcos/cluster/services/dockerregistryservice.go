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
	DOCKER_REGISTRY_ERROR_EMPTY_NAME string = "E57000"
	DOCKER_REGISTRY_ERROR_EXIST_NAME string = "E57001"

	DOCKER_REGISTRY_ERROR_EMPTY_VALUE string = "E57002"

	DOCKER_REGISTRY_ERROR_CALL_USERMGMT string = "E57003"

	DOCKER_REGISTRY_ERROR_QUERY string = "E57004"

	DOCKER_REGISTRY_ERROR_DELETE              string = "E57005"
	DOCKER_REGISTRY_ERROR_USER_SELF           string = "E57006"
	DOCKER_REGISTRY_ERROR_DELETE_EXISTCLUSTER string = "E57007"
	DOCKER_REGISTRY_ERROR_USED                string = "E57008"
	DOCKER_REGISTRY_ERROR_BAD_VALUE           string = "E57009"
)

var (
	dockerRegistryService *DockerRegistryService = nil
	onceDockerRegistry    sync.Once
)

type DockerRegistryService struct {
	collectionName string
}

func GetDockerRegistryService() *DockerRegistryService {
	onceDockerRegistry.Do(func() {
		logrus.Debugf("Once called from DockerRegistryService ......................................")
		dockerRegistryService = &DockerRegistryService{"dockerregistry"}
	})
	return dockerRegistryService
}

func (p *DockerRegistryService) Save(registry entity.DockerRegistry, x_auth_token string) (newRegistry *entity.DockerRegistry,
	errorCode string, err error) {
	logrus.Infof("start to save docker registry [%v]", registry)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_docker_registry", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("save docker registry [%v] error is %v", registry, err)
		return
	}

	//check registry name
	if len(registry.Name) == 0 {
		return nil, DOCKER_REGISTRY_ERROR_EMPTY_NAME, errors.New("registry name cann't be null .")
	}

	//check registry value
	if len(registry.Registry) == 0 {
		return nil, DOCKER_REGISTRY_ERROR_EMPTY_VALUE, errors.New("registry endpoint cann't be null .")
	}

	//do not allow protocol like "http://", "https://"
	if strings.Contains(registry.Registry, "//") {
		return nil, DOCKER_REGISTRY_ERROR_BAD_VALUE, errors.New("registry endpoint contains //")
	}

	ok, errorCode, err := p.IsRegistryNameExist(registry.Name, registry.UserId)
	if ok {
		return nil, DOCKER_REGISTRY_ERROR_EXIST_NAME, errors.New("Registry name exists .")
	}

	return p.SaveDockerRegistry(registry, x_auth_token)
}

func (p *DockerRegistryService) IsRegistryUsed(registryName string, userId string) (flag bool, errorCode string, err error) {
	query := bson.M{}
	querymatch := make(bson.M)
	queryvalue := make(bson.M)
	queryOr := make(bson.M)

	sQuery1, sQuery2, sQuery3 := make(bson.M), make(bson.M), make(bson.M)
	sQuery1["status"] = CLUSTER_STATUS_RUNNING
	sQuery2["status"] = CLUSTER_STATUS_INSTALLING
	sQuery3["status"] = CLUSTER_STATUS_MODIFYING

	queryOr["$or"] = []bson.M{sQuery1, sQuery2, sQuery3}
	querymatch["$elemMatch"] = queryvalue

	queryvalue["name"] = registryName
	queryvalue["user_id"] = userId
	query["docker_registries"] = querymatch

	queryAnd := make(bson.M)
	queryAnd["$and"] = []bson.M{queryOr, query}

	clusters := []entity.Cluster{}
	queryStruct := dao.QueryStruct{"cluster", queryAnd, 0, 0, "...."}
	total, err := dao.HandleQueryAll(&clusters, queryStruct)
	if err != nil {
		logrus.Errorf("query docker registry by name and userid [%v]  [%v] error is %v", query, userId, err)
		errorCode = DOCKER_REGISTRY_ERROR_QUERY
		return true, errorCode, err
	}

	if total > 0 {
		return true, "", nil
	}

	return false, "", nil
}

func (p *DockerRegistryService) IsRegistryNameExist(registryName string, userId string) (flag bool, errorCode string, err error) {

	query := bson.M{}

	query["name"] = registryName
	query["user_id"] = userId

	n, _, errorCode, err := p.queryDockerRegistryByName(query, 0, 0, "", "", true)
	if err != nil {
		return false, errorCode, err
	}

	if n > 0 {
		//name  exist
		return true, "", nil
	}
	return false, "", nil
}

func (p *DockerRegistryService) queryDockerRegistryByName(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, registries []entity.DockerRegistry,
	errorCode string, err error) {

	authQuery := bson.M{}

	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_docker_registry", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	registries = []entity.DockerRegistry{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&registries, queryStruct)
	logrus.Debugf("total is %v", total)
	if err != nil {
		logrus.Errorf("query docker registry by name [%v] error is %v", query, err)
		errorCode = DOCKER_REGISTRY_ERROR_QUERY
		return
	}
	return
}

func (p *DockerRegistryService) SaveDockerRegistry(registry entity.DockerRegistry, x_auth_token string) (newRegistry *entity.DockerRegistry,
	errorCode string, err error) {

	// generate ObjectId
	registry.ObjectId = bson.NewObjectId()

	userId := registry.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		logrus.Errorf("save docker registry [%v] error is %v", registry, err)
		return
	}

	user, err := GetUserById(userId, x_auth_token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = DOCKER_REGISTRY_ERROR_CALL_USERMGMT
		return nil, errorCode, err
	}

	registry.UserId = user.ObjectId.Hex()
	registry.TenantId = user.TenantId

	// set created_time and updated_time
	registry.TimeCreate = dao.GetCurrentTime()

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, registry)
	if err != nil {
		errorCode = CLUSTER_ERROR_CALL_MONGODB
		logrus.Errorf("save docker registry [%v] to bson error is %v", registry, err)
		return
	}

	newRegistry = &registry

	return
}

//query docker registry
//filter by registry name
//filter by user id
func (p *DockerRegistryService) QueryDockerRegistries(name string, userId string, skip int,
	limit int, sort string, x_auth_token string) (total int, pubkeys []entity.DockerRegistry,
	errorCode string, err error) {

	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	query := bson.M{}
	if len(strings.TrimSpace(name)) > 0 {
		query["name"] = name
	}
	if len(strings.TrimSpace(userId)) > 0 {
		query["user_id"] = userId
	}

	return p.queryDockerRegistryByName(query, skip, limit, sort, x_auth_token, false)

}

func (p *DockerRegistryService) QueryById(objectId string, x_auth_token string) (registry entity.DockerRegistry,
	errorCode string, err error) {

	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	// do authorize first
	if authorized := GetAuthService().Authorize("get_docker_registry", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get pubkey with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	registry = entity.DockerRegistry{}
	err = dao.HandleQueryOne(&registry, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query docker registry [objectId=%v] error is %v", objectId, err)
		errorCode = DOCKER_REGISTRY_ERROR_QUERY
	}
	return
}

func (p *DockerRegistryService) DeleteById(objectId string, x_auth_token string) (registry entity.DockerRegistry, errorCode string, err error) {
	logrus.Infof("start to delete Docker Registry with id [%v]", objectId)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("delete_docker_registry", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting pubkey with id [%v] , error is %v", objectId, err)
		return registry, errorCode, err
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("Invalid docker registry id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return registry, errorCode, err
	}
	registryQ, _, errQ := p.QueryById(objectId, x_auth_token)
	if errQ != nil {
		logrus.Errorf("query registry err is %v", errQ)
	} else {
		registry = registryQ
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		errorCode = DOCKER_REGISTRY_ERROR_DELETE
		logrus.Errorf("delete Docker registry failure with id [%v] , error is %v", objectId, err)
		return registry, errorCode, err
	}
	return
}

func (p *DockerRegistryService) GetRegistryInfo(registry []entity.DockerRegistry, x_auth_token string) (registryinfos []entity.DockerRegistry) {
	logrus.Infof("start to get registryinfo")
	registryinfos = []entity.DockerRegistry{}
	if len(registry) <= 0 {
		logrus.Infof("registry lenght is 0!")
		return registryinfos
	}
	for _, oneregistry := range registry {
		registryinfo := convertToRegistryInfo(oneregistry, x_auth_token)
		registryinfos = append(registryinfos, registryinfo)
	}

	return registryinfos

}

func convertToRegistryInfo(registry entity.DockerRegistry, x_auth_token string) (registryinfo entity.DockerRegistry) {
	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["user_id"] = registry.UserId
	query2["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["$and"] = []bson.M{query1, query2}
	total, clusters, _, errQ := GetClusterService().queryByQuery(query, 0, 0, "", x_auth_token, false)
	if errQ != nil {
		logrus.Errorf("query cluster err is %v", errQ)
		return
	}
	if total == 0 {
		logrus.Errorf("there is no cluster by query")
		//	return
	}
	regs := []entity.DockerRegistry{}

	for _, cluster := range clusters {
		for _, reg := range cluster.DockerRegistries {
			regs = append(regs, reg)
		}
	}
	isuse := false

	for _, regt := range regs {
		id := regt.ObjectId.Hex()
		if id == registry.ObjectId.Hex() {
			isuse = true
		}
	}
	registryinfo = entity.DockerRegistry{
		ObjectId:         registry.ObjectId,
		Name:             registry.Name,
		Registry:         registry.Registry,
		Secure:           registry.Secure,
		CAText:           registry.CAText,
		Username:         registry.Username,
		Password:         registry.Password,
		UserId:           registry.UserId,
		TenantId:         registry.TenantId,
		IsUse:            isuse,
		TimeCreate:       registry.TimeCreate,
		IsSystemRegistry: registry.IsSystemRegistry,
	}
	return registryinfo

}
