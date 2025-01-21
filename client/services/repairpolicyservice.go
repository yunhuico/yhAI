package services

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var (
	repairPolicyService           *RepairPolicyService = nil
	onceRepairPolicy              sync.Once
	REPAIR_POLICY_ERROR_CREATE    string = "E11130"
	REPAIR_POLICY_ERROR_UPDATE    string = "E11131"
	REPAIR_POLICY_ERROR_DELETE    string = "E11132"
	REPAIR_POLICY_ERROR_QUERY     string = "E11133"
	REPAIR_RECORD_ERROR_CREATE    string = "E11134"
	REPAIR_RECORD_ERROR_UPDATE    string = "E11135"
	REPAIR_RECORD_ERROR_QUERY     string = "E11136"
	REPAIR_ERROR                  string = "E11137"
	repairRecordCollection        string = "repairRecord"
	repairPolicyCollection        string = "repairPolicy"
	REPAIR_ACTION_TYPE_SCALE      string = "SCALE"
	REPAIR_ACTION_TYPE_SCALE_STEP string = "SCALESTEP"
	REPAIR_ACTION_FAILURE         string = "REPAIR_ACTION_FAILURE"
	REPAIR_ACTION_SUCCESS         string = "REPAIR_ACTION_SUCCESS"
	REPAIR_ACTION_PARTIALSUCCESS  string = "REPAIR_ACTION_PARTIALSUCCESS"
	REPAIR_ACTION_SUCCESS_OUT     string = "REPAIR_ACTION_SUCCESS_OUT"
	REPAIR_ACTION_SUCCESS_IN      string = "REPAIR_ACTION_SUCCESS_IN"
	REPAIR_ACTION_DONOTHING       string = "REPAIR_ACTION_DONOTHING"
	REPAIR_ACTION_DONOTHING_MAX   string = "REPAIR_ACTION_DONOTHING_MAX"
	REPAIR_ACTION_DONOTHING_MIN   string = "REPAIR_ACTION_DONOTHING_MIN"
	REPAIR_FAILED                 string = "REPAIR_FAILED"
)

type RepairPolicyService struct {
	collectionName string
}

func GetRepairPolicyService() *RepairPolicyService {
	onceRepairPolicy.Do(func() {
		logrus.Debugf("Once called from repairPolicyService ......................................")
		repairPolicyService = &RepairPolicyService{repairPolicyCollection}
	})
	return repairPolicyService
}

func (p *RepairPolicyService) Create(repairPolicy entity.RepairPolicy, x_auth_token string) (newRepairPolicy entity.RepairPolicy,
	errorCode string, err error) {
	logrus.Infof("start to create repairPolicy [%v]", repairPolicy)
	// // do authorize first
	// if authorized := services.GetAuthService().Authorize("create_repairpolicy", x_auth_token, "", p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("create repairPolicy [%v] error is %v", repairPolicy, err)
	// 	return
	// }
	// generate ObjectId
	repairPolicy.ObjectId = bson.NewObjectId()

	// token, err := services.GetTokenById(x_auth_token)
	// if err != nil {
	// 	errorCode = REPAIR_POLICY_ERROR_CREATE
	// 	logrus.Errorf("get token failed when create repairPolicy [%v], error is %v", repairPolicy, err)
	// 	return
	// }

	// // set token_id and user_id from token
	// repairPolicy.Tenant_id = token.Tenant.Id
	// repairPolicy.User_id = token.User.Id

	// set created_time and updated_time
	repairPolicy.TimeCreate = dao.GetCurrentTime().String()
	repairPolicy.TimeUpdate = repairPolicy.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, repairPolicy)
	if err != nil {
		errorCode = REPAIR_POLICY_ERROR_CREATE
		logrus.Errorf("create repairPolicy [%v] to bson error is %v", repairPolicy, err)
		return
	}

	newRepairPolicy = repairPolicy
	return
}

func (p *RepairPolicyService) UpdateById(objectId string, repairPolicy entity.RepairPolicy, x_auth_token string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update repairPolicy [%v]", repairPolicy)
	// // do authorize first
	// if authorized := services.GetAuthService().Authorize("update_repairpolicy", x_auth_token, objectId, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("update repairPolicy with objectId [%v] error is %v", objectId, err)
	// 	return
	// }
	// validate repairPolicy
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	// reset objectId and updated_time
	repairPolicy.ObjectId = bson.ObjectIdHex(objectId)
	repairPolicy.TimeUpdate = dao.GetCurrentTime().String()

	// insert bson to mongodb
	created, err = dao.HandleUpdateOne(&repairPolicy, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("update repairPolicy [%v] error is %v", repairPolicy, err)
		errorCode = REPAIR_POLICY_ERROR_UPDATE
	}
	return
}

func (p *RepairPolicyService) DeleteById(objectId string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to delete repairPolicy with objectId [%v]", objectId)
	// // do authorize first
	// if authorized := services.GetAuthService().Authorize("delete_repairpolicy", x_auth_token, objectId, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("delete repairPolicy with objectId [%v] error is %v", objectId, err)
	// 	return
	// }

	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		logrus.Errorf("delete repairPolicy [objectId=%v] error is %v", objectId, err)
		errorCode = REPAIR_POLICY_ERROR_DELETE
	}
	return
}

func (p *RepairPolicyService) QueryAll(skip int, limit int, x_auth_token string) (total int, repairPolicys []entity.RepairPolicy,
	errorCode string, err error) {
	// // get auth query from auth service first
	// authQuery, err := services.GetAuthService().BuildQueryByAuth("list_repairpolicies", x_auth_token)
	// if err != nil {
	// 	logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
	// 	errorCode = COMMON_ERROR_INTERNAL
	// 	return
	// }

	// logrus.Debugf("auth query is %v", authQuery)

	// selector := services.GenerateQueryWithAuth(bson.M{}, authQuery)
	selector := bson.M{}
	logrus.Debugf("selector is %v", selector)
	sort := ""
	repairPolicys = []entity.RepairPolicy{}
	queryStruct := dao.QueryStruct{p.collectionName, selector, skip, limit, sort}
	total, err = dao.HandleQueryAll(&repairPolicys, queryStruct)
	if err != nil {
		logrus.Errorf("list repairPolicy [token=%v] failed, error is %v", x_auth_token, err)
		errorCode = REPAIR_POLICY_ERROR_QUERY
	}
	return
}

func (p *RepairPolicyService) QueryById(objectId string, x_auth_token string) (repairPolicy entity.RepairPolicy,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	// // do authorize first
	// if authorized := services.GetAuthService().Authorize("get_repairpolicy", x_auth_token, objectId, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("get repairPolicy with objectId [%v] error is %v", objectId, err)
	// 	return
	// }

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	repairPolicy = entity.RepairPolicy{}
	err = dao.HandleQueryOne(&repairPolicy, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query repairPolicy [objectId=%v] error is %v", objectId, err)
		errorCode = REPAIR_POLICY_ERROR_QUERY
	}
	return
}

func (p *RepairPolicyService) GetOperationById(objectId string, x_auth_token string) (operations map[string]int,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// operationList := []string{"update_repairpolicy", "delete_repairpolicy"}
	// operations, err = services.GetAuthService().AuthOperation(operationList, x_auth_token, objectId, p.collectionName)
	// if err != nil {
	// 	logrus.Errorf("get auth operation of [objectId=%v] error is %v", objectId, err)
	// 	errorCode = COMMON_ERROR_INTERNAL
	// }

	operations = make(map[string]int)
	operations["update_repairpolicy"] = 1
	operations["delete_repairpolicy"] = 1
	return
}

func (p *RepairPolicyService) NotifyUser(alertName, hostIP, currentVal string, hostrules entity.HostRules, startAt time.Time) (errCode string, err error) {
	var friendlyAlert, threshold string
	switch alertName {
	case "HostHighCPUAlert":
		friendlyAlert = "High CPU usage"
		threshold = fmt.Sprintf("%0.2f", hostrules.Thresholds["cpu_high"])
	case "HostLowCPUAlert":
		friendlyAlert = "Low CPU usage"
		threshold = fmt.Sprintf("%0.2f", hostrules.Thresholds["cpu_low"])
	case "HostHighMemoryAlert":
		friendlyAlert = "High memory usage"
		threshold = fmt.Sprintf("%0.2f", hostrules.Thresholds["mem_high"])
	case "HostLowMemoryAlert":
		friendlyAlert = "Low memory usage"
		threshold = fmt.Sprintf("%0.2f", hostrules.Thresholds["mem_low"])
	}
	var duration = hostrules.Duration
	var startAtStr = startAt.Format(time.RFC1123)
	// type BasicInfo struct {
	// 	MgmtIp      []string `bson:"mgmtIp" json:"mgmtIp"`
	// 	ClusterName string   `bson:"clusterName" json:"clusterName"`
	// 	ClusterId   string   `bson:"clusterId" json:"clusterId"`
	// 	UserName    string   `bson:"userName" json:"userName"`
	// 	UserId      string   `bson:"user_id" json:"user_id"`
	// 	TenantId    string   `bson:"tenant_id" json:"tenant_id"`
	// 	MonitorIp string `bson:"monitorIp" json:"monitorIp"`
	// }
	info, lg := common.BasicInfo, common.Logger // cluster description JSON file
	availAPI := lg.CreateEmailURL(info.MgmtIp)

	var userName, clusterName = info.UserName, info.ClusterName
	var subject = fmt.Sprintf("[Linker DC/OS] %s on cluster '%s'\n", friendlyAlert, clusterName)

	content, err := generateEmailContent(userName, clusterName, hostIP, alertName, duration, threshold, currentVal, startAtStr)
	if err != nil {
		logrus.Errorf("generate email content error: %v\n", err)
		return COMMON_ERROR_INTERNAL, err
	}
	if err = lg.SendEmail(availAPI, info.UserId, subject, content); err != nil {
		return COMMON_ERROR_INTERNAL, err
	}
	return "", err
}

func generateEmailContent(userName, clusterName, hostIP, alertName, duration, threshold,
	currentVal, startAt string) (content string, err error) {
	const emailTempl = `
		Dear {{.Username}},

		  We have detected unusual resource usage on your Linker DC/OS cluster.
		  Here is a short description of the alert.

		  Cluster name: {{.ClusterName}}
		  Host machine IP: {{.HostIP}}
		  Alert name: {{.AlertName}}
		  Have lasted for: {{.Duration}}
		  Theresholds: {{.Threshold}}% (current value: {{.CurrentValue}}%)
		  Start at: {{.StartAt}}

		  Please login to the host and check if anything goes wrong.

		  Best regards
		  Linker Networks
		   `

	type ToReplaceFields struct {
		Username     string
		ClusterName  string
		HostIP       string
		AlertName    string
		Duration     string
		Threshold    string
		CurrentValue string
		StartAt      string
	}
	var fields = ToReplaceFields{userName, clusterName, hostIP, alertName, duration,
		threshold, currentVal, startAt}
	// Create a new template and parse the letter into it.
	t := template.Must(template.New("email").Parse(emailTempl))
	var buf bytes.Buffer
	if err = t.Execute(&buf, fields); err != nil {
		return "", err
	}
	return buf.String(), nil
}

/*
Get Alert and find the correct repair actions
*/
func (p *RepairPolicyService) AnalyzeAlert(repairTemplateId, serviceGroupId, AppCointainerId,
	serviceGroupInsanceId, orderId, alertId, alertName, alertValue string) (errorCode string, err error) {
	// get the xtoken
	// x_auth_token, err := GenerateToken()
	if err != nil {
		logrus.Errorf("get token for repair [repairTemplateId=%v], [serviceGroupId=%v] and [AppCointainerId=%v] error is %v", repairTemplateId, serviceGroupId, AppCointainerId, err)
		GetAlertService().NotifyRepairResult(alertId, "REPAIR_FAILED")
		return
	}

	//find the repair policy
	_, polices, _, err := queryRepairPolicy(repairTemplateId, serviceGroupId, AppCointainerId, "x_auth_token")
	logrus.Infof("repairepolicy is %v", polices)
	if err != nil || len(polices) <= 0 {
		logrus.Errorf("query policy for repair [repairTemplateId=%v], [serviceGroupId=%v] and [AppCointainerId=%v] error is %v", repairTemplateId, serviceGroupId, AppCointainerId, err)
		GetAlertService().NotifyRepairResult(alertId, "REPAIR_FAILED")
		return
	}
	repairPolicy := polices[0]
	logrus.Infof("repairpolicy[0] is %v", repairPolicy)

	action := entity.RepairAction{}

	//find the detailed action,current only support the first action
	for _, policy := range repairPolicy.Polices {
		for _, condition := range policy.Conditions {
			if condition.Name == alertName {
				action = policy.Actions[0]
				logrus.Infof("action is %v", action)
			}
		}
	}

	intanceMaxNumber := repairPolicy.InstanceMaxNum
	intanceMinNumber := repairPolicy.InstanceMinNum

	//call the do repair operation
	p.doRepairOperation(serviceGroupId, alertId, alertName, orderId, serviceGroupInsanceId, action, intanceMaxNumber, intanceMinNumber, "x_auth_token")
	return
}

/*
Doing Repair Operation
*/
func (p *RepairPolicyService) doRepairOperation(serviceGroupId, alertId string, alertName string, orderId string,
	serviceGroupInsanceId string, action entity.RepairAction, intanceMaxNumber string, intanceMinNumber string, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to doRepairOperation!")

	if action.Type == REPAIR_ACTION_TYPE_SCALE {
		logrus.Infof("action.type is equal to SCALE")
		//get the current status
		// sgi, _, err1 := GetSgiService().QueryById(serviceGroupInsanceId, x_auth_token)
		// TODO: what is this mean?
		// GetAppsetService().queryById(serviceGroupId)

		// if err1 != nil {
		// 	errorCode = REPAIR_ERROR
		// 	err = errors.New("repair action failed while loading the instance!")
		// 	logrus.Errorf("query instance for repair [objectId=%v] error is %v", serviceGroupInsanceId, err)
		// 	GetAlertService().NotifyRepairResult(alertId, "SGI_STATUS_FAILED")
		// 	return
		// }
		// currentStatus := sgi.LifeCycleStatus

		//start the scaleout repair
		appPath := action.AppContainerId
		scaleStep := ""
		isPartial := false
		logrus.Infof("scaleStep is %v", scaleStep)

		for _, parameter := range action.Parameters {
			if parameter.Name == REPAIR_ACTION_TYPE_SCALE_STEP {
				scaleStep = parameter.Value
				logrus.Infof("scaleStep two is %v", scaleStep)
			}
		}

		if len(scaleStep) <= 0 {
			return
		}

		//insert repair record
		repairRecord := entity.RepairRecord{}
		repairRecord.AppCointainerId = appPath
		repairRecord.ServiceGroupId = serviceGroupId
		repairRecord.ServiceGroupInstanceId = serviceGroupInsanceId
		repairRecord.Status = "REPAIRING"
		repairRecord.AlertId = alertId
		repairRecord.AlertName = alertName
		//temp set deployment id with alert id
		repairRecord.DeploymentId = alertId
		repairRecord.Action = strings.Join([]string{REPAIR_ACTION_TYPE_SCALE, scaleStep}, ":")

		logrus.Infof("start to createRepairRecord")
		newRepairRecord, _, err1 := createRepairRecord(repairRecord)

		if err1 != nil {
			err = errors.New("repair action failed while save repair record!")
			errorCode = REPAIR_ERROR
			logrus.Errorf("create repair record failed [instanceid=%v] error is %v", serviceGroupInsanceId, err)
			GetAlertService().NotifyRepairResult(alertId, "REPAIR_FAILED")
			return
		}

		//TODO check if the app is already in Repair
		// _, errorCode, err = GetSgiService().UpdateRepairIdAndStatusById(serviceGroupInsanceId, newRepairRecord.RepairId, "SGI_STATUS_REPARING", x_auth_token)

		//start scale
		logrus.Infof("start to scale")
		number, appNum, err2 := caculateNewScaleNumber(appPath, serviceGroupId, scaleStep)

		if err2 != nil {
			err = errors.New("repair action failed while caculate the number for scale")
			errorCode = REPAIR_ERROR
			logrus.Errorf("can't caculate new instance number for reparing scaling  err is %v", err)
			GetAlertService().NotifyRepairResult(alertId, "REPAIR_FAILED")
			return
		}

		if intanceMaxNumber != "" || intanceMinNumber != "" {
			var errCheck error
			var result bool
			//Check that the number is out of range
			number, isPartial, result, errCheck = checkScaleNumber(number, appNum, intanceMaxNumber, intanceMinNumber)
			logrus.Infof("after check the scale number is %v ", number)
			if errCheck != nil {
				err = errors.New("repair action failed while check the number for scale")
				errorCode = REPAIR_ERROR
				logrus.Errorf("check scale number failed, err is %v", errCheck)
			}
			if !result {
				logrus.Infof("instance has reached the upper limit or lower limit, the number is v%", number)
				if number == intanceMaxNumber {
					GetAlertService().NotifyRepairResult(alertId, REPAIR_ACTION_DONOTHING_MAX)
				}
				if number == intanceMinNumber {
					GetAlertService().NotifyRepairResult(alertId, REPAIR_ACTION_DONOTHING_MIN)
				}
				return
			}
		}

		//call App Set Service to Scale Out Service
		logrus.Infof("start to call appsetservice to scale out service")
		deploymentId, _, err3 := GetComponentService().Scale(appPath, number)

		//if repair failed change the instance status back
		if err3 != nil {
			err = errors.New("repair action failed while scaling")
			logrus.Errorf("scale instance for repairing [serviceGroupId=%v] ,[appPath=%v] error is %v", serviceGroupId, appPath, err3)
			p.AnalyzeNotify(REPAIR_ACTION_FAILURE, alertId)
			return
		}

		logrus.Infof("start to updateDeploymentidbyid")
		_, err4 := updateDeploymentIdById(newRepairRecord.ObjectId.Hex(), deploymentId)

		if err4 != nil {
			err = errors.New("repair action failed while update repair record ")
			logrus.Errorf("update repair record for deploymentId [deploymentId=%v] ,[appPath=%v] error is %v", deploymentId, appPath, err)
			p.AnalyzeNotify(REPAIR_ACTION_FAILURE, deploymentId)
			return
		}

		// update repairRecord
		err5 := common.UTIL.MarathonClient.WaitOnDeployment(deploymentId, 0)
		if err5 != nil {
			err = errors.New("wait for deployment failed")
			logrus.Errorf("waitfor deployment [Id=%v] failed, error is %v", deploymentId, err5)
			p.AnalyzeNotify(REPAIR_ACTION_FAILURE, deploymentId)
			return
		}
		if isPartial {
			logrus.Infof("waitfor deployment [Id=%v] partial success", deploymentId)
			p.AnalyzeNotify(REPAIR_ACTION_PARTIALSUCCESS, deploymentId)
		} else {
			logrus.Infof("waitfor deployment [Id=%v] success", deploymentId)
			if number > appNum {
				p.AnalyzeNotify(REPAIR_ACTION_SUCCESS_OUT, deploymentId)
			} else {
				p.AnalyzeNotify(REPAIR_ACTION_SUCCESS_IN, deploymentId)
			}
		}
	} else {
		logrus.Errorf("can't find the repaie action type, type is %v", action.Type)
		GetAlertService().NotifyRepairResult(alertId, "REPAIR_FAILED")
		return
	}
	return
}

/*
Get instance repair finished Notify and Notify repair and Alert Finished
*/
func (p *RepairPolicyService) AnalyzeNotify(deploymentStatus string, deploymentId string) (errorCode string, err error) {

	logrus.Info("find record record with deploymentId: [%v]", deploymentId)
	record, _, err := queryRecordByDeploymentId(deploymentId)
	if err != nil {
		logrus.Errorf("can't find repair record with deploymentId [%v] err is [%v]", deploymentId, err)
		return
	}

	//update repair record status to finished
	logrus.Info("update record record with repair id: [%v], reparing result: [%v]", record.ObjectId.Hex(), deploymentStatus)
	_, _, err = updateStatusById(record.ObjectId.Hex(), deploymentStatus)
	if err != nil {
		errorCode = REPAIR_ERROR
		logrus.Errorf("can't update record record with reparing result err is %v", err)
		GetAlertService().NotifyRepairResult(record.ObjectId.Hex(), REPAIR_ACTION_FAILURE)
		return
	}

	//call the alert of reparing finished/Failed
	GetAlertService().NotifyRepairResult(record.AlertId, deploymentStatus)

	return
}

func createRepairRecord(repairRecord entity.RepairRecord) (newRepairRecord entity.RepairRecord,
	errorCode string, err error) {
	logrus.Infof("start to create repairRecord [%v]", repairRecord)

	// generate ObjectId
	repairRecord.ObjectId = bson.NewObjectId()
	repairRecord.RepairId = strings.Join([]string{repairRecord.AlertId, repairRecord.ObjectId.Hex()}, "-")

	// set created_time and updated_time
	repairRecord.TimeCreate = dao.GetCurrentTime().String()
	repairRecord.TimeUpdate = repairRecord.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(repairRecordCollection, repairRecord)
	if err != nil {
		errorCode = REPAIR_RECORD_ERROR_CREATE
		logrus.Errorf("create repairRecord [%v] to bson error is %v", repairRecord, err)
		return
	}

	newRepairRecord = repairRecord
	return
}

func queryRecordByDeploymentId(deploymentId string) (repairRecord entity.RepairRecord,
	errorCode string, err error) {
	var selector = bson.M{}
	selector["deployment_id"] = deploymentId
	repairRecord = entity.RepairRecord{}
	err = dao.HandleQueryOne(&repairRecord, dao.QueryStruct{repairRecordCollection, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query repairRecord [deploymentId=%v] error is %v", deploymentId, err)
		errorCode = REPAIR_RECORD_ERROR_QUERY
	}
	return
}

func queryRecordById(objectId string) (repairRecord entity.RepairRecord,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	repairRecord = entity.RepairRecord{}
	err = dao.HandleQueryOne(&repairRecord, dao.QueryStruct{repairRecordCollection, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query repairRecord [objectId=%v] error is %v", objectId, err)
		errorCode = REPAIR_RECORD_ERROR_QUERY
	}
	return
}

func updateDeploymentIdById(recordId string, deploymentId string) (
	errorCode string, err error) {
	logrus.Infof("start to update repairRecord [%v]", recordId)

	// validate recordId
	if !bson.IsObjectIdHex(recordId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// get record by recordId
	record, _, err := queryRecordById(recordId)
	if err != nil {
		logrus.Errorf("get record by recordId [%v] failed, error is %v", recordId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(recordId)

	// reset objectId and updated_time
	record.ObjectId = bson.ObjectIdHex(recordId)
	record.DeploymentId = deploymentId
	record.TimeUpdate = dao.GetCurrentTime().String()

	// insert bson to mongodb
	_, err = dao.HandleUpdateOne(&record, dao.QueryStruct{repairRecordCollection, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("update record with recordId [%v] deploymentId to [%v] failed, error is %v", recordId, deploymentId, err)
	}
	return
}

func updateStatusById(recordId string, status string) (created bool,
	errorCode string, err error) {
	logrus.Infof("start to update repairRecord [%v]", recordId)

	// validate recordId
	if !bson.IsObjectIdHex(recordId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// get record by recordId
	record, _, err := queryRecordById(recordId)
	if err != nil {
		logrus.Errorf("get record by recordId [%v] failed, error is %v", recordId, err)
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(recordId)

	// reset objectId and updated_time
	record.ObjectId = bson.ObjectIdHex(recordId)
	record.Status = status
	record.TimeUpdate = dao.GetCurrentTime().String()

	// insert bson to mongodb
	created, err = dao.HandleUpdateOne(&record, dao.QueryStruct{repairRecordCollection, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("update record with recordId [%v] status to [%v] failed, error is %v", recordId, status, err)
	}
	return
}

func queryRepairPolicy(repairTemplateId, serviceGroupId, appContainerId string, x_auth_token string) (total int, polices []entity.RepairPolicy,
	errorCode string, err error) {
	// // do authorize first
	// authQuery, err := services.GetAuthService().BuildQueryByAuth("get_repairpolicy", x_auth_token)
	// if err != nil {
	// 	logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
	// 	errorCode = COMMON_ERROR_INTERNAL
	// 	return
	// }

	var selector = bson.M{}
	selector["repair_template_id"] = repairTemplateId
	// selector["service_group_id"] = serviceGroupId
	selector["app_container_id"] = appContainerId
	// selector = services.GenerateQueryWithAuth(selector, authQuery)

	queryStruct := dao.QueryStruct{repairPolicyCollection, selector, 0, 0, ""}
	total, err = dao.HandleQueryAll(&polices, queryStruct)
	logrus.Debugf("queryRepairPolicy Total is %v", total)
	if err != nil {
		logrus.Errorf("get repairPolicy with serviceGroupId [%v] and appContainerId [%v] error is %v", serviceGroupId, appContainerId, err)
		errorCode = REPAIR_POLICY_ERROR_QUERY
	}
	return
}

func caculateNewScaleNumber(componentName, appsetName, step string) (numberStr string, appNum string, err error) {
	// currentNumber, _, err := GetComponentService().QueryInstances(componentName, appsetName)
	component, _, err := GetComponentService().Detail(componentName)
	if err != nil {
		logrus.Errorf("Can not find the instance numbmer in the appset instance, err is %v", err)
		return
	}

	stepNumber, err := strconv.Atoi(step)
	if err != nil {
		logrus.Errorf("convert step number [%v] failed, error is %v", step, err)
		return
	}
	number := *component.App.Instances + stepNumber
	numberStr = strconv.Itoa(number)
	appNum = strconv.Itoa(*component.App.Instances)
	return
}

func checkScaleNumber(scaleNumber, appNumber, intanceMaxNumber, intanceMinNumber string) (numberStr string, isPartial bool, result bool, err error) {
	var maxSet bool
	scaleNum, err := strconv.Atoi(scaleNumber)
	if err != nil {
		// ut 1
		logrus.Errorf("convert scale number [%v] failed, error is %v", scaleNumber, err)
		return
	}
	appNum, err := strconv.Atoi(appNumber)
	if err != nil {
		logrus.Errorf("convert app number [%v] failed, error is %v", appNum, err)
		return
	}
	logrus.Infof("before check the scale number is %v, the app intance number is %v ", scaleNum, appNum)

	if intanceMaxNumber != "" {
		maxSet = true
		maxNumber, err2 := strconv.Atoi(intanceMaxNumber)
		if err2 != nil {
			logrus.Errorf("convert intanceMaxNumber  [%v] failed, error is %v", intanceMaxNumber, err2)
			err = err2
			return
		}
		if scaleNum <= maxNumber {
			// ut 2
			result = true
			numberStr = strconv.Itoa(scaleNum)
		} else {
			if appNum != maxNumber {
				// ut 3
				isPartial = true
				result = true
				numberStr = strconv.Itoa(maxNumber)
			} else {
				// ut 4
				result = false
				numberStr = strconv.Itoa(maxNumber)
			}
		}
	}

	if intanceMinNumber != "" {
		minNumber, err3 := strconv.Atoi(intanceMinNumber)
		if err3 != nil {
			logrus.Errorf("convert intanceMinNumber [%v] failed, error is %v", intanceMinNumber, err3)
			err = err3
			return
		}
		if scaleNum >= minNumber && maxSet {
			// ut 5
			return
		} else {
			if scaleNum >= minNumber {
				// ut 6
				result = true
				numberStr = strconv.Itoa(scaleNum)
			} else {
				if appNum != minNumber {
					// ut 7
					isPartial = true
					result = true
					numberStr = strconv.Itoa(minNumber)
				} else {
					// ut 8
					result = false
					numberStr = strconv.Itoa(minNumber)
				}
			}
		}
	}
	return
}
