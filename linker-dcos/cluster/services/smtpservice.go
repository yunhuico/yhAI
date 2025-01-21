package services

import (
	"errors"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/cluster/common"
)

const (
	SMTP_ERROR_EMPTY_NAME    string = "E53000"
	SMTP_ERROR_EMPTY_ADDRESS string = "E53001"
	SMTP_ERROR_QUERY         string = "E53002"
	SMTP_ERROR_EXIST_NAME    string = "E53003"
	SMTP_ERROR_EXIST_ADDRESS string = "E53004"
	SMTP_ERROR_DELETE        string = "E53005"
	SMTP_ERROR_UPDATE        string = "E53006"
)

var (
	smtpService *SmtpService = nil
	onceSmtp    sync.Once
)

type SmtpService struct {
	collectionName string
}

func GetSmtpService() *SmtpService {
	onceSmtp.Do(func() {
		logrus.Debugf("Once called from stmpService ......................................")
		smtpService = &SmtpService{"smtp"}
	})
	return smtpService

}

func (p *SmtpService) Save(smtp entity.Smtp, x_auth_token string) (newSmtp *entity.Smtp,
	errorCode string, err error) {
	logrus.Infof("start to save smtp [%v]", smtp)

	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_smtp", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("save smtp [%v] error is %v", smtp, err)
		return
	}

	//check smtp name
	if len(smtp.Name) == 0 {
		return nil, SMTP_ERROR_EMPTY_NAME, errors.New("Smtp name cann't be null .")
	}

	//check smtp address
	if len(smtp.Address) == 0 {
		return nil, SMTP_ERROR_EMPTY_ADDRESS, errors.New("Smtp address cann't be null .")
	}

	// ok, errorCode, err := p.isSmtpNameExit(smtp.Name, x_auth_token)
	// if ok {
	// 	return nil, SMTP_ERROR_EXIST_NAME, errors.New("Smtp name exists .")
	// }
	// result, errorCode, err := p.isSmtpAddressExit(smtp.Address)
	// if result {
	// 	return nil, SMTP_ERROR_EXIST_ADDRESS, errors.New("Smtp address exists .")
	// }

	smtp.ObjectId = bson.NewObjectId()
	if smtp.PassWd != "" {
		passwdbyte := common.Base64Encode([]byte(smtp.PassWd))
		smtp.PassWd = string(passwdbyte)
	}

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, smtp)
	if err != nil {
		errorCode = CLUSTER_ERROR_CALL_MONGODB
		logrus.Errorf("save smtp [%v] to bson error is %v", smtp, err)
		return
	}

	newSmtp = &smtp
	return
}

// func (p *SmtpService) isSmtpNameExit(stmpName string) (flag bool, errorCode string, err error) {

// 	query := bson.M{}

// 	query["name"] = stmpName

// 	n, _, errorCode, err := p.querySmtpByQuery(query, 0, 0, "", "", true)
// 	if err != nil {
// 		return false, errorCode, err
// 	}
// 	if n > 0 {
// 		//name  exist
// 		return true, "", nil
// 	}
// 	return false, "", nil
// }
// func (p *SmtpService) isSmtpAddressExit(address string) (flag bool, errorCode string, err error) {

// 	query := bson.M{}

// 	query["address"] = address

// 	n, _, errorCode, err := p.querySmtpByQuery(query, 0, 0, "", "", true)
// 	if err != nil {
// 		return false, errorCode, err
// 	}
// 	if n > 0 {
// 		//address  exist
// 		return true, "", nil
// 	}
// 	return false, "", nil
// }

func (p *SmtpService) querySmtpByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, smtps []entity.Smtp,
	errorCode string, err error) {

	authQuery := bson.M{}

	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_smtp", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	smtps = []entity.Smtp{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&smtps, queryStruct)
	if err != nil {
		logrus.Errorf("query smtps by query [%v] error is %v", query, err)
		errorCode = SMTP_ERROR_QUERY
		return
	}
	return
}

func (p *SmtpService) QueryStmp(name string, address string, skip int,
	limit int, sort string, x_auth_token string) (total int, smtps []entity.Smtp,
	errorCode string, err error) {

	query := bson.M{}
	if len(strings.TrimSpace(name)) > 0 {
		query["name"] = name
	}
	if len(strings.TrimSpace(address)) > 0 {
		query["address"] = address
	}

	authQuery := bson.M{}
	selector := generateQueryWithAuth(query, authQuery)
	smtps = []entity.Smtp{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&smtps, queryStruct)
	if err != nil {
		logrus.Errorf("query smtps by query [%v] error is %v", query, err)
		errorCode = SMTP_ERROR_QUERY
		return
	}

	return
}

func (p *SmtpService) QueryById(objectId string, x_auth_token string) (smtp entity.Smtp,
	errorCode string, err error) {

	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed for provider creation [%v]", err)
		return
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	// do authorize first
	if authorized := GetAuthService().Authorize("get_smtp", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get smtp with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	smtp = entity.Smtp{}
	err = dao.HandleQueryOne(&smtp, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query smtp [objectId=%v] error is %v", objectId, err)
		errorCode = SMTP_ERROR_QUERY
	}
	return
}

func (p *SmtpService) DeleteById(objectId string, x_auth_token string) (smtp entity.Smtp, errorCode string, err error) {
	logrus.Infof("start to delete smtp with id [%v]", objectId)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}
	// do authorize first
	if authorized := GetAuthService().Authorize("delete_smtp", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting smtp with id [%v] , error is %v", objectId, err)
		return smtp, errorCode, err
	}
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("Invalid stmp id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return smtp, errorCode, err
	}
	smtpQ, _, errQ := p.QueryById(objectId, x_auth_token)
	if errQ != nil {
		logrus.Errorf("query smtp err is %v", errQ)
	} else {
		smtp = smtpQ
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		errorCode = SMTP_ERROR_DELETE
		logrus.Errorf("delete smtp failure with id [%v] , error is %v", objectId, err)
		return smtp, errorCode, err
	}
	return
}
func (p *SmtpService) SmtpUpdate(token string, newsmtp entity.Smtp, objectId string) (created bool, id string, errorCode string, err error) {
	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed [%v]", err)
		return
	}

	if authorized := GetAuthService().Authorize("update_smtp", token, id, p.collectionName); !authorized {
		logrus.Errorln("required opertion is not allowed!")
		return false, objectId, COMMON_ERROR_UNAUTHORIZED, errors.New("Required opertion is not authorized!")
	}

	if !bson.IsObjectIdHex(objectId) {
		logrus.Errorf("invalid smtp id format for smtp update %v", objectId)
		return false, "", COMMON_ERROR_INVALIDATE, errors.New("Invalid object Id for smtp update")
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	smtp := new(entity.Smtp)
	err = dao.HandleQueryOne(smtp, queryStruct)
	if err != nil {
		logrus.Errorf("get smtp by id error %v", err)
		return false, "", SMTP_ERROR_UPDATE, err
	}

	query := bson.M{}

	query["name"] = newsmtp.Name

	_, dbSmtps, errorCode, err := p.querySmtpByQuery(query, 0, 0, "", "", true)
	if err != nil {
		logrus.Errorf("update_get smtp by name error %v", err)
		return false, "", SMTP_ERROR_UPDATE, err
	}
	for _, v := range dbSmtps {
		if v.ObjectId != bson.ObjectIdHex(objectId) {
			return false, "", SMTP_ERROR_EXIST_NAME, errors.New("Smtp name exists .")
		}
	}

	queryAddress := bson.M{}

	queryAddress["address"] = newsmtp.Address

	_, dbSmtpsAddress, errorCode, err := p.querySmtpByQuery(queryAddress, 0, 0, "", "", true)
	if err != nil {
		logrus.Errorf("update_get smtp by address error %v", err)
		return false, "", SMTP_ERROR_UPDATE, err
	}
	for _, v := range dbSmtpsAddress {
		if v.ObjectId != bson.ObjectIdHex(objectId) {
			return false, "", SMTP_ERROR_EXIST_ADDRESS, errors.New("Smtp address exists .")
		}
	}
	if len(newsmtp.Address) > 0 {
		smtp.Address = newsmtp.Address
	}
	if len(newsmtp.Name) > 0 {
		smtp.Name = newsmtp.Name
	}
	if len(newsmtp.PassWd) > 0 {
		passwdbyte := common.Base64Encode([]byte(newsmtp.PassWd))
		smtp.PassWd = string(passwdbyte)
	}

	created, err = dao.HandleUpdateOne(smtp, queryStruct)
	return created, objectId, "", nil
}
