package services

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	ALERT_MESSAGES_STATUS_FIRING        = "firing"
	ALERT_MESSAGES_STATUS_REPAIRED      = "repaired"
	ALERT_MESSAGES_STATUS_REPAIR_FAILED = "failed"
	ALERT_MESSAGES_STATUS_IGNORED       = "ignored"

	ALERT_ERROR_CREATE string = "E54001"
	ALERT_ERROR_QUERY  string = "E54002"
)

var (
	alertService *AlertService = nil
	onceAlert    sync.Once
)

type AlertService struct {
	collectionName string
}

func GetAlertService() *AlertService {
	onceAlert.Do(func() {
		logrus.Debugf("Once called from alertService ......................................")
		alertService = &AlertService{"alert"}
	})
	return alertService
}

func (p *AlertService) Create(alert entity.Alert) (newAlert entity.Alert,
	errorCode string, err error) {
	logrus.Infof("start to create alert [%v]", alert)

	// generate ObjectId
	alert.ObjectId = bson.NewObjectId()

	// set created_time and updated_time
	alert.TimeCreate = dao.GetCurrentTime()
	alert.TimeUpdate = alert.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, alert)
	if err != nil {
		errorCode = ALERT_ERROR_CREATE
		logrus.Errorf("create alert [%v] to bson error is %v", alert, err)
		return
	}

	newAlert = alert

	return
}

func (p *AlertService) CheckRelatedRepairs(alert *entity.Alert) bool {

	selector := make(bson.M)
	//		querymatch := make(bson.M)
	//		queryvalue := make(bson.M)

	//		querymatch["$elemMatch"] = queryvalue

	//	selector["alert"] = querymatch
	selector["status"] = ALERT_MESSAGES_STATUS_FIRING
	selector["labels.app_id"] = alert.Labels.AppId
	selector["labels.group_id"] = alert.Labels.GroupId

	messages := []entity.AlertMessage{}
	queryStruct := dao.QueryStruct{p.collectionName, selector, 0, 0, "...."}
	total, err := dao.HandleQueryAll(&messages, queryStruct)

	logrus.Debugf("Query related alert with GroupId: %s, AppId: %s, total: $d", alert.Labels.GroupId, alert.Labels.AppId, total)

	if err != nil {
		logrus.Errorf("Query alert failed, error is %v", err)
		return false
	}
	if total > 0 {
		return false
	} else {
		return true
	}

}

func (p *AlertService) NotifyRepairResult(alertId, result string) {
	logrus.Infof("Alert id is %s, result is %s", alertId, result)
	selector := bson.M{}
	selector["_id"] = bson.ObjectIdHex(alertId)

	change := bson.M{}
	change["status"] = result
	err := dao.HandlePartialUpdateByQuery(p.collectionName, selector, change)
	if err != nil {
		logrus.Errorf("Update alert by id: %s failed, error is %v", alertId, err)
	}
}

//list all alerts
func (p *AlertService) ListAlerts(groupID, appID, alertName, action string, doNothing bool,
	skip int, limit int, sort string) (total int, alerts *[]entity.AlertResp, errorCode string, err error) {

	selector := bson.M{}
	if len(groupID) > 0 {
		selector["labels.group_id"] = groupID
	}
	if len(appID) > 0 {
		selector["labels.app_id"] = appID
	}
	if len(alertName) > 0 {
		selector["labels.alert_name"] = alertName
	}
	// true: the results contain *DO_NOTHING* actions
	if doNothing {
		//
	} else {
		// false: ignore REPAIR_ACTION_DONOTHING[_MAX|_MIN] actions
		notEqual1, notEqual2, notEqual3 := bson.M{}, bson.M{}, bson.M{}
		notEqual1["status"] = bson.M{"$ne": REPAIR_ACTION_DONOTHING}
		notEqual2["status"] = bson.M{"$ne": REPAIR_ACTION_DONOTHING_MAX}
		notEqual3["status"] = bson.M{"$ne": REPAIR_ACTION_DONOTHING_MIN}
		selector["$and"] = []bson.M{notEqual1, notEqual2, notEqual3}
	}

	if len(action) > 0 {
		selector["status"] = action
	}

	total, alertMessages, errorCode, err := p.query(selector, skip, limit, sort)
	if err != nil {
		return 0, nil, errorCode, err
	}

	var alertRespArr []entity.AlertResp
	for _, v := range alertMessages {
		var a = entity.AlertResp{
			ObjectId:   v.ObjectId,
			GroupID:    v.Labels.GroupId,
			AppID:      v.Labels.AppId,
			AlertName:  v.Labels.AlertName,
			Action:     v.Status,
			TimeUpdate: v.TimeUpdate,
		}
		alertRespArr = append(alertRespArr, a)
	}

	alerts = &alertRespArr
	return
}

func (p *AlertService) query(selector bson.M, skip int, limit int, sort string) (total int, alertMessages []entity.Alert,
	errorCode string, err error) {

	alertMessages = []entity.Alert{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&alertMessages, queryStruct)
	if err != nil {
		logrus.Errorf("query alerts by query [%v] error is %v", selector, err)
		errorCode = ALERT_ERROR_QUERY
		return
	}
	return
}
