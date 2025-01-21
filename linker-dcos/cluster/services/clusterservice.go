package services

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"linkernetworks.com/dcos-backend/cluster/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var f_date = "2006-01-02"
var f_datetime = "2006-01-02 15:04:05"
var f_rfc3339 = "2006-01-02T15:04:05Z07:00"

var MasterImages = []string{"mongodb", "adminrouter", "universeregistry", "universenginx", "dnsserver", "cosmos", "marathon", "mesosmaster", "exhibitor", "metricscollector"}
var SlaveImages = []string{"cadvisormonitor", "mesosslave"}
var OneInMasterImage = []string{"webconsole", "dcosclient", "alertmanager", "prometheus"}
var AllImage = []string{"genresolvconf"}

const (
	CLUSTER_STATUS_TERMINATED  = "TERMINATED"
	CLUSTER_STATUS_RUNNING     = "RUNNING"
	CLUSTER_STATUS_FAILED      = "FAILED"
	CLUSTER_STATUS_TERMINATING = "TERMINATING"
	CLUSTER_STATUS_INSTALLING  = "INSTALLING"
	CLUSTER_STATUS_MODIFYING   = "MODIFYING"

	CLUSTER_CATEGORY_COMPACT string = "compact"
	CLUSTER_CATEGORY_HA      string = "ha"

	MINMUM_NODE_NUMBER_HA      int = 5
	MINMUM_NODE_NUMBER_COMPACT int = 2

	CLUSTER_ERROR_CREATE     string = "E50000"
	CLUSTER_ERROR_UPDATE     string = "E50001"
	CLUSTER_ERROR_DELETE     string = "E50002"
	CLUSTER_ERROR_NAME_EXIST string = "E50003"
	CLUSTER_ERROR_QUERY      string = "E50004"

	CLUSTER_ERROR_INVALID_NUMBER     string = "E50010"
	CLUSTER_ERROR_INVALID_NAME       string = "E50011"
	CLUSTER_ERROR_INVALID_TYPE       string = "E50012"
	CLUSTER_ERROR_CALL_USERMGMT      string = "E50013"
	CLUSTER_ERROR_CALL_DEPLOYMENT    string = "E50014"
	CLUSTER_ERROR_CALL_MONGODB       string = "E50015"
	CLUSTER_ERROR_INVALID_STATUS     string = "E50016"
	CLUSTER_ERROR_DELETE_NOT_ALLOWED string = "E50017"
	CLUSTER_ERROR_DELETE_NODE_NUM    string = "E50018"
	CLUSTER_ERROR_IPEXIST            string = "E50019"
	GETCOMPONENT_HEALTHCHECK_ERROR   string = "E50020"
)

var (
	clusterService *ClusterService = nil
	onceCluster    sync.Once
)

type ClusterService struct {
	collectionName string
}

func GetClusterService() *ClusterService {
	onceCluster.Do(func() {
		logrus.Debugf("Once called from clusterService ......................................")
		clusterService = &ClusterService{"cluster"}

		clusterService.initialize()
	})
	return clusterService

}

func (p *ClusterService) initialize() {
	logrus.Infoln("initialize cluster check and change cluster status process")
	interval := common.UTIL.Props.GetString("cluster_check_interval", "86400")
	if len(interval) <= 0 {
		interval = "86400"
	}
	exec := common.UTIL.Props.GetString("cluster_check_time", "03:00:00")
	if len(exec) <= 0 {
		exec = "03:00:00"
	}

	formatdate := time.Now().Format(f_date)
	newexec := formatdate + " " + exec
	execTime, err := time.ParseInLocation(f_datetime, newexec, time.Now().Location())
	if err != nil {
		logrus.Warnln("failed to parse exec check time: ", newexec)
		execTime, _ = time.ParseInLocation(f_datetime, formatdate+" 03:00:00", time.Now().Location())
	}

	intervalInt, err := strconv.ParseInt(interval, 10, 64)
	if err != nil {
		logrus.Warnln("failed to parse intervalTime: ", interval)
		intervalInt, _ = strconv.ParseInt("259200", 10, 64)
	}

	waitTime := GetWaitTime(execTime)

	go p.startClusterTimer(waitTime, intervalInt)
}

func (p *ClusterService) startClusterTimer(waitTime int64, intervalTime int64) {
	logrus.Infoln("waiting for checking cluster process to start...")

	// waitTime = 10
	t := time.NewTimer(time.Second * time.Duration(waitTime))
	<-t.C

	logrus.Infoln("begin to do cluster check process")
	p.checkAndChangeCluster()

	logrus.Infoln("set ticker for interval check")
	ticker := time.NewTicker(time.Second * time.Duration(intervalTime))
	go p.run(ticker)

}

func (p *ClusterService) run(ticker *time.Ticker) {
	for t := range ticker.C {
		logrus.Debugln("ticker ticked: ", t)
		p.checkAndChangeCluster()
	}
}

func (p *ClusterService) checkAndChangeCluster() {
	logrus.Infoln("start to check cluster")

	selector := bson.M{}
	selector1 := bson.M{}
	selector2 := bson.M{}
	selector3 := bson.M{}
	selector1["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	selector2["status"] = bson.M{"$ne": CLUSTER_STATUS_FAILED}
	selector3["status"] = bson.M{"$ne": CLUSTER_STATUS_RUNNING}
	selector["$and"] = []bson.M{selector1, selector2, selector3}

	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	clusters := []entity.Cluster{}
	_, err := dao.HandleQueryAll(&clusters, queryStruct)
	if err != nil {
		logrus.Errorln("query user by state error %v", err)
	}

	currenttime := time.Now()
	for i := 0; i < len(clusters); i++ {
		record := clusters[i]
		updateTime := record.TimeUpdate
		statusNow := record.Status
		id := record.ObjectId.Hex()

		dur, _ := time.ParseDuration("+3h")
		expireTime := updateTime.Add(dur)
		if expireTime.Before(currenttime) {
			logrus.Debugln("change expired cluster id:", id)
			if statusNow == CLUSTER_STATUS_TERMINATING || statusNow == CLUSTER_STATUS_INSTALLING {
				err = p.changeClusterAndHostStatus(id)
				if err != nil {
					logrus.Errorf("change cluster and host status err is %v", err)
				}
			} else if statusNow == CLUSTER_STATUS_MODIFYING {
				err = p.changeModifyingClusterandHostStatus(id)
				if err != nil {
					logrus.Errorf("change cluster and host status err is %v", err)
				}

			}
		}
	}
}

func (p *ClusterService) changeModifyingClusterandHostStatus(id string) (err error) {
	query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
	query["clusterId"] = id
	query1["status"] = HOST_STATUS_INSTALLING
	query2["status"] = HOST_STATUS_TERMINATING
	query["$or"] = []bson.M{query1, query2}
	hosts, errq := GetHostService().query(query)
	if errq != nil {
		logrus.Errorf("query host err is %v", err)
		return errq
	}

	for _, host := range hosts {
		var selector1 = bson.M{}
		selector1["_id"] = host.ObjectId
		change := bson.M{"status": HOST_STATUS_FAILED, "time_update": dao.GetCurrentTime()}
		err = dao.HandleUpdateByQueryPartial("hosts", selector1, change)

		if err != nil {
			logrus.Errorf("change host status err is %v", err)
			continue
		}
	}

	//reset cluster running instances
	selector := bson.M{}
	selector["clusterId"] = id
	selector["status"] = HOST_STATUS_RUNNING
	hostss, errQ := GetHostService().query(selector)
	if errQ != nil {
		logrus.Errorf("query host err is %v", err)
		return errQ
	}
	instances := len(hostss)

	var selectorC = bson.M{}
	selectorC["_id"] = bson.ObjectIdHex(id)
	change := bson.M{"status": CLUSTER_STATUS_RUNNING, "instances": instances, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selectorC, change)
	if err != nil {
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", id, CLUSTER_STATUS_RUNNING, err)
		return
	}

	return
}

func (p *ClusterService) changeClusterAndHostStatus(objId string) (err error) {
	logrus.Debugf("change cluster %s status to falied", objId)
	var selectorC = bson.M{}
	selectorC["_id"] = bson.ObjectIdHex(objId)
	change := bson.M{"status": CLUSTER_STATUS_FAILED, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selectorC, change)
	if err != nil {
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objId, CLUSTER_STATUS_FAILED, err)
	}

	logrus.Debugf("change cluster %s hosts to failed", objId)
	var selector1 = bson.M{}
	selector1["clusterId"] = objId
	selector1["status"] = bson.M{"$ne": HOST_STATUS_TERMINATED}
	change = bson.M{"status": HOST_STATUS_FAILED, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial("hosts", selector1, change)
	if err != nil {
		logrus.Errorf("change host status err is %v", err)
		return
	}

	return

}

func (p *ClusterService) CheckClusterName(userId string, clusterName string, x_auth_token string) (errorCode string, err error) {
	checkLog := logrus.WithFields(logrus.Fields{"clustername": clusterName})
	checkLog.Infof("checking clustername [%s] for user with id [%s]", clusterName, userId)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		checkLog.Errorf("token validate err is %v", err)
		return
	}
	// authorization
	// if authorized := GetAuthService().Authorize("list_cluster", x_auth_token, "", p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("check cluster name auth failure [%v]", err)
	// 	return
	// }

	//check userId(must be a objectId at least)
	if len(userId) > 0 && !bson.IsObjectIdHex(userId) {
		checkLog.Errorf("invalid userid [%s],not a object id\n", clusterName)
		return COMMON_ERROR_INVALIDATE, errors.New("Invalid userid,not a object id")
	}

	ok, errorCode, err := p.isClusterNameUnique(userId, x_auth_token, clusterName)
	if err != nil {
		return errorCode, err
	}
	if !ok {
		checkLog.Errorf("clustername [%s] already exist for user with id [%s]\n", clusterName, userId)
		return CLUSTER_ERROR_NAME_EXIST, errors.New("Invalid clustername,conflict")
	}
	return
}

//check if name of a user's clusters is conflict
func (p *ClusterService) isClusterNameUnique(userId string, token string, clusterName string) (ok bool, errorCode string, err error) {
	//check cluster name
	//name of someone's unterminated clusters should be unique
	query := bson.M{}
	query["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	if len(userId) > 0 {
		query["user_id"] = userId
	}

	query["name"] = clusterName
	n, _, errorCode, err := p.queryByQuery(query, 0, 0, "", token, false)
	if err != nil {
		return false, errorCode, err
	}
	if n > 0 {
		//name already exist
		return false, "", nil
	}
	return true, "", nil
}

func (p *ClusterService) Create(createRequest entity.CreateRequest, logobjid string, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {
	createLog := logrus.WithFields(logrus.Fields{"clustername": createRequest.Name})
	createLog.Infof("start to create cluster [%v]", createRequest)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		createLog.Errorf("token validate err is %v", err)
		return nil, errorCode, err
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_cluster", x_auth_token, "", p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		createLog.Errorf("create cluster [%v] error is %v", createRequest, err)
		return
	}

	//check cluster name
	if !IsClusterNameValid(createRequest.Name) {
		return nil, CLUSTER_ERROR_INVALID_NAME, errors.New("Invalid cluster name.")
	}
	if len(createRequest.Name) > 15 {
		return nil, CLUSTER_ERROR_INVALID_NAME, errors.New("clustername must be less than 15")
	}

	//check userId(must be a objectId at least)
	if !bson.IsObjectIdHex(createRequest.UserId) {
		createLog.Errorf("invalid userid [%s],not a object id\n", createRequest.UserId)
		return nil, COMMON_ERROR_INVALIDATE, errors.New("Invalid userid,not a object id")
	}

	//check if cluster name is unique
	ok, errorCode, err := p.isClusterNameUnique(createRequest.UserId, x_auth_token, createRequest.Name)
	if err != nil {
		return nil, errorCode, err
	}
	if !ok {
		createLog.Errorf("clustername [%s] already exist for user with id [%s]\n", createRequest.Name, createRequest.UserId)
		return nil, CLUSTER_ERROR_NAME_EXIST, errors.New("Conflict clustername")
	}

	//check cluster category
	categoryValue := createRequest.CreateCategory
	if !strings.EqualFold(categoryValue, CLUSTER_CATEGORY_COMPACT) && !strings.EqualFold(categoryValue, CLUSTER_CATEGORY_HA) {
		createLog.Warnf("cluster category is invalid %s, will use defaut value %s", categoryValue, CLUSTER_CATEGORY_COMPACT)
		createRequest.CreateCategory = CLUSTER_CATEGORY_COMPACT
	}

	//check instances count
	err = p.checkInstanceCount(createRequest)
	if err != nil {
		return
	}

	if createRequest.Type == "customized" {
		if createRequest.MasterCount != len(createRequest.MasterNodes) || createRequest.SharedCount != len(createRequest.SharedNodes) || createRequest.PureSlaveCount != len(createRequest.PureSlaveNodes) {
			createLog.Errorf("request node number is inconsistence with add nodes number!")
			return nil, COMMON_ERROR_INVALIDATE, errors.New("request node number is inconsistence with add nodes number!")
		}
		Nodes := []entity.Node{}
		for _, masterNode := range createRequest.MasterNodes {
			Nodes = append(Nodes, masterNode)
		}
		for _, sharedNode := range createRequest.SharedNodes {
			Nodes = append(Nodes, sharedNode)
		}
		if len(createRequest.PureSlaveNodes) > 0 {
			for _, pureslaveNode := range createRequest.PureSlaveNodes {
				Nodes = append(Nodes, pureslaveNode)
			}
		}
		createLog.Infof("Nodes is %s ", Nodes)

		isexist, err := p.isIpExist(Nodes, x_auth_token)
		createLog.Infof("isexist is %v", isexist)
		if err != nil {
			createLog.Errorf("query ip isexist err is %v", err)
			return nil, HOST_ERROR_QUERY, errors.New("query ip isexist err")
		}

		if isexist {
			createLog.Errorf("the ip of this node has exist")
			return nil, CLUSTER_ERROR_IPEXIST, errors.New("ip is exist!")
		}

	}

	return p.CreateUserCluster(createRequest, logobjid, x_auth_token)

}

func (p *ClusterService) isIpExist(nodes []entity.Node, token string) (isexist bool, err error) {
	var nodeIps []string
	for _, node := range nodes {
		nodeIps = append(nodeIps, node.IP)
	}

	for i := 0; i < len(nodeIps)-1; i++ {
		for j := i + 1; j < len(nodeIps); j++ {
			if nodeIps[i] == nodeIps[j] {
				return true, nil
			}
		}
	}

	_, hosts, _, err := GetHostService().QueryHosts("", 0, 0, "unterminated", token)
	if err != nil {
		logrus.Errorf("query host err is %v", err)
		return false, err
	}

	var ips []string
	if len(hosts) != 0 {
		for _, host := range hosts {
			ips = append(ips, host.IP)
		}
	}

	for _, nodeip := range nodeIps {
		for _, ip := range ips {
			if nodeip == ip {
				return true, nil
			}
		}
	}
	return false, nil
}

func (p *ClusterService) checkInstanceCount(request entity.CreateRequest) error {
	chLog := logrus.WithFields(logrus.Fields{"clustername": request.Name})
	category := request.CreateCategory
	instance := request.MasterCount + request.SharedCount + request.PureSlaveCount

	if request.MasterCount <= 0 {
		return errors.New("cluster mast have one master node at least")
	}
	if request.SharedCount <= 0 {
		return errors.New("cluster must have one shared node at least")
	}
	if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
		if instance < MINMUM_NODE_NUMBER_COMPACT {
			chLog.Errorf("compact cluster's minmum node can not be less than %d", MINMUM_NODE_NUMBER_COMPACT)
			return errors.New("Invalid cluster instance number for compact cluster!")
		}
	} else if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
		if instance < MINMUM_NODE_NUMBER_HA {
			chLog.Errorf("compact cluster's minmum node can not be less than %d", MINMUM_NODE_NUMBER_HA)
			return errors.New("Invalid cluster instance number for ha cluster!")
		}
	} else {
		chLog.Errorf("not supported cluster category %s", category)
		return errors.New("Invliad cluster category!")
	}

	return nil
}

func (p *ClusterService) CreateUserCluster(createRequest entity.CreateRequest, logobjid string, x_auth_token string) (newCluster *entity.Cluster,
	errorCode string, err error) {
	userLog := logrus.WithFields(logrus.Fields{"clustername": createRequest.Name})
	Count := createRequest.MasterCount + createRequest.SharedCount + createRequest.PureSlaveCount
	cluster := entity.Cluster{
		Name: createRequest.Name, Owner: createRequest.Owner, Instances: Count,
		PubKeyId: createRequest.PubKeyId, ProviderId: createRequest.ProviderId, Details: createRequest.Details,
		UserId: createRequest.UserId, Type: createRequest.Type, CreateCategory: createRequest.CreateCategory,
		DockerRegistries: createRequest.DockerRegistries}

	// generate ObjectId
	cluster.ObjectId = bson.NewObjectId()

	userId := cluster.UserId
	if len(userId) == 0 {
		err = errors.New("user_id not provided")
		errorCode = COMMON_ERROR_INVALIDATE
		userLog.Errorf("create cluster [%v] error is %v", cluster, err)
		return
	}

	user, err := GetUserById(userId, x_auth_token)
	if err != nil {
		userLog.Errorf("get user by id err is %v", err)
		errorCode = CLUSTER_ERROR_CALL_USERMGMT
		return nil, errorCode, err
	}
	cluster.TenantId = user.TenantId
	cluster.Owner = user.Username

	// set created_time and updated_time
	cluster.TimeCreate = dao.GetCurrentTime()
	cluster.TimeUpdate = cluster.TimeCreate
	cluster.Status = CLUSTER_STATUS_INSTALLING
	cluster.SetProjectValue.Cmi = false

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, cluster)
	if err != nil {
		errorCode = CLUSTER_ERROR_CALL_MONGODB
		userLog.Errorf("create cluster [%v] to bson error is %v", cluster, err)
		return
	}

	//add records of hosts in db
	for i := 0; i < cluster.Instances; i++ {
		host := entity.Host{}
		host.ClusterId = cluster.ObjectId.Hex()
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_INSTALLING
		host.UserId = cluster.UserId
		host.Type = cluster.Type
		host.TimeCreate = dao.GetCurrentTime()
		host.TimeUpdate = host.TimeCreate

		_, _, err := GetHostService().Create(host, x_auth_token)
		if err != nil {
			userLog.Errorf("insert host to db error is [%v]", err)
		}
	}

	//call deployment
	err = CreateCluster(cluster, createRequest, logobjid, x_auth_token)
	if err != nil {
		userLog.Errorf("create cluster is err")
		// create cluster is err ,change cluster and host status failed
		_, errorCodeu, erru := p.UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_FAILED, x_auth_token)
		if erru != nil {
			userLog.Errorf("update cluster is err")
			return nil, errorCodeu, erru
		}
		erri := updateHostStatusInCluster(cluster.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
		if erri != nil {
			userLog.Errorf("update host status to failed  err %v", erri)
			errorCode = HOST_ERROR_QUERY
			return nil, errorCode, erri
		}
		return nil, errorCode, err
	}

	newCluster = &cluster
	return
}

//query unterminated clusters
//filter by cluster id
//filter by user id
func (p *ClusterService) QueryCluster(name string, userId string, username string, status string, skip int,
	limit int, sort string, x_auth_token string) (total int, clusters []entity.Cluster,
	errorCode string, err error) {
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return total, nil, errorCode, err
	}

	query := bson.M{}
	if len(strings.TrimSpace(name)) > 0 {
		query["name"] = name
	}
	if len(strings.TrimSpace(userId)) > 0 {
		query["user_id"] = userId
	}
	if len(strings.TrimSpace(username)) > 0 {
		query["owner"] = username
	}
	if strings.TrimSpace(status) == "" {
		//query all clusters by default if this parameter is not provided
		//do nothing
	} else if status == "unterminated" {
		//assume a special status
		//"unterminated" means !TERMINATED(DEPLOYING|RUNNING|FAILED|TERMINATING)
		query["status"] = bson.M{"$ne": CLUSTER_STATUS_TERMINATED}
	} else if status == CLUSTER_STATUS_RUNNING || status == CLUSTER_STATUS_INSTALLING ||
		status == CLUSTER_STATUS_FAILED || status == CLUSTER_STATUS_TERMINATED ||
		status == CLUSTER_STATUS_TERMINATING || status == CLUSTER_STATUS_MODIFYING {
		query["status"] = status
	} else {
		errorCode = COMMON_ERROR_INVALIDATE
		err := errors.New("Invalid parameter status")
		return 0, nil, errorCode, err
	}

	return p.queryByQuery(query, skip, limit, sort, x_auth_token, false)
}

func (p *ClusterService) queryByQuery(query bson.M, skip int, limit int, sort string,
	x_auth_token string, skipAuth bool) (total int, clusters []entity.Cluster,
	errorCode string, err error) {
	authQuery := bson.M{}
	if !skipAuth {
		// get auth query from auth service first
		authQuery, err = GetAuthService().BuildQueryByAuth("list_cluster", x_auth_token)
		if err != nil {
			logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
	}

	selector := generateQueryWithAuth(query, authQuery)
	clusters = []entity.Cluster{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&clusters, queryStruct)
	if err != nil {
		logrus.Errorf("query clusters by query [%v] error is %v", query, err)
		errorCode = CLUSTER_ERROR_QUERY
		return
	}
	return
}

func updateHostStatusInCluster(clusterId string, status string, x_auth_token string) (err error) {
	clustername := getClusterNameById(clusterId, x_auth_token)
	upLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	var hosts []entity.Host
	_, hosts, _, err = GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
	if err != nil {
		upLog.Errorf("query all hosts by cluster id error is [%v]", err)
		return err
	}

	for _, host := range hosts {
		_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), status, x_auth_token)
		if err != nil {
			upLog.Errorf("update host by id error is [%v]", err)
			return err
		}
	}
	return
}

func generateQueryWithAuth(oriQuery bson.M, authQuery bson.M) (query bson.M) {
	if len(authQuery) == 0 {
		query = oriQuery
	} else {
		query = bson.M{}
		query["$and"] = []bson.M{oriQuery, authQuery}
	}
	logrus.Debugf("generated query [%v] with auth [%v], result is [%v]", oriQuery, authQuery, query)
	return
}

func (p *ClusterService) UpdateStatusById(objectId string, status string, x_auth_token string) (created bool,
	errorCode string, err error) {

	logrus.Infof("start to update cluster by objectId [%v] status to %v", objectId, status)
	// do authorize first
	if authorized := GetAuthService().Authorize("update_cluster", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
		return
	}
	// validate objectId
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	cluster, _, err := p.QueryById(objectId, x_auth_token)
	if err != nil {
		logrus.Errorf("get cluster by objeceId [%v] failed, error is %v", objectId, err)
		return
	}
	clustername := cluster.Name
	upLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	if cluster.Status == status {
		upLog.Infof("this cluster [%v] is already in state [%v]", cluster, status)
		return false, "", nil
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	change := bson.M{"status": status, "time_update": dao.GetCurrentTime()}
	err = dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if err != nil {
		upLog.Errorf("update cluster with objectId [%v] status to [%v] failed, error is %v", objectId, status, err)
	}
	created = true
	return

}

//update cluster in db
func (p *ClusterService) UpdateCluster(cluster entity.Cluster, x_auth_token string) (created bool,
	errorCode string, err error) {
	query := bson.M{}
	query["_id"] = cluster.ObjectId
	created, err = dao.HandleUpdateOne(cluster, dao.QueryStruct{p.collectionName, query, 0, 0, ""})
	if err != nil {
		errorCode = CLUSTER_ERROR_UPDATE
		return
	}
	return
}

func (p *ClusterService) QueryById(objectId string, x_auth_token string) (cluster entity.Cluster,
	errorCode string, err error) {
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

	// do authorize first
	if authorized := GetAuthService().Authorize("get_cluster", x_auth_token, objectId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("get cluster with objectId [%v] error is %v", objectId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	cluster = entity.Cluster{}
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query cluster [objectId=%v] error is %v", objectId, err)
		errorCode = CLUSTER_ERROR_QUERY
	}
	return

}

func (p *ClusterService) DeleteById(clusterId string, logobjid string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to delete Cluster with id [%v]", clusterId)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("delete_cluster", x_auth_token, clusterId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when deleting cluster with id [%v] , error is %v", clusterId, err)
		return errorCode, err
	}
	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return errorCode, err
	}

	//query cluster
	cluster, errorCode, err := p.QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster error is %v", err)
		return errorCode, err
	}
	clustername := cluster.Name
	delLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	//check status
	switch cluster.Status {
	case CLUSTER_STATUS_INSTALLING, CLUSTER_STATUS_TERMINATING, CLUSTER_STATUS_TERMINATED,
		CLUSTER_STATUS_MODIFYING:
		delLog.Errorf("Cannot operate on a %s cluster", cluster.Status)
		return CLUSTER_ERROR_INVALID_STATUS, errors.New("Cannot operate on a " + cluster.Status + " cluster")

	case CLUSTER_STATUS_RUNNING, CLUSTER_STATUS_FAILED:
		//query all hosts
		total, hosts, errorCode, err := GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
		if err != nil {
			delLog.Errorf("query hosts in cluster %s error is %v", clusterId, err)
			return errorCode, err
		}

		if total <= 0 {
			delLog.Infof("no unterminated host in cluser %s, the cluster will be terminated directly!", cluster.Name)
			_, errorCode, err := p.UpdateStatusById(clusterId, CLUSTER_STATUS_TERMINATED, x_auth_token)
			return errorCode, err
		}

		//set status of all hosts TERMINATING
		for _, host := range hosts {
			_, _, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_TERMINATING, x_auth_token)
			if err != nil {
				delLog.Warnf("update host [objectId=%v] status to terminating  error is %v", host.ObjectId.Hex(), err)
			}
		}

		_, errorCode, erro := p.UpdateStatusById(clusterId, CLUSTER_STATUS_TERMINATING, x_auth_token)
		if err != nil {
			delLog.Errorf("update cluster objectId [%v] status to terminating  failed, error is %v", clusterId, err)
			return errorCode, erro
		}

		//call deployment module API
		err = DeleteCluster(cluster, logobjid, x_auth_token)
		if err != nil {
			delLog.Errorf("delete cluster err %v", err)
			//delete cluster is err change cluster and host status failed
			_, _, erru := p.UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_FAILED, x_auth_token)
			if erru != nil {
				delLog.Warnf("set cluster status to failed err %v", erru)
			}

			_, hosts, _, errq := GetHostService().QueryHosts(cluster.ObjectId.Hex(), 0, 0, HOST_STATUS_TERMINATING, x_auth_token)
			if errq != nil {
				delLog.Warnf("query hosts err %v", errq)
			}
			for _, host := range hosts {
				_, _, errs := GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
				if errs != nil {
					delLog.Errorf("set host status to failed  error %v", err)
				}
			}

			return CLUSTER_ERROR_DELETE, err
		}

	default:
		delLog.Errorf("Unknown cluster status %s", cluster.Status)
		return CLUSTER_ERROR_INVALID_STATUS, errors.New("Unknown cluster status " + cluster.Status)
	}

	return
}

func (p *ClusterService) AddHosts(addrequest entity.AddRequest, logobjid string, x_auth_token string) (cluster entity.Cluster, errorCode string, err error) {
	logrus.Infof("start to add hosts, request is %v", addrequest)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return cluster, errorCode, err
	}

	clusterId := addrequest.ClusterId
	number := addrequest.SharedCount + addrequest.PureSlaveCount
	addmode := addrequest.AddMode
	nodes := []entity.Node{}
	for _, shareNode := range addrequest.SharedNodes {
		nodes = append(nodes, shareNode)
	}
	for _, pureslaveNode := range addrequest.PureSlaveNodes {
		nodes = append(nodes, pureslaveNode)
	}

	// do authorize first
	if authorized := GetAuthService().Authorize("create_host", x_auth_token, clusterId, p.collectionName); !authorized {
		err = errors.New("required opertion is not authorized!")
		errorCode = COMMON_ERROR_UNAUTHORIZED
		logrus.Errorf("authorize failure when adding hosts in cluster with id [%v] , error is %v", clusterId, err)
		return
	}

	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster id.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)
	cluster = entity.Cluster{}
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query cluster [objectId=%v] error is %v", clusterId, err)
		errorCode = CLUSTER_ERROR_QUERY
		return
	}
	clustername := cluster.Name
	addLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	status := cluster.Status
	if status == CLUSTER_STATUS_INSTALLING || status == CLUSTER_STATUS_MODIFYING || status == CLUSTER_STATUS_TERMINATING {
		addLog.Errorf("Cannot operate on a %s cluster", status)
		return cluster, CLUSTER_ERROR_INVALID_STATUS, errors.New("Cannot operate on a " + cluster.Status + " cluster")
	}

	if len(cluster.DockerRegistries) > 0 {
		addrequest.DockerRegistries = cluster.DockerRegistries
	}

	if addmode == "reuse" {
		if len(addrequest.SharedNodes) != int(addrequest.SharedCount) || len(addrequest.PureSlaveNodes) != int(addrequest.PureSlaveCount) {
			addLog.Errorf("add reuse node number is inconsist with request number!")
			err = errors.New("invalid number parameter")
			errorCode = COMMON_ERROR_INVALIDATE
			return
		}

		isexist, err := p.isIpExist(nodes, x_auth_token)
		addLog.Infof("isexist is %v", isexist)
		if err != nil {
			addLog.Errorf("query ip isexist err is %v", err)
			return cluster, HOST_ERROR_QUERY, errors.New("query ip isexist err")
		}

		if isexist {
			addLog.Errorf("the ip of this node has exist")
			return cluster, CLUSTER_ERROR_IPEXIST, errors.New("ip is exist!")
		}

	}

	if number <= 0 {
		//call terminate hosts to do this
		errorCode := CLUSTER_ERROR_INVALID_NUMBER
		err = errors.New("Invalid number, it should be positive")
		return cluster, errorCode, err
	}

	newHosts := []entity.Host{}
	for i := 0; i < int(number); i++ {
		host := entity.Host{}
		host.ClusterId = clusterId
		host.ClusterName = cluster.Name
		host.Status = HOST_STATUS_INSTALLING
		if addmode == "reuse" {
			host.Type = "customized"
		} else {
			host.Type = cluster.Type
		}

		//insert info to db
		newHost, _, err := GetHostService().Create(host, x_auth_token)
		newHosts = append(newHosts, newHost)
		if err != nil {
			addLog.Errorf("create host error is %v", err)
		}
	}

	//start to change cluster status to modifying, modifying means the cluster add node or delete node
	_, _, err = p.UpdateStatusById(clusterId, CLUSTER_STATUS_MODIFYING, x_auth_token)
	if err != nil {
		addLog.Warnf("update cluster with objecid [%v] to modifying failed, error is %v", clusterId, err)
	}

	//call deployment module to add nodes
	err = AddNodes(cluster, addrequest, newHosts, logobjid, x_auth_token)
	if err != nil {
		addLog.Errorf("add nodes to cluster error %v", err)

		//addnode is err change cluster running and add host failed
		_, _, erru := p.UpdateStatusById(clusterId, CLUSTER_STATUS_RUNNING, x_auth_token)
		if erru != nil {
			addLog.Errorf("update cluster with objecid [%v] to running failed, error is %v", clusterId, erru)
			// return cluster, errorCode, err
		}

		for _, host := range newHosts {
			_, _, errs := GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
			if errs != nil {
				addLog.Errorf("update host status to terminated  error %v", errs)
			}
		}

		return cluster, errorCode, err
	}

	return
}

//terminate specified hosts of a cluster
func (p *ClusterService) TerminateHosts(clusterId string, hostIds []string, logobjid string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to decrease cluster hosts [%v]", hostIds)
	errorCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}

	if !bson.IsObjectIdHex(clusterId) {
		err = errors.New("Invalid cluster_id")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	if len(hostIds) == 0 {
		errorCode = COMMON_ERROR_INVALIDATE
		err = errors.New("Empty array of host id")
		return errorCode, err
	}

	//query cluster by clusterId
	cluster := entity.Cluster{}
	clusterSelector := bson.M{}
	clusterSelector["_id"] = bson.ObjectIdHex(clusterId)
	err = dao.HandleQueryOne(&cluster, dao.QueryStruct{p.collectionName, clusterSelector, 0, 0, ""})

	status := cluster.Status
	clustername := cluster.Name
	terLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	if status == CLUSTER_STATUS_INSTALLING || status == CLUSTER_STATUS_MODIFYING || status == CLUSTER_STATUS_TERMINATING {
		terLog.Errorf("Cannot operate on a %s cluster", status)
		return CLUSTER_ERROR_INVALID_STATUS, errors.New("Cannot operate on a " + status + " cluster")
	}

	_, currentHosts, errorCode, err := GetHostService().QueryHosts(clusterId, 0, 0, "unterminated", x_auth_token)
	if err != nil {
		terLog.Errorf("get host by clusterId[%v] error [%v]", clusterId, err)
		return errorCode, err
	}

	if !deletable(currentHosts, hostIds, cluster) {
		terLog.Errorf("cluster's running node should not less than 5 nodes for HA cluster, 2 nodes for compact cluster, or cluster need one(two) shared server for compact(ha) cluster ")
		return CLUSTER_ERROR_DELETE_NODE_NUM, errors.New("too few running node")
	}

	_, sharedhosts, _, errS := GetHostService().QueryHosts(clusterId, 0, 0, "sharedserver", x_auth_token)
	if errS != nil {
		terLog.Errorf("get host by clusterId[%v] error [%v]", clusterId, errS)
	}
	terLog.Infof("sharedhosts len is %v", sharedhosts)
	var sharedids []string
	if len(sharedhosts) != 0 {
		terLog.Infof("sharedhosts len is %v", sharedhosts)
		for _, shared := range sharedhosts {
			sharedids = append(sharedids, shared.ObjectId.Hex())
		}
	}
	var deleshared int
	for _, id := range hostIds {
		if StringInSlice(id, sharedids) {
			deleshared = deleshared + 1
		}
	}
	var nowshared int
	if len(sharedhosts) > deleshared {
		if deleshared != 0 {
			nowshared = len(sharedhosts) - deleshared
		} else {
			nowshared = len(sharedhosts)
		}
	}

	hosts := []entity.Host{}
	originStatus := make(map[string]string)
	for _, hostId := range hostIds {
		//query host
		host, errorCode, err := GetHostService().QueryById(hostId, x_auth_token)
		if err != nil {
			// return errorCode, err
			terLog.Warnf("get host by id[%s] error %v", hostId, err)
			continue
		}

		hosts = append(hosts, host)

		//protect master node
		if host.IsMasterNode {
			return HOST_ERROR_DELETE_MASTER, errors.New("Cannot delete master node")
		}

		originStatus[host.ObjectId.Hex()] = host.Status
		//call API to terminate host(master node cannot be deleted now)
		_, errorCode, err = GetHostService().UpdateStatusById(hostId, HOST_STATUS_TERMINATING, x_auth_token)
		if err != nil {
			terLog.Errorf("terminate host error is %s,%v", errorCode, err)
			continue
		}
	}

	if len(hosts) <= 0 {
		terLog.Infof("no valid hosts will be deleted!")
		return
	}

	// change cluster status modifying
	_, _, err = p.UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_MODIFYING, x_auth_token)
	if err != nil {
		terLog.Warnf("change cluster status to modifying err is %v", err)
	}

	//call deployment module to delete nodes
	err = DeleteNodes(cluster, hosts, logobjid, x_auth_token, nowshared)
	if err != nil {
		terLog.Errorf("delete nodes err is %v", err)

		//delete node is err
		_, _, erru := p.UpdateStatusById(cluster.ObjectId.Hex(), CLUSTER_STATUS_RUNNING, x_auth_token)
		if erru != nil {
			terLog.Errorf("change cluster status err is %v", erru)
			return
		}

		for _, host := range hosts {
			preStatus := originStatus[host.ObjectId.Hex()]
			_, _, errs := GetHostService().UpdateStatusById(host.ObjectId.Hex(), preStatus, x_auth_token)
			if errs != nil {
				terLog.Errorf("rollback host status to %s  error %v", preStatus, errs)
				continue
			}
		}

		errorCode = HOST_ERROR_DELETE
		return errorCode, err
	}

	return
}

func deletable(hosts []entity.Host, hostIds []string, cluster entity.Cluster) bool {
	if hosts == nil {
		return false
	}

	var runningHost []entity.Host
	//	runningHost := 0
	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		if host.Status == HOST_STATUS_RUNNING && !StringInSlice(host.ObjectId.Hex(), hostIds) {
			runningHost = append(runningHost, host)
		}
	}

	category := cluster.CreateCategory
	if strings.EqualFold(category, CLUSTER_CATEGORY_HA) && len(runningHost) >= 5 {
		if includeshared(category, runningHost) {
			return true
		}
	} else if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) && len(runningHost) >= 2 {
		if includeshared(category, runningHost) {
			return true
		}
	}
	return false

}

func includeshared(category string, hosts []entity.Host) bool {
	var sharednode []entity.Host
	for _, runhost := range hosts {
		if runhost.IsSharedNode {
			sharednode = append(sharednode, runhost)
		}
	}
	if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
		if len(sharednode) < 2 {
			return false
		}
	} else if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
		if len(sharednode) < 1 {
			return false
		}
	} else {
		return false
	}
	return true

}

func (p *ClusterService) NotifyCluster(clusterNotify entity.NotifyCluster, x_auth_token string) (errCode string, err error) {
	logrus.Infoln("Start to notify cluster!")
	errCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}

	total, clusters, _, erro := p.QueryCluster(clusterNotify.ClusterName, "", clusterNotify.UserName, "unterminated", 0, 0, "", x_auth_token)
	if erro != nil {
		errCode = CLUSTER_ERROR_QUERY
		err = errors.New("query cluster is err")
		return errCode, err
	}

	if total == 0 {
		logrus.Warnf("Notify cluster! No cluster with username %s and clustername %s", clusterNotify.UserName, clusterNotify.ClusterName)
		return
	}

	cluster := clusters[0]
	clustername := cluster.Name
	notiLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	objId := cluster.ObjectId.Hex()

	if clusterNotify.Operation == "install" {
		if clusterNotify.IsSuccess {
			//success to create cluster change cluster status running
			//installing means cluster initialization
			_, errCode, err = p.UpdateStatusById(objId, CLUSTER_STATUS_RUNNING, x_auth_token)
			if err != nil {
				notiLog.Errorf("update cluster with clustername [%v] to congfiguring failed, error is %v", clusterNotify.ClusterName, err)
				return errCode, err
			}

			notiLog.Infof("start to create cluster components")
			errCode, err = p.CreateComponent(objId, x_auth_token)
			if err != nil {
				notiLog.Errorf("create component err is %v", err)
			}

		} else {

			_, errCode, err = p.UpdateStatusById(objId, CLUSTER_STATUS_FAILED, x_auth_token)
			if err != nil {
				notiLog.Errorf("update cluster with clustername [%v] to failed failed, error is %v", clusterNotify.ClusterName, err)
				return errCode, err
			}

		}
	} else if clusterNotify.Operation == "delete" {
		if clusterNotify.IsSuccess {
			//delete means terminated cluster, if success change cluster status terminated
			_, errCode, err = p.UpdateStatusById(objId, CLUSTER_STATUS_TERMINATED, x_auth_token)
			if err != nil {
				notiLog.Warnf("update cluster with clustername [%v] to terminated, error is %v", clusterNotify.ClusterName, err)
			}

			//here must change host status terminated
			errCode, err = p.updateDeleteHosts(objId, true, x_auth_token)
			if err != nil {
				notiLog.Errorf("update host is err")
				return errCode, err
			}

			notiLog.Infof("start to delete cluster component")
			_, errD := GetComponentService().DeleteByClusterid(objId, x_auth_token)
			if errD != nil {
				notiLog.Errorf("delete component by clusterid err is %v", errD)
			}

		} else {
			_, errCode, err = p.UpdateStatusById(objId, CLUSTER_STATUS_FAILED, x_auth_token)
			if err != nil {
				notiLog.Warnf("update cluster with clustername [%v] to failed failed, error is %v", clusterNotify.ClusterName, err)
			}

			// terminated cluster err start change host status failed
			errCode, err = p.updateDeleteHosts(objId, false, x_auth_token)
			if err != nil {
				notiLog.Errorf("update host is err ")
				return errCode, err
			}
		}
	} else { //for "add" and "move" operations
		_, errCode, err = p.UpdateStatusById(objId, CLUSTER_STATUS_RUNNING, x_auth_token)
		if err != nil {
			notiLog.Errorf("update cluster with clustername [%v] to failed failed, error is %v", clusterNotify.ClusterName, err)
			return errCode, err
		}
	}

	return
}

func (p *ClusterService) CreateComponent(objectid string, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to create component")
	errCode, err = TokenValidation(x_auth_token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}
	cluster, _, err := GetClusterService().QueryById(objectid, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		return
	}

	component := entity.Components{}
	component.ClusterName = cluster.Name
	component.ClusterId = objectid
	component.UserName = cluster.Owner
	component.UserId = cluster.UserId
	masterIps, errCodeM, errM := p.GetIpsOrSwarmFromCluster(objectid, "master", x_auth_token)
	if errM != nil {
		logrus.Errorf("get masterips is fail, err is %v", errM)
		return errCodeM, errM
	}
	if len(masterIps) != 0 {
		masterCom := entity.Image{}
		masterCom.ImageName = MasterImages
		masterCom.NodeInfo = masterIps
		component.MasterComponents = masterCom
	} else {
		logrus.Errorf("masterips can not be 0")
		return
	}

	slaveIps, errCodeS, errS := p.GetIpsOrSwarmFromCluster(objectid, "slave", x_auth_token)
	if errS != nil {
		logrus.Errorf("get slaveIps is fail, err is %v", errS)
		return errCodeS, errS
	}
	if len(slaveIps) != 0 {
		slaveCom := entity.Image{}
		slaveCom.ImageName = SlaveImages
		slaveCom.NodeInfo = slaveIps
		component.SlaveComponents = slaveCom
	} else {
		logrus.Errorf("slaveIps can not be 0")
		return
	}

	allIps, errCodeA, errA := p.GetIpsOrSwarmFromCluster(objectid, "all", x_auth_token)
	if errA != nil {
		logrus.Errorf("get allIps is fail, err is %v", errA)
		return errCodeA, errA
	}
	if len(allIps) != 0 {
		allCom := entity.Image{}
		allCom.ImageName = AllImage
		allCom.NodeInfo = allIps
		component.AllComponents = allCom
	} else {
		logrus.Errorf("allIps can not be 0")
		return
	}

	clientip, errCodeC, errC := p.GetClientOrMonitorIp(objectid, "client", x_auth_token)
	if errC != nil {
		logrus.Errorf("get clientip is fail, err is %v", errC)
		return errCodeC, errC
	}

	monitorip, errCodeT, errT := p.GetClientOrMonitorIp(objectid, "monitor", x_auth_token)
	if errT != nil {
		logrus.Errorf("get clientip is fail, err is %v", errT)
		return errCodeT, errT
	}
	component.ClientIp = clientip
	component.MonitorIp = monitorip
	onlyonecom := entity.Image{}
	onlyonecom.ImageName = OneInMasterImage
	onlyonecom.NodeInfo = masterIps
	component.OnlyOneComponents = onlyonecom
	component.SwarmName = masterIps[0].HostName

	_, errCodeQ, errQ := GetComponentService().Create(component, x_auth_token)
	if errQ != nil {
		logrus.Errorf("create component is fail, err is %v", errQ)
		return errCodeQ, errQ
	}
	return
}

func (p *ClusterService) GetClientOrMonitorIp(clusterid string, label string, x_auth_token string) (ip string, errCode string, err error) {
	logrus.Infof("start to get ips from cluster")
	total, hosts, _, errq := GetHostService().QueryHosts(clusterid, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return ip, errCode, err
	}
	logrus.Infof("get hosts is %v", hosts)
	if total != 0 {
		for _, host := range hosts {
			if label == "client" {
				if host.IsClientNode {
					ip = host.IP
				}
			} else if label == "monitor" {
				if host.IsMonitorServer {
					ip = host.IP
				}
			}
		}
	}
	return
}

func (p *ClusterService) GetIpsOrSwarmFromCluster(clusterid string, label string, x_auth_token string) (results []entity.IpHostName, errCode string, err error) {
	logrus.Infof("start to get ips from cluster")
	total, hosts, _, errq := GetHostService().QueryHosts(clusterid, 0, 0, "runandoffline", x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return results, errCode, err
	}
	logrus.Infof("get hosts is %v", hosts)
	if total != 0 {
		for _, host := range hosts {
			if label == "master" {
				logrus.Infof("start to get master ips from cluster")
				master := entity.IpHostName{}
				if host.IsMasterNode {
					master.IP = host.IP
					master.HostName = host.HostName
					results = append(results, master)
				}
				logrus.Infof("master ips from cluster is %v", results)
			} else if label == "slave" {
				logrus.Infof("start to get slave ips from cluster")
				slave := entity.IpHostName{}
				if host.IsSlaveNode {
					slave.IP = host.IP
					slave.HostName = host.HostName
					results = append(results, slave)
				}
				logrus.Infof("slave ips from cluster is %v", results)
			} else if label == "all" {
				logrus.Infof("start to get all ips from cluster")
				all := entity.IpHostName{}
				all.IP = host.IP
				all.HostName = host.HostName
				results = append(results, all)
				logrus.Infof("all ips from cluster is %v", results)
			}
		}
	}

	return
}

func (p *ClusterService) NotifyHosts(hostnotify entity.NotifyHost, x_auth_token string) (errCode string, err error) {
	clustername := hostnotify.ClusterName
	hostLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	hostLog.Infoln("start to notify hosts")
	hostLog.Infof("notify service is %v", hostnotify.Servers)
	errCode, err = TokenValidation(x_auth_token)
	if err != nil {
		hostLog.Errorf("token validate err is %v", err)
		return
	}

	total, clusters, _, erro := p.QueryCluster(clustername, "", hostnotify.UserName, "unterminated", 0, 0, "", x_auth_token)
	if erro != nil {
		errCode = CLUSTER_ERROR_QUERY
		return errCode, erro
	}
	if total == 0 {
		hostLog.Warnf("Notify Hosts! No cluster with username %s and clustername %s", hostnotify.UserName, clustername)
		return
	}
	cluster := clusters[0]

	if hostnotify.Operation == "install" || hostnotify.Operation == "add" {
		//update install host status and info
		errcodeu, erru := p.updateInstallHosts(hostnotify, cluster, x_auth_token)
		if erru != nil {
			return errcodeu, erru
		}
	} else if hostnotify.Operation == "move" {
		errCodem, errm := p.updateMoveHosts(hostnotify, cluster, x_auth_token)
		if errm != nil {
			return errCodem, errm
		}
	} else if hostnotify.Operation == "temporary" {
		errCodeT, errT := p.updateTemporaryHosts(hostnotify, cluster, x_auth_token)
		if errT != nil {
			return errCodeT, errT
		}
	}

	return

}

func (p *ClusterService) updateTemporaryHosts(hostnotify entity.NotifyHost, cluster entity.Cluster, x_auth_token string) (errCode string, err error) {
	clustername := cluster.Name
	uptLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	uptLog.Infof("notify host for operation: %s", hostnotify.Operation)
	objId := cluster.ObjectId.Hex()
	var status string
	if hostnotify.IsSuccess {
		status = HOST_STATUS_INSTALLING
	} else {
		status = HOST_STATUS_FAILED
	}

	if len(hostnotify.Servers) != 0 {
		for _, server := range hostnotify.Servers {

			total, hosts, _, errq := GetHostService().QueryHosts(objId, 0, 0, HOST_STATUS_INSTALLING, x_auth_token)
			if errq != nil {
				errCode = HOST_ERROR_QUERY
				err = errors.New("query hosts is err")
				return errCode, err
			}
			uptLog.Infof("total is %v", total)

			if total == 0 {
				uptLog.Warnf("there is no installing host for cluster Id %s", objId)
				return
			}

			flag := false
			for _, host := range hosts {
				if host.HostName == server.Hostname {
					flag = true
					break
				}

			}
			if !flag {
				for _, host := range hosts {
					if host.HostName == "" {
						host.HostName = server.Hostname
						host.IP = server.IpAddress
						host.PrivateIp = server.PrivateIpAddress
						host.IsMasterNode = server.IsMaster
						host.IsSlaveNode = server.IsSlave
						host.IsSharedNode = server.IsSharedServer
						host.IsMonitorServer = server.IsMonitorServer
						host.IsFullfilled = server.IsFullfilled
						host.IsClientNode = server.IsClientServer
						host.SshUser = server.SshUser
						host.Status = status

						_, _, err := GetHostService().UpdateById(host.ObjectId.Hex(), host, x_auth_token)
						if err != nil {
							uptLog.Errorf("update host by id error is [%v]", err)
							break
						}
						break
					}

				}
			}

		}
	}

	return errCode, nil

}

func (p *ClusterService) updateMoveHosts(hostnotify entity.NotifyHost, cluster entity.Cluster, x_auth_token string) (errCode string, err error) {
	clustername := cluster.Name
	upmLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	upmLog.Infof("notify host for operation: %s", hostnotify.Operation)
	objId := cluster.ObjectId.Hex()
	total, hosts, _, errq := GetHostService().QueryHosts(objId, 0, 0, HOST_STATUS_TERMINATING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}

	if total == 0 {
		upmLog.Warnf("there is no terminating host for cluster Id %s", objId)
		return
	}

	if hostnotify.IsSuccess {
		var hostnames []string
		var status string
		for _, server := range hostnotify.Servers {
			hostnames = append(hostnames, server.Hostname)
		}

		for _, host := range hosts {
			if StringInSlice(host.HostName, hostnames) {
				status = HOST_STATUS_TERMINATED
			} else {
				status = HOST_STATUS_FAILED
			}

			_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), status, x_auth_token)
			if err != nil {
				upmLog.Errorf("update host status error is [%v]", err)
			}
			if status == HOST_STATUS_TERMINATED {
				upmLog.Infof("start to delete the ovs network")
				p.deleteNetwork(host, x_auth_token)
			}
		}

		errCode, err = p.changeClusterInstance(objId, x_auth_token)
		if err != nil {
			return errCode, err
		}

	} else {
		for _, host := range hosts {
			_, errCode, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
			if err != nil {
				upmLog.Errorf("change host status to %s error %v", HOST_STATUS_FAILED, err)
				continue
			}
		}

		errCode, err = p.changeClusterInstance(objId, x_auth_token)
		if err != nil {
			return errCode, err
		}
	}
	return
}

func (p *ClusterService) deleteNetwork(host entity.Host, x_auth_token string) (err error) {
	cluster, _, err := p.QueryById(host.ClusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("get cluster by objeceId [%v] failed, error is %v", host.ClusterId, err)
		return
	}
	clientenpoint := cluster.Endpoint
	_, err = DeleteOvsNetwork(clientenpoint, cluster.ObjectId.Hex(), host.HostName)
	return
}

// TODO
func (p *ClusterService) updateInstallHosts(hostnotify entity.NotifyHost, cluster entity.Cluster, x_auth_token string) (errCode string, err error) {
	clustername := cluster.Name
	upiLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	upiLog.Infof("notify host for operation: %s", hostnotify.Operation)
	objId := cluster.ObjectId.Hex()
	length := len(hostnotify.Servers)

	total, hosts, _, errq := GetHostService().QueryHosts(objId, 0, 0, HOST_STATUS_INSTALLING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, err
	}

	upiLog.Infof("total is %v", total)

	if total == 0 {
		upiLog.Warnf("there is no host for cluster Id %s", objId)
		return
	}

	if hostnotify.IsSuccess {
		if length == total {
			for _, host := range hosts {
				_, _, err := GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_RUNNING, x_auth_token)
				if err != nil {
					upiLog.Errorf("update host status err is %v", err)
					continue
				}
			}
		}

		// find the running host and change the cluster instances
		errCode, err = p.changeClusterInstance(objId, x_auth_token)
		if err != nil {
			return errCode, err
		}

		//deal with cluster endpoint
		if hostnotify.Operation == "install" {
			selector := bson.M{}
			selector["_id"] = cluster.ObjectId
			ser := hostnotify.Servers
			point := buildEndPoint(ser, hostnotify.ClusterName)

			change := bson.M{"endPoint": point, "time_update": dao.GetCurrentTime()}
			erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
			if erro != nil {
				upiLog.Errorf("update cluster with objectId [%v] Endpoints to [%s] failed, error is %v", cluster.ObjectId, point, erro)
			}
		}

	} else {
		if len(hostnotify.Servers) != 0 {
			for _, server := range hostnotify.Servers {
				flag := false

				total, hosts, _, errq := GetHostService().QueryHosts(objId, 0, 0, HOST_STATUS_INSTALLING, x_auth_token)
				if errq != nil {
					errCode = HOST_ERROR_QUERY
					err = errors.New("query hosts is err")
					return errCode, err
				}

				upiLog.Infof("total is %v", total)

				if total == 0 {
					upiLog.Warnf("there is no host for cluster Id %s", objId)
					return
				}

				for _, host := range hosts {
					if host.HostName == server.Hostname {
						flag = true
						break
					}

				}
				if !flag {
					for _, host := range hosts {
						if host.HostName == "" {
							host.HostName = server.Hostname
							host.IP = server.IpAddress
							host.PrivateIp = server.PrivateIpAddress
							host.IsMasterNode = server.IsMaster
							host.IsSlaveNode = server.IsSlave
							host.IsSharedNode = server.IsSharedServer
							host.IsMonitorServer = server.IsMonitorServer
							host.IsFullfilled = server.IsFullfilled
							host.SshUser = server.SshUser
							host.IsClientNode = server.IsClientServer
							host.Status = HOST_STATUS_FAILED

							_, _, err := GetHostService().UpdateById(host.ObjectId.Hex(), host, x_auth_token)
							if err != nil {
								upiLog.Errorf("update host by id error is [%v]", err)
								break
							}
							break
						}

					}
				}

			}
		}

		for _, host := range hosts {
			_, errCode, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
			if err != nil {
				upiLog.Errorf("update host status is err")
				continue
			}
		}

	}

	return errCode, nil
}

func (p *ClusterService) changeClusterInstance(objId string, x_auth_token string) (errorCode string, err error) {
	clustername := getClusterNameById(objId, x_auth_token)
	changeLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	changeLog.Infof("start to change cluster instances")
	totalNow, _, errCode, errn := GetHostService().QueryHosts(objId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errn != nil {
		return errCode, errn
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(objId)

	change := bson.M{"instances": totalNow, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		changeLog.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", objId, totalNow, erro)
		return errCode, erro
	}
	return
}

func (p *ClusterService) updateDeleteHosts(clusterObjId string, isSuccess bool, x_auth_token string) (errCode string, err error) {
	clustername := getClusterNameById(clusterObjId, x_auth_token)
	uodLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	total, hosts, _, errq := GetHostService().QueryHosts(clusterObjId, 0, 0, HOST_STATUS_TERMINATING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}
	if total == 0 {
		// errCode = HOST_ERROR_NUMBER
		uodLog.Infof("there is no terminating host for clusterId: %s", clusterObjId)
		return
	}
	if isSuccess {
		for _, host := range hosts {
			_, errCode, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_TERMINATED, x_auth_token)
			if err != nil {
				uodLog.Errorf("update host status is err")
				continue
			}
		}
	} else {
		for _, host := range hosts {
			_, errCode, err = GetHostService().UpdateStatusById(host.ObjectId.Hex(), HOST_STATUS_FAILED, x_auth_token)
			if err != nil {
				uodLog.Errorf("update host status is err")
				continue
			}
		}
	}
	return
}

func getClusterNameById(objid, token string) (clusterName string) {
	cluster, _, err := GetClusterService().QueryById(objid, token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		return
	}
	clusterName = cluster.Name
	return
}

func (p *ClusterService) AddPubkeys(clusterId string, pubkeyIds []string, x_auth_token string) (errCode string, err error) {
	cluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		errCode = CLUSTER_ERROR_QUERY
		return errCode, err
	}
	logrus.Infof("query cluster is %v", cluster)
	if cluster.Status != "RUNNING" {
		logrus.Infof("cluster is not running")
		return CLUSTER_ERROR_QUERY, errors.New("cluster's status is not running")
	}
	clusterName := cluster.Name
	username := cluster.Owner
	providerId := cluster.ProviderId
	provider := entity.IaaSProvider{}
	sshuser := ""
	if len(providerId) != 0 {
		provider, errCode, err = providerService.QueryById(providerId, x_auth_token)
		if err != nil {
			logrus.Errorf("querry provider err is %v", err)
			return errCode, err
		}
		logrus.Infof("query provider is %v", provider)
		sshuser = provider.SshUser
	}

	total, hosts, _, errq := GetHostService().QueryHosts(clusterId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}
	if total == 0 {
		// errCode = HOST_ERROR_NUMBER
		logrus.Infof("there is no terminating host for clusterId: %s", clusterId)
		return HOST_ERROR_NUMBER, errors.New("host num cannot be 0")
	}
	logrus.Infof("query hosts is %v", hosts)

	keys := cluster.PubKeyId
	var arrys []string
	for _, key := range pubkeyIds {
		if key != "" {
			flag := true
			for _, k := range keys {
				if key == k {
					flag = false
				}
			}
			if flag {
				arrys = append(arrys, key)
			}
		}
	}
	logrus.Infof("arrys is %v", arrys)
	if len(arrys) != 0 {
		for _, arry := range arrys {
			keys = append(keys, arry)
		}
	} else {
		logrus.Errorf("pubkey has add to cluster")
		return
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)

	change := bson.M{"pubkeyId": keys, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		logrus.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", clusterId, keys, erro)
		return errCode, erro
	}

	errA := AddPub(clusterName, username, hosts, sshuser, arrys, x_auth_token)
	if errA != nil {
		logrus.Errorf("add pubkey is fail, err is %v", errA)
		return
	}

	return
}

func (p *ClusterService) DeletePubkeys(clusterId string, pubkeyIds []string, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to delete pubkey from cluster")
	if len(pubkeyIds) == 0 {
		logrus.Errorf("there is no pubkey to delete")
		return
	}

	cluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		errCode = CLUSTER_ERROR_QUERY
		return errCode, err
	}

	if cluster.Status != "RUNNING" {
		logrus.Infof("cluster is not running")
		return CLUSTER_ERROR_QUERY, errors.New("cluster's status is not running")
	}

	clusterName := cluster.Name
	username := cluster.Owner
	providerId := cluster.ProviderId
	provider := entity.IaaSProvider{}
	sshuser := ""
	if len(providerId) != 0 {
		provider, errCode, err = providerService.QueryById(providerId, x_auth_token)
		if err != nil {
			logrus.Errorf("querry provider err is %v", err)
			return errCode, err
		}
		logrus.Infof("query provider is %v", provider)
		sshuser = provider.SshUser
	}

	total, hosts, _, errq := GetHostService().QueryHosts(clusterId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}
	if total == 0 {
		// errCode = HOST_ERROR_NUMBER
		logrus.Infof("there is no terminating host for clusterId: %s", clusterId)
		return HOST_ERROR_NUMBER, errors.New("host num cannot be 0")
	}
	logrus.Infof("query hosts is %v", hosts)

	keys := cluster.PubKeyId
	var arrys []string

	for _, k := range keys {
		flag := true
		for _, key := range pubkeyIds {
			if key == k {
				flag = false
			}
		}
		if flag {
			arrys = append(arrys, k)
		}
	}

	logrus.Infof("arrys is %v", arrys)
	var de []string
	for _, k := range pubkeyIds {
		flag := false
		for _, v := range keys {
			if k == v {
				flag = true
			}
		}
		if flag {
			de = append(de, k)
		}
	}

	logrus.Infof("de pubkey is %v", de)

	errA := DeletePub(clusterName, username, hosts, sshuser, de, x_auth_token)
	if errA != nil {
		logrus.Errorf("delete pubkey is fail, err is %v", errA)
		return
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)

	change := bson.M{"pubkeyId": arrys, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		logrus.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", clusterId, keys, erro)
		return errCode, erro
	}

	return
}

func (p *ClusterService) AddRegistry(clusterId string, registryIds []string, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to add registry to cluster")

	cluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		errCode = CLUSTER_ERROR_QUERY
		return errCode, err
	}
	//	logrus.Infof("query cluster is %v", cluster)
	if cluster.Status != "RUNNING" {
		logrus.Infof("cluster is not running")
		return CLUSTER_ERROR_QUERY, errors.New("cluster's status is not running")
	}

	clusterName := cluster.Name
	username := cluster.Owner

	if len(registryIds) == 0 {
		logrus.Errorf("add registry is 0")
		return
	}

	providerId := cluster.ProviderId
	provider := entity.IaaSProvider{}
	sshuser := ""
	if len(providerId) != 0 {
		provider, errCode, err = providerService.QueryById(providerId, x_auth_token)
		if err != nil {
			logrus.Errorf("querry provider err is %v", err)
			return errCode, err
		}
		logrus.Infof("query provider is %v", provider)
		sshuser = provider.SshUser
	}

	total, hosts, _, errq := GetHostService().QueryHosts(clusterId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}
	if total == 0 {
		// errCode = HOST_ERROR_NUMBER
		logrus.Infof("there is no terminating host for clusterId: %s", clusterId)
		return HOST_ERROR_NUMBER, errors.New("host num cannot be 0")
	}
	logrus.Infof("query hosts is %v", hosts)

	slavehost := make([]entity.Host, len(hosts))
	for i, host := range hosts {
		slavehost[i] = host
	}
	logrus.Infof("slave host is %v", slavehost)

	registrys := cluster.DockerRegistries
	var addregistry []entity.DockerRegistry

	for _, registryid := range registryIds {
		if len(registrys) != 0 {
			flag := false
			registry, _, errq := GetDockerRegistryService().QueryById(registryid, x_auth_token)
			if errq != nil {
				logrus.Errorf("query registry err is %v", errq)
				continue
			}
			for _, existre := range registrys {
				if registry.Secure {
					logrus.Infof("registry is secure")
					if !registry.IsSystemRegistry {
						logrus.Infof("registry is not systemregistry")
						if existre.ObjectId != registry.ObjectId {
							logrus.Infof("registry is not exist in cluster")
							flag = true
						} else {
							flag = false
						}
						if !flag {
							logrus.Infof("the registry is had add to cluster")
							break
						}
					}
				}
			}
			logrus.Infof("flag is %v", flag)
			if flag {
				registrys = append(registrys, registry)
				addregistry = append(addregistry, registry)
			}
		} else {
			logrus.Infof("exist registry is 0")
			flag := false
			registry, _, errq := GetDockerRegistryService().QueryById(registryid, x_auth_token)
			if errq != nil {
				logrus.Errorf("query registry err is %v", errq)
				flag = false
			}
			if registry.Secure {
				if !registry.IsSystemRegistry {
					flag = true
				}
			}
			logrus.Infof("flag is %v", flag)
			if flag {
				registrys = append(registrys, registry)
				addregistry = append(addregistry, registry)
			}
		}

	}
	logrus.Infof("addregistry is %v", addregistry)
	logrus.Infof("registrys is %v", registrys)

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)

	change := bson.M{"dockerRegistries": registrys, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		logrus.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", clusterId, registrys, erro)
		return errCode, erro
	}

	errA := AddRegi(clusterName, username, slavehost, sshuser, addregistry, x_auth_token)
	if errA != nil {
		logrus.Errorf("add registry is fail, err is %v", errA)
		return
	}

	return

}

func (p *ClusterService) DeleteRegistry(clusterId string, registryIds []string, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to add registry to cluster")
	cluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		errCode = CLUSTER_ERROR_QUERY
		return errCode, err
	}
	//	logrus.Infof("query cluster is %v", cluster)
	if cluster.Status != "RUNNING" {
		logrus.Infof("cluster is not running")
		return CLUSTER_ERROR_QUERY, errors.New("cluster's status is not running")
	}

	clusterName := cluster.Name
	username := cluster.Owner

	if len(registryIds) == 0 {
		logrus.Errorf("delete registry is 0")
		return
	}

	providerId := cluster.ProviderId
	provider := entity.IaaSProvider{}
	sshuser := ""
	if len(providerId) != 0 {
		provider, errCode, err = providerService.QueryById(providerId, x_auth_token)
		if err != nil {
			logrus.Errorf("querry provider err is %v", err)
			return errCode, err
		}
		logrus.Infof("query provider is %v", provider)
		sshuser = provider.SshUser
	}

	total, hosts, _, errq := GetHostService().QueryHosts(clusterId, 0, 0, HOST_STATUS_RUNNING, x_auth_token)
	if errq != nil {
		errCode = HOST_ERROR_QUERY
		err = errors.New("query hosts is err")
		return errCode, errq
	}
	if total == 0 {
		// errCode = HOST_ERROR_NUMBER
		logrus.Infof("there is no terminating host for clusterId: %s", clusterId)
		return HOST_ERROR_NUMBER, errors.New("host num cannot be 0")
	}
	logrus.Infof("query hosts is %v", hosts)

	slavehost := make([]entity.Host, len(hosts))
	for i, host := range hosts {
		slavehost[i] = host
	}
	logrus.Infof("slave host is %v", slavehost)

	registrys := cluster.DockerRegistries
	var arrys []entity.DockerRegistry
	var deRegistry []entity.DockerRegistry

	if len(registrys) != 0 {
		for _, registry := range registrys {
			flag := true
			for _, deleid := range registryIds {
				deleregi, _, errq := GetDockerRegistryService().QueryById(deleid, x_auth_token)
				if errq != nil {
					logrus.Errorf("query registry err is %v", errq)
					continue
				}

				if deleregi.Secure {
					if !deleregi.IsSystemRegistry {
						if deleregi.ObjectId == registry.ObjectId {
							flag = false
						}
					}
				}
			}
			logrus.Infof("flag is %v", flag)
			if flag {
				arrys = append(arrys, registry)
			} else {
				deRegistry = append(deRegistry, registry)
			}
		}

	} else {
		logrus.Errorf("there is no registrys")
		return
	}

	logrus.Infof("arrys is %v", arrys)
	logrus.Infof("delete registry is %v", deRegistry)

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)

	change := bson.M{"dockerRegistries": arrys, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		logrus.Errorf("update cluster with objectId [%v] instances to [%d] failed, error is %v", clusterId, registrys, erro)
		return errCode, erro
	}

	errA := DeleRegi(clusterName, username, slavehost, sshuser, deRegistry, x_auth_token)
	if errA != nil {
		logrus.Errorf("delete registry is fail, err is %v", errA)
		return
	}

	return

}

func (p *ClusterService) SettingProject(clusterId, cmi, x_auth_token string) (errCode string, err error) {
	logrus.Infof("start to set cmi project value")
	if !strings.EqualFold(cmi, "true") && !strings.EqualFold(cmi, "false") {
		logrus.Errorf("the cmi value is invalid")
		return
	}
	cluster, _, err := GetClusterService().QueryById(clusterId, x_auth_token)
	if err != nil {
		logrus.Errorf("query cluster is fail, err is %v", err)
		errCode = CLUSTER_ERROR_QUERY
		return errCode, err
	}
	//	logrus.Infof("query cluster is %v", cluster)
	if cluster.Status != "RUNNING" {
		logrus.Infof("cluster is not running")
		return CLUSTER_ERROR_QUERY, errors.New("cluster's status is not running")
	}
	var Value bool
	if cmi == "true" {
		Value = true
	} else if cmi == "false" {
		Value = false
	}

	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(clusterId)

	change := bson.M{"setProjectvalue.cmi": Value, "time_update": dao.GetCurrentTime()}
	erro := dao.HandleUpdateByQueryPartial(p.collectionName, selector, change)
	if erro != nil {
		logrus.Errorf("update cluster with objectId [%v] cmi [%d] failed, error is %v", clusterId, Value, erro)
		return errCode, erro
	}
	return
}

func (p *ClusterService) GetComponents(clusterId, x_auth_token string) (componentsInfo entity.ComponentsInfo, errCode string, err error) {
	logrus.Infof("start to get cluster component")
	_, Components, errCodeG, errG := GetComponentService().QuerycomponentByClusterid(clusterId, 0, 0, x_auth_token)
	if errG != nil {
		logrus.Errorf("query component err is %v", errG)
		return componentsInfo, errCodeG, errG
	}
	if len(Components) != 1 {
		logrus.Errorf("component len is err")
		return
	}
	logrus.Infof("cluster component len is %v", len(Components))
	Component := Components[0]
	logrus.Infof("get cluster component is %v", Component)
	slaveIps, errCodeS, errS := p.GetIpsOrSwarmFromCluster(clusterId, "slave", x_auth_token)
	if errS != nil {
		logrus.Errorf("get slaveIps is fail, err is %v", errS)
		return componentsInfo, errCodeS, errS
	}

	allIps, errCodeA, errA := p.GetIpsOrSwarmFromCluster(clusterId, "all", x_auth_token)
	if errA != nil {
		logrus.Errorf("get allIps is fail, err is %v", errA)
		return componentsInfo, errCodeA, errA
	}

	var compareIps []entity.IpHostName
	for _, nodeip := range Component.SlaveComponents.NodeInfo {
		compareIps = append(compareIps, nodeip)
	}
	logrus.Infof("get cluster compareIps is %v", compareIps)
	logrus.Infof("get cluster slaveIps is %v", slaveIps)

	if !reflect.DeepEqual(compareIps, slaveIps) {
		logrus.Infof("cluster slave host has change")
		Component.SlaveComponents.NodeInfo = slaveIps
		Component.AllComponents.NodeInfo = allIps

		_, errCodeU, errU := GetComponentService().UpdateById(Component.ObjectId.Hex(), Component, x_auth_token)
		if errU != nil {
			logrus.Errorf("update components  is fail, err is %v", errU)
			return componentsInfo, errCodeU, errU
		}
	}
	logrus.Infof("cluster slave host not change")

	logrus.Infof("start to send component check to deploy")

	componentsInfo, err = SendComponentCheck(Component)
	if err != nil {
		errCode = GETCOMPONENT_HEALTHCHECK_ERROR
		return
	}

	return
}
