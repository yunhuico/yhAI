package services

import (
	"errors"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	LOG_ERROR_CREATE               string = "E55001"
	LOG_ERROR_CREATE_CALL_USERMGMT string = "E55002"
	LOG_ERROR_CREATE_PARAM_EMPTY   string = "E55003"
	LOG_ERROR_QUERY                string = "E55004"

	LOG_OPERATE_TYPE_CREATE_CLUSTER string = "create_cluster"
	LOG_OPERATE_TYPE_DELETE_CLUSTER string = "delete_cluster"
	LOG_OPERATE_TYPE_ADD_HOSTS      string = "add_hosts"
	LOG_OPERATE_TYPE_DELETE_HOSTS   string = "delete_hosts"

	LOG_OPERATE_STATUS_START   string = "start"
	LOG_OPERATE_STATUS_SUCCESS string = "success"
	LOG_OPERATE_STATUS_FAIL    string = "fail"
)

var (
	logService *LogService = nil
	onceLog    sync.Once
)

type LogService struct {
	collectionName string
}

func GetLogService() *LogService {
	onceLog.Do(func() {
		logrus.Debugf("Once called from logService ......................................")
		logService = &LogService{"log"}
	})
	return logService
}

func (p *LogService) CreateByClusterNotifyLog(clusterNotify entity.NotifyCluster, clusterid string, x_auth_token string) (newLog *entity.LogMessage,
	errorCode string, err error) {
	if len(clusterNotify.ClusterName) == 0 || len(clusterNotify.UserName) == 0 {
		logrus.Errorf("username or clustername can not be empty")
		return newLog, LOG_ERROR_CREATE_PARAM_EMPTY, errors.New("username or clustername is empty")
	}

	id := clusterNotify.LogId
	logmessage, errorCode, err := p.QueryById(id, x_auth_token)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterNotify.ClusterName}).Errorf("query log err is %v", err)
		return
	}

	//get clusterId by clustername
	//	_, clusters, _, err := GetClusterService().QueryCluster(clusterNotify.ClusterName, "", "", "unterminated", 0, 0, "", x_auth_token)
	//	if err != nil {
	//		logrus.Warnf("can not found cluster with name %s, error is %v", clusterNotify.ClusterName, err)
	//	}

	//    var clusterId string
	//    if len(clusters) <=0 {
	//       logrus.Warnf("no cluster with name %s", clusterNotify.ClusterName)
	//    } else {
	//       clusterId = clusters[0].ObjectId.Hex()
	//    }

	var status string
	var comments string
	if clusterNotify.IsSuccess {
		status = LOG_OPERATE_STATUS_SUCCESS
	} else {
		status = LOG_OPERATE_STATUS_FAIL
		comments = clusterNotify.Comments
	}

	var selector = bson.M{}
	selector["_id"] = logmessage.ObjectId
	change := bson.M{"status": status, "clusterId": clusterid, "comments": comments, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterNotify.ClusterName}).Errorf("update log with objectId [%v] status to [%v] failed, error is %v", logmessage.ObjectId, status, err)
		return
	}
	return

}

func (p *LogService) CreateByLogParam(queryType string, clusterId string, clusterName string, operateType string, status string, x_auth_token string) (newLog *entity.LogMessage,
	errorCode string, err error) {

	logmessage := entity.LogMessage{}
	logmessage.OperateType = operateType
	logmessage.Status = status
	logmessage.QueryType = queryType

	//set clustername here, and set clusterId in notify method (no clusterId for cluster creation)
	if len(clusterId) > 0 {
		cluster, errorCode, err := GetClusterService().QueryById(clusterId, x_auth_token)
		if err != nil {
			logrus.Errorf("select  cluster by clsterId[%v]  error is [%v]", clusterId, err)
			return nil, errorCode, err
		}
		logmessage.ClusterName = cluster.Name
	} else if len(clusterName) > 0 {
		logmessage.ClusterName = clusterName
	} else {
		logrus.Errorf("log param error by clsterId[%v]  clusterName[%v] is empty!", clusterId, clusterName)
		err = errors.New("log param error by clsterId  clusterName[%v] is empty!")
		errorCode = LOG_ERROR_CREATE_PARAM_EMPTY
		return nil, errorCode, err
	}

	return p.Create(&logmessage, x_auth_token)

}
func (p *LogService) Create(log *entity.LogMessage, x_auth_token string) (newLog *entity.LogMessage,
	errorCode string, err error) {
	logrus.Infof("start to create log [%v]", log)

	//save log Necessary parameters
	if log.Status == "" || log.OperateType == "" {
		logrus.Errorf("log param error is empty")
		err = errors.New("log param error is empty!")
		errorCode = LOG_ERROR_CREATE_PARAM_EMPTY
		return nil, errorCode, err
	}

	var token *entity.Token
	if x_auth_token != "" {
		token, err = GetTokenById(x_auth_token)
		if err != nil {
			logrus.Errorf("get token by id error is %v", err)
			errorCode = LOG_ERROR_CREATE_CALL_USERMGMT
			return nil, errorCode, err
		}
	}

	// generate ObjectId
	log.ObjectId = bson.NewObjectId()
	if log.UserId == "" {
		log.UserId = token.User.Id
		log.TenantId = token.Tenant.Id
	}
	if log.Username == "" {
		log.Username = token.User.Username
	}
	// set created_time and updated_time
	log.TimeCreate = dao.GetCurrentTime()
	log.TimeUpdate = log.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, log)
	if err != nil {
		errorCode = LOG_ERROR_CREATE
		logrus.Errorf("create log [%v] to bson error is %v", log, err)
		return
	}

	newLog = log

	return
}

func (p *LogService) QueryLogs(clusterId string, username string, querytype string, userId string, skip int, limit int, sort string, token string) (total int, logs []entity.LogMessage,
	errorCode string, err error) {
	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validation failed for query logs [%v]", err)
		return
	}
	query := bson.M{}
	if len(userId) > 0 && bson.IsObjectIdHex(userId) {
		query["user_id"] = userId
	}
	
	if len(clusterId) > 0 {
		logrus.Infof("query log by clusterId, cluster id : %s", clusterId)
		query["clusterId"] = clusterId
	}
	if len(username) > 0 {
		query["userName"] = username
	}
	if len(querytype) > 0 {
		query["queryType"] = querytype
	}

	return p.queryByQuery(query, skip, limit, sort, token, false)
}

func (p *LogService) queryByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, logMessages []entity.LogMessage,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_logs", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	logMessages = []entity.LogMessage{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&logMessages, queryStruct)
	if err != nil {
		logrus.Errorf("query logMessages by query [%v] error is %v", selector, err)
		errorCode = LOG_ERROR_QUERY
		return
	}
	return
}

func (p *LogService) UpdateLogStatus(objectId string, status string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to update log by objectId [%v] status to %v", objectId, status)
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	log, _, errq := p.QueryById(objectId, x_auth_token)
	if errq != nil {
		logrus.Errorf("get log by objeceId [%v] failed, error is %v", objectId, errq)
		errorCode = LOG_ERROR_QUERY
		return errorCode, errq
	}
	if log.Status == status {
		logrus.Infof("this log [%v] is already in state [%v]", log, status)
		return
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	change := bson.M{"status": status, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
		return
	}
	return
}

func (p *LogService) QueryById(objectId string, x_auth_token string) (log entity.LogMessage, errorCode string, err error) {
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}

	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	log = entity.LogMessage{}
	err = dao.HandleQueryOne(&log, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query log [objectId=%v] error is %v", objectId, err)
		errorCode = CLUSTER_ERROR_QUERY
		return
	}
	return
}
