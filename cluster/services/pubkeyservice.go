package services

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"

	command "linkernetworks.com/dcos-backend/common/common"
)

const (
	PUBKEY_ERROR_EMPTY_NAME string = "E52000"
	PUBKEY_ERROR_EXIST_NAME string = "E52001"

	PUBKEY_ERROR_EMPTY_VALUE string = "E52002"

	PUBKEY_ERROR_CALL_USERMGMT string = "E52003"

	PUBKEY_ERROR_QUERY string = "E52004"

	PUBKEY_ERROR_DELETE              string = "E52005"
	PUBKEY_ERROR_USER_SELF           string = "E52006"
	PUBKEY_ERROR_DELETE_EXISTCLUSTER        = "E52007"

	PUBKEY_STORAGEPATH_USER_SELF string = "/tmp/linker/pubkey/"
)

var (
	pubkeyService *PubKeyService = nil
	oncePubKey    sync.Once
)

type PubKeyService struct {
	collectionName string
}

func GetPubKeyService() *PubKeyService {
	oncePubKey.Do(func() {
		logrus.Debugf("Once called from pubkeyService ......................................")
		pubkeyService = &PubKeyService{"pubkey"}
	})
	return pubkeyService

}
func (p *PubKeyService) Save(pubkey entity.PubKey, x_auth_token string) (newPubKey *entity.PubKey,
	errorCode string, err error) {
	logrus.Infof("start to save pubkey [%v]", pubkey)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_pubkey", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("save pubkey [%v] error is %v", pubkey, err)
		return
	}

	//check pubkey name
	if len(pubkey.Name) == 0 {
		return nil, PUBKEY_ERROR_EMPTY_NAME, errors.New("PubKey name cann't be null .")
	}

	//check pubkey value
	if len(pubkey.PubKeyValue) == 0 {
		return nil, PUBKEY_ERROR_EMPTY_VALUE, errors.New("PubKey value cann't be null .")
	}

	//p.isPubKeyNameExit(pubkey.Name)
	ok, errorCode, err := p.isPubKeyNameExit(pubkey.Name, pubkey.UserId)
	if ok {
		return nil, PUBKEY_ERROR_EXIST_NAME, errors.New("PubKey name exists .")
	}

	return p.SavePubKey(pubkey, x_auth_token)
}

func (p *PubKeyService) CreateAndSave(userId string, pubkey entity.PubKey, x_auth_token string) (privateKeyFile string,
	errorCode string, err error) {

	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed for create and save pubkey [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_pubkey", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("save pubkey [%v] error is %v", pubkey, err)
		return
	}

	//check pubkey name
	if len(pubkey.Name) == 0 {
		return "", PUBKEY_ERROR_EMPTY_NAME, errors.New("PubKey name cann't be null .")
	}
	//p.isPubKeyNameExit(pubkey.Name)
	ok, _, err := p.isPubKeyNameExit(pubkey.Name, pubkey.UserId)
	if ok {
		return "", PUBKEY_ERROR_EXIST_NAME, errors.New("PubKey name exists .")
	}

	storagePath := PUBKEY_STORAGEPATH_USER_SELF + userId + ""
	logrus.Infof("storagePath %v", storagePath)
	_, err = os.Stat(storagePath)
	if err != nil {
		logrus.Infof("start to save pubkey %v", err)
	}
	if err == nil {
		logrus.Infof("os.state")
		err = os.RemoveAll(storagePath)
		logrus.Infof("os.removeall")
		if err != nil {
			logrus.Infof("os.removeall wrong")
			return "", PUBKEY_ERROR_USER_SELF, err
		}
	}
	logrus.Infof("os.makdir")
	err = os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		logrus.Infof("os.mkdir wrong")
		return "", PUBKEY_ERROR_USER_SELF, err
	}
	privateKeyFile = storagePath + "/" + "user_key" + ""
	logrus.Infof("privateKeyFile %v", privateKeyFile)

	commandStr := "ssh-keygen " + "-f " + privateKeyFile + " -N ''"
	logrus.Infof("cmmandStr")
	logrus.Infof("Executing new pubkey command: %s", commandStr)
	_, _, err = command.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("Call ssh-keygen failed , err is %v", err)
		return "", PUBKEY_ERROR_USER_SELF, err
	}

	pubKeyFile := storagePath + "/" + "user_key.pub" + ""

	b, err := ioutil.ReadFile(pubKeyFile)
	if err != nil {
		logrus.Infof("read file err %v", err)
		return "", PUBKEY_ERROR_USER_SELF, err
	}

	pubkey.PubKeyValue = string(b)

	logrus.Infof("start to save pubkey [%v]", pubkey)

	//check pubkey value
	if len(pubkey.PubKeyValue) == 0 {
		return "", PUBKEY_ERROR_EMPTY_VALUE, errors.New("PubKey value cann't be null .")
	}

	_, _, err = p.SavePubKey(pubkey, x_auth_token)
	if err != nil {
		return "", PUBKEY_ERROR_USER_SELF, err
	}
	return
}

func (p *PubKeyService) isPubKeyNameExit(pubkeyName string, userId string) (flag bool, errorCode string, err error) {

	query := bson.M{}

	query["name"] = pubkeyName
	query["user_id"] = userId

	n, _, errorCode, err := p.queryPubKeyByQuery(query, 0, 0, "", "", true)
	if err != nil {
		return false, errorCode, err
	}
	if n > 0 {
		//name  exist
		return true, "", nil
	}
	return false, "", nil
}

func (p *PubKeyService) queryPubKeyByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, pubkeys []entity.PubKey,
	errorCode string, err error) {

	authQuery := bson.M{}

	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_pubkey", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	pubkeys = []entity.PubKey{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&pubkeys, queryStruct)
	if err != nil {
		logrus.Errorf("query pubkeys by query [%v] error is %v", query, err)
		errorCode = PUBKEY_ERROR_QUERY
		return
	}
	return
}

func (p *PubKeyService) SavePubKey(pubkey entity.PubKey, x_auth_token string) (newPubKey *entity.PubKey,
	errorCode string, err error) {

	// generate ObjectId
	pubkey.ObjectId = bson.NewObjectId()

	userId := pubkey.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		logrus.Errorf("save pubkey [%v] error is %v", pubkey, err)
		return
	}

	user, err := GetUserById(userId, x_auth_token)
	if err != nil {
		logrus.Errorf("get user by id err is %v", err)
		errorCode = PUBKEY_ERROR_CALL_USERMGMT
		return nil, errorCode, err
	}

	pubkey.TenantId = user.TenantId

	pubkey.Owner = user.Username

	// set created_time and updated_time
	pubkey.TimeCreate = dao.GetCurrentTime()
	pubkey.TimeUpdate = pubkey.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, pubkey)
	if err != nil {
		errorCode = CLUSTER_ERROR_CALL_MONGODB
		logrus.Errorf("save pubkey [%v] to bson error is %v", pubkey, err)
		return
	}

	newPubKey = &pubkey
	return
}

//query unterminated pubkeys
//filter by pubkey name
//filter by user id
func (p *PubKeyService) QueryPubKey(name string, userId string, skip int,
	limit int, sort string, x_auth_token string) (total int, pubkeys []entity.PubKey,
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

	return p.queryPubKeyByQuery(query, skip, limit, sort, x_auth_token, false)

}

func (p *PubKeyService) QueryById(objectId string, x_auth_token string) (pubkey entity.PubKey,
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
	if authorized := GetAuthService().Authorize("get_pubkey", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get pubkey with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	pubkey = entity.PubKey{}
	err = dao.HandleQueryOne(&pubkey, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query pubkey [objectId=%v] error is %v", objectId, err)
		errorCode = PUBKEY_ERROR_QUERY
	}
	return
}

func (p *PubKeyService) DeleteById(objectId string, x_auth_token string) (pubkey entity.PubKey, errorCode string, err error) {
	logrus.Infof("start to delete PubKey with id [%v]", objectId)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("delete_pubkey", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting pubkey with id [%v] , error is %v", objectId, err)
		return pubkey, errorCode, err
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("Invalid pubkey id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return pubkey, errorCode, err
	}
	pubkeyQ, _, errQ := p.QueryById(objectId, x_auth_token)
	if errQ != nil {
		logrus.Errorf("query pubkey err is %v", errQ)
	} else {
		pubkey = pubkeyQ
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["pubkeyId"] = objectId
	query2["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	query["$and"] = []bson.M{query1, query2}
	total, _, errCodeQ, errQ := GetClusterService().queryByQuery(query, 0, 0, "", x_auth_token, false)
	if errQ != nil {
		logrus.Errorf("query cluster err is %v", errQ)
		return pubkey, errCodeQ, errQ
	}
	if total != 0 {
		logrus.Infof("cluster total is %v", total)
		logrus.Errorf("some cluster using this pubkey cannot delete this provider")
		return pubkey, PUBKEY_ERROR_DELETE_EXISTCLUSTER, errors.New("some cluster using this pubkey,cannot delete this pubkey")
	}

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		errorCode = PUBKEY_ERROR_DELETE
		logrus.Errorf("delete PubKey failure with id [%v] , error is %v", objectId, err)
		return pubkey, errorCode, err
	}
	return
}

func (p *PubKeyService) GetPubkeyInfo(pubkeyss []entity.PubKey, token string) (pubkeyinfos []entity.PubKey) {
	logrus.Infof("start to get pubkeyinfo")
	pubkeyinfos = []entity.PubKey{}
	if len(pubkeyss) <= 0 {
		logrus.Infof("pubkey lenght is 0!")
		return pubkeyinfos
	}
	for _, onepubkey := range pubkeyss {
		pubkeyinfo := convertToPubkeyInfo(onepubkey, token)
		pubkeyinfos = append(pubkeyinfos, pubkeyinfo)
	}

	return pubkeyinfos
}

func convertToPubkeyInfo(pubkey entity.PubKey, token string) (pubkeyinfo entity.PubKey) {
	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query1["pubkeyId"] = pubkey.ObjectId.Hex()
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

	pubkeyinfo = entity.PubKey{
		ObjectId:    pubkey.ObjectId,
		PubKeyValue: pubkey.PubKeyValue,
		Name:        pubkey.Name,
		Owner:       pubkey.Owner,
		UserId:      pubkey.UserId,
		TenantId:    pubkey.TenantId,
		IsUse:       isuse,
		TimeCreate:  pubkey.TimeCreate,
		TimeUpdate:  pubkey.TimeUpdate,
	}

	return pubkeyinfo

}
