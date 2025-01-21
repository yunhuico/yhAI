package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	HOSTRULES_ERROR_UPDATE string = "E55001"
	HOSTRULES_ERROR_QUERY  string = "E55002"
)

var (
	hostMonitorService *HostMonitorService = nil
	onceHostMonitor    sync.Once
)

type HostMonitorService struct {
	collectionName string
}

func GetHostMonitorService() *HostMonitorService {
	onceHostMonitor.Do(func() {
		logrus.Debugf("Once called from onceHostMonitor ......................................")
		hostMonitorService = &HostMonitorService{"hostrules"}
	})
	return hostMonitorService
}

func (p *HostMonitorService) GetHostRules(xAuthToken string) (hostrules *entity.HostRules, errCode string, err error) {
	hostrules = &entity.HostRules{}
	query := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       bson.M{},
		Skip:           0,
		Limit:          0,
		Sort:           "",
	}
	if err = dao.HandleQueryOne(hostrules, query); err != nil {
		logrus.Errorf("query hostrules error: %v\n", err)
		return nil, HOSTRULES_ERROR_QUERY, err
	}
	return hostrules, "", nil
}

func (p *HostMonitorService) UpdateRules(request entity.ReqPutRules, token string) (hostrules *entity.HostRules, errorCode string, err error) {
	// type BasicInfo struct {
	// 	MgmtIp      []string `bson:"mgmtIp" json:"mgmtIp"`
	// 	ClusterName string   `bson:"clusterName" json:"clusterName"`
	// 	ClusterId   string   `bson:"clusterId" json:"clusterId"`
	// 	UserName    string   `bson:"userName" json:"userName"`
	// 	UserId      string   `bson:"user_id" json:"user_id"`
	// 	TenantId    string   `bson:"tenant_id" json:"tenant_id"`
	// 	MonitorIp string `bson:"monitorIp" json:"monitorIp"`
	// }
	data, err := json.Marshal(request)
	if err != nil {
		return nil, COMMON_ERROR_INVALIDATE, err
	}
	info := common.BasicInfo
	// rulegen API
	apiURL := fmt.Sprintf("%s:10006/rules", info.MonitorIp)
	logrus.Infof("call API(%s) to generate rule file ...\n", apiURL)
	resp, err := httpclient.Http_put(apiURL, string(data))
	if err != nil {
		logrus.Errorf("call rulegen(%s) to update rules failed: %v\n", apiURL, err)
		return nil, COMMON_ERROR_INTERNAL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, COMMON_ERROR_INTERNAL, errors.New("rulegen returned status code not OK")
	}

	// reload Prometheus rules
	promAPI := fmt.Sprintf("%s:9090/-/reload", info.MonitorIp)
	logrus.Infof("call API(%s) to reload Prometheus rules ...\n", promAPI)
	respProm, err := httpclient.Http_post(promAPI, "")
	if err != nil {
		logrus.Errorf("call prometheus(%s) to reload rule files failed: %v\n", promAPI, err)
		return nil, COMMON_ERROR_INTERNAL, err
	}
	defer respProm.Body.Close()
	if resp.StatusCode != 200 {
		return nil, COMMON_ERROR_INTERNAL, errors.New("Prometheus returned status code not OK")
	}

	// save to db
	hostrules = &entity.HostRules{}

	existingRule, _, err := p.GetHostRules("x-auth-token")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			// not found, create one
			hostrules.ObjectId = bson.NewObjectId()
			hostrules.ReqPutRules = request
			hostrules.TimeCreate = dao.GetCurrentTime()
			hostrules.TimeUpdate = hostrules.TimeCreate
			if errCreate := dao.HandleInsert(p.collectionName, hostrules); errCreate != nil {
				logrus.Errorf("update hostrules [%+v] to db error is %v", *hostrules, errCreate)
				return nil, HOSTRULES_ERROR_UPDATE, errCreate
			} else {
				return hostrules, "", nil
			}
		}
		logrus.Errorf("query hostrules [%+v] error is %v", *hostrules, err)
		return nil, HOSTRULES_ERROR_QUERY, err
	}

	// already exist
	existingRule.ReqPutRules = request
	existingRule.TimeUpdate = dao.GetCurrentTime()
	var selector = bson.M{}
	selector["_id"] = existingRule.ObjectId
	query := dao.QueryStruct{p.collectionName, selector, 0, 0, ""}
	if _, err = dao.HandleUpdateOne(existingRule, query); err != nil {
		logrus.Errorf("update existing rules [%+v] to db error is %v", *existingRule, err)
		return nil, HOSTRULES_ERROR_UPDATE, err
	}

	return existingRule, "", nil
}
