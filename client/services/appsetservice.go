package services

import (
	//	"strconv"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var (
	appsetService  *AppsetService = nil
	onceAppService sync.Once
)

type AppsetService struct {
	collectionName string
}

const (
	//E70000~E70099  Appset
	APPSET_ERR_DECODE_JSON   string = "E70000" // json format error
	APPSET_ERR_NAME_CONFLICT string = "E70003" // appset with name already exist
	APPSET_ERR_NO_APPS       string = "E70011" // can not start an empty appset
	APPSET_ERR_PORT_CONFLICT string = "E70014" // appset has port conflict
	APPSET_ERR_IDS_INVALID   string = "E70015" // IDs of group and app is not belongs to appset

	APPSET_STATUS_IDLE       string = "IDLE"       // no app on marathon
	APPSET_STATUS_DEPLOYING  string = "DEPLOYING"  // at least one app on marathon is deploying
	APPSET_STATUS_RUNNING    string = "RUNNING"    // all apps on marathon are running
	APPSET_STATUS_FAILED     string = "FAILED"     // at least one app on marathon is failed
	APPSET_STATUS_INCOMPLETE string = "INCOMPLETE" // at least one app is not on marathon
	APPSET_STATUS_WAITING    string = "WAITING"    // queue on marathon contains at least one app
	APPSET_STATUS_UNKNOWN    string = "UNKNOWN"    // can not touch marathon

	ENV_APPSET_OBJ_ID      string = "APPSET_OBJ_ID"
	ENV_APPSET_GROUP_ID    string = "LINKER_GROUP_ID"
	ENV_APPSET_APP_ID      string = "LINKER_APP_ID"
	ENV_APPSET_TEMPLATE_ID string = "LINKER_REPAIR_TEMPLATE_ID"
)

func GetAppsetService() *AppsetService {
	onceAppService.Do(func() {
		logrus.Debugf("Once called from appsetService ......................................")
		appsetService = &AppsetService{"appset"}
	})
	return appsetService
}

/**
 * create appset, save appset in db
 * 		appset:	the appset object you want to create
 */
func (p *AppsetService) Create(appset entity.Appset) (newAppset *entity.Appset,
	errorCode string, err error) {
	// init log
	createLog := logrus.WithFields(logrus.Fields{
		"operation": "createappset",
		"appset":    appset.Name})
	createLog.Infof("received request to create appset")
	createLog.Debugf("received request to create appset, appset is %v", appset)

	// check each app and inject uri field if need, for marathon to auth
	appsetBackup := appset
	err = checkAndInjectUriAppset(&appset)
	if err != nil {
		createLog.Errorf("inject uri for appset error: %v", err)
		appset = appsetBackup
		createLog.Infoln("appset restored to init state, nothing changed")
	}

	// validate name firest, name should be unique in db.
	if !appset.IsValid() {
		createLog.Errorf("invalid appset.")
		return nil, COMMON_ERROR_INVALIDATE, errors.New("invalid appset")
	} else if p.isAppsetNameConflict(appset.Name) {
		createLog.Errorf("appset name already exists in db.")
		return nil, APPSET_ERR_NAME_CONFLICT, errors.New("appset name already exists in the cluster")
	}

	// if appset is created by json, validate json's group_id, must equal to appset's name
	if appset.CreatedByJson {
		if len(strings.TrimSpace(appset.Group.ID)) == 0 ||
			strings.TrimLeft(strings.TrimSpace(appset.Group.ID), "/") != appset.Name {
			createLog.Errorf("the id of appset's group must equal to appset's name.")
			return nil, APPSET_ERR_IDS_INVALID, errors.New("invalid group id.")
		}

		// all apps and groups belongs to this group must have save rootid
		groups := appset.GetAllGroups()
		for _, group := range groups {
			if !isBelongsToGroup(group.ID, appset.Group.ID) {
				err = errors.New(fmt.Sprintf("the id of group [%v] is not belongs to the root group.", group.ID))
				createLog.Errorf("%v", err)
				return nil, APPSET_ERR_IDS_INVALID, err
			}
		}
		apps := appset.GetAllApps()
		for _, app := range apps {
			if !isBelongsToGroup(app.ID, appset.Group.ID) {
				err = errors.New(fmt.Sprintf("the id of app [%v] is not belongs to the root group.", app.ID))
				createLog.Errorf("%v", err)
				return nil, APPSET_ERR_IDS_INVALID, err
			}
			// check serviceport unique
			// self check first
			if isServicePortUsedByAppset(*app, appset) {
				err = errors.New("has service port conflict")
				createLog.Errorf("%v", err)
				return nil, APPSET_ERR_PORT_CONFLICT, err
			}
			// check other appsets from db
			isused, errorCode, err := isServicePortUsed(*app)
			if err != nil {
				createLog.Errorf("check port failed: %v", err)
				return nil, errorCode, err
			}
			if isused {
				err = errors.New("has service port conflict")
				createLog.Errorf("%v", err)
				return nil, APPSET_ERR_PORT_CONFLICT, err
			}

		}
	}

	createLog.Infof("appset is validated, will save it into db.")
	// save appset into db.
	appset.ObjectId = bson.NewObjectId()
	appset.TemplateId = appset.Name
	appset.Group.ID = appset.Name
	appset.Status = APPSET_STATUS_IDLE
	appset.TimeCreate = dao.GetCurrentTime()
	appset.TimeUpdate = appset.TimeCreate
	// convert ids to absolute path
	appset.ConvertPath()
	// change each app's dependencies to the absolute path
	appset.OperateOnAllApps(convertDependency)

	if appset.CreatedByJson {
		appsTemp := appset.GetAllApps()
		for _, appTemp := range appsTemp {
			env := appTemp.Env
			if env != nil {
				AlertEnable := (*env)["ALERT_ENABLE"]
				logrus.Infof("alert enable is %v", AlertEnable)
				if AlertEnable == "true" {
					scaleNum := (*env)["SCALESTEP"]
					if scaleNum == "" {
						scaleNum = "1"
					}
					InsMaxNum := (*env)["INSTANCE_MAX_NUM"]
					InsMinNum := (*env)["INSTANCE_MIN_NUM"]

					repairpolicy := entity.RepairPolicy{}
					repairpolicy.RepairTemplateId = appset.Group.ID
					repairpolicy.ServiceGroupId = appset.Group.ID

					appContainerId := appTemp.ID
					if !strings.HasPrefix(appTemp.ID, "/") {
						appContainerId = "/" + appTemp.ID
					}

					repairpolicy.AppCointainerId = appContainerId
					repairpolicy.InstanceMaxNum = InsMaxNum
					repairpolicy.InstanceMinNum = InsMinNum

					Polices := make([]entity.Policy, 4)
					Polices[0].Conditions = []entity.RepairCondition{entity.RepairCondition{Name: "HighCpuAlert", Value: (*env)["CPU_USAGE_HIGH_THRESHOLD"]}}
					Polices[0].Actions = []entity.RepairAction{
						entity.RepairAction{
							Type:           "SCALE",
							AppContainerId: appContainerId,
							Parameters:     []entity.RepairParameter{entity.RepairParameter{Name: "SCALESTEP", Value: scaleNum}},
						},
					}

					Polices[1].Conditions = []entity.RepairCondition{entity.RepairCondition{Name: "LowCpuAlert", Value: (*env)["CPU_USAGE_LOW_THRESHOLD"]}}
					Polices[1].Actions = []entity.RepairAction{
						entity.RepairAction{
							Type:           "SCALE",
							AppContainerId: appContainerId,
							Parameters:     []entity.RepairParameter{entity.RepairParameter{Name: "SCALESTEP", Value: "-" + scaleNum}},
						},
					}

					Polices[2].Conditions = []entity.RepairCondition{entity.RepairCondition{Name: "HighMemoryAlert", Value: (*env)["MEMORY_USAGE_HIGH_THRESHOLD"]}}
					Polices[2].Actions = []entity.RepairAction{
						entity.RepairAction{
							Type:           "SCALE",
							AppContainerId: appContainerId,
							Parameters:     []entity.RepairParameter{entity.RepairParameter{Name: "SCALESTEP", Value: scaleNum}},
						},
					}

					Polices[3].Conditions = []entity.RepairCondition{entity.RepairCondition{Name: "LowMemoryAlert", Value: (*env)["MEMORY_USAGE_LOW_THRESHOLD"]}}
					Polices[3].Actions = []entity.RepairAction{
						entity.RepairAction{
							Type:           "SCALE",
							AppContainerId: appContainerId,
							Parameters:     []entity.RepairParameter{entity.RepairParameter{Name: "SCALESTEP", Value: "-" + scaleNum}},
						},
					}

					repairpolicy.Polices = Polices
					_, _, errR := GetRepairPolicyService().Create(repairpolicy, "")
					if errR != nil {
						logrus.Errorf("create repairpolicy err is %v", errR)
					}

				}
			}

		}
	}

	//save to db

	err = dao.HandleInsert(p.collectionName, appset)
	if err != nil {
		createLog.Errorf("insert appset to db failed, %v", err)
		return nil, COMMON_ERROR_DB, err
	}

	createLog.Infof("appset is saved into db")
	newAppset = &appset
	return newAppset, "", nil
}

/**
 * list all appsets
 * 		skip:	number of items to skip in the result set, default=0
 * 		limit:	maximum number of items in the result set, default=0,
 *				limit=0 means return all records
 *		sort:	comma separated list of field names to sort
 */
func (p *AppsetService) List(skip int, limit int, sort string) (
	total int, appsetlist []entity.AppsetListlViewObj, errorCode string, err error) {
	listLog := logrus.WithFields(logrus.Fields{"operation": "listappsets"})
	listLog.Infof("received request to list appsets")

	total, appsets, errorCode, err := p.getAll(skip, limit, sort)
	if err != nil {
		listLog.Errorf("query appsets form db failed, %v", err)
		return
	}
	appsetlist = []entity.AppsetListlViewObj{}
	for _, appset := range appsets {
		// for each appset, check its status form marathon.
		appset.ConvertPath()
		status, _, _, _ := getAppsetStatus(appset)
		appset.Status = status
		appsetlist = append(appsetlist, appset.ToListView())
	}
	listLog.Infof("list appsets finished, len=%v", len(appsetlist))
	return
}

/**
 * delete an appset, remove it from db
 * 		name:	the name of the appset you want to delete
 */
func (p *AppsetService) Delete(name string) (errorCode string, err error) {
	appset := entity.Appset{Name: name}
	deleteLog := logrus.WithFields(logrus.Fields{
		"operation": "deleteappset",
		"appset":    appset.Name})
	deleteLog.Infof("received request to delete appset")
	// validate name first
	if !appset.IsValid() {
		deleteLog.Errorf("invalid appset name.")
		return COMMON_ERROR_INVALIDATE, errors.New("invalid name")
	}

	// get appset from db
	appset, errorCode, err = p.queryByName(name)
	if err != nil {
		deleteLog.Errorf("query appset by name failed, %v", err)
		return COMMON_ERROR_DB, err
	}

	apps := appset.GetAllApps()
	logrus.Infof("apps is %v", apps)
	for _, app := range apps {
		query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
		id := app.ID
		logrus.Infof("service_group_id is %v", id)
		arr := strings.Split(id, "/")
		if len(arr) >= 2 {
			query["service_group_id"] = "/" + arr[1]

		}

		query1["app_container_id"] = app.ID
		query2["$and"] = []bson.M{query, query1}
		queryStruct := dao.QueryStruct{"repairPolicy", query2, 0, 0, ""}
		polices := []entity.RepairPolicy{}
		_, errQ := dao.HandleQueryAll(&polices, queryStruct)
		if errQ != nil {
			logrus.Errorf("query repairpolicy err is %v", errQ)
		}

		for _, policy := range polices {
			_, errD := GetRepairPolicyService().DeleteById(policy.ObjectId.Hex(), "")
			if errD != nil {
				logrus.Errorf("delete repairpolicy err is %v", errD)
				continue
			}
		}

	}

	// if appset is not empty, delete it from marathon first
	if len(appset.GetAllApps()) > 0 {
		// if the group is existed on marathon, delete it from marathon
		exist, err := common.UTIL.MarathonClient.HasGroup(appset.Group.ID)
		if err != nil {
			deleteLog.Errorf("check group in marathon failed, %v", err)
			return COMMON_ERROR_MARATHON, err
		}
		if exist {
			id, err := common.UTIL.MarathonClient.DeleteGroup(appset.Group.ID, true)
			if err != nil {
				deleteLog.Errorf("delete group from marathon failed, %v", err)
				return COMMON_ERROR_MARATHON, err
			}
			err = common.UTIL.MarathonClient.WaitOnDeployment(id.DeploymentID, 0)
			if err != nil {
				deleteLog.Errorf("wait for deployment finish im marathon failed, %v", err)
				return COMMON_ERROR_MARATHON, err
			}
		}
	}

	// delete appset from db
	err = dao.HandleDelete(p.collectionName, true, bson.M{"name": name})
	if err != nil {
		deleteLog.Errorf("delete group from db failed, %v", err)
		return COMMON_ERROR_DB, err
	}
	deleteLog.Infof("appset is deleted from db")
	return
}

/**
 * start an appset, create a marathon group
 * 		name:	the name of the appset you want to start
 */
func (p *AppsetService) Start(name string) (errorCode string, err error) {
	appset := entity.Appset{Name: name}
	startLog := logrus.WithFields(logrus.Fields{
		"operation": "startappset",
		"appset":    appset.Name})
	startLog.Infof("received request to start appset")
	// validate name first
	if !appset.IsValid() {
		startLog.Errorf("invalid appset name.")
		return COMMON_ERROR_INVALIDATE, errors.New("invalid name")
	}

	// get appset from db
	appset, errorCode, err = p.queryByName(name)
	if err != nil {
		startLog.Errorf("query appset by name failed, %v", err)
		return COMMON_ERROR_DB, err
	}

	// if appset is empty, return error
	if len(appset.GetAllApps()) <= 0 {
		err = errors.New("can not start an empty appset")
		startLog.Errorf("%v", err)
		errorCode = APPSET_ERR_NO_APPS
		return
	}
	// create group on marathon
	// convert apps and group path to absolute path,
	// and refine apps under group, set ENVs.
	appset.ConvertPath()
	appset.OperateOnAllApps(refineEnv, appset)
	group := appset.Group
	startLog.Infof("start to create group in marathon, group is %v", group)
	// if the group is existed on marathon, delete it from marathon
	exist, err := common.UTIL.MarathonClient.HasGroup(group.ID)
	if err != nil {
		startLog.Errorf("check group in marathon failed, %v", err)
		return COMMON_ERROR_MARATHON, err
	}
	if exist {
		// do update
		startLog.Infof("start to update group from marathon")
		_, err := common.UTIL.MarathonClient.UpdateGroup(group.ID, &group, true)
		if err != nil {
			startLog.Errorf("update group from marathon failed, %v", err)
			return COMMON_ERROR_MARATHON, err
		}
	} else {
		// do create
		err = common.UTIL.MarathonClient.CreateGroup(&group)
		if err != nil {
			startLog.Errorf("create group in marathon failed, %v", err)
			errorCode = COMMON_ERROR_MARATHON
			return
		}
	}
	startLog.Infof("appset is started")
	return
}

/**
 * stop an appset, delete a marathon group
 * 		name:	the name of the appset you want to stop
 */
func (p *AppsetService) Stop(name string) (errorCode string, err error) {
	appset := entity.Appset{Name: name}
	stopLog := logrus.WithFields(logrus.Fields{
		"operation": "stopappset",
		"appset":    appset.Name})
	stopLog.Infof("received request to stop appset")
	// validate name first
	if !appset.IsValid() {
		stopLog.Errorf("invalid appset name.")
		return COMMON_ERROR_INVALIDATE, errors.New("invalid name")
	}
	// get appset from db
	appset, errorCode, err = p.queryByName(name)
	if err != nil {
		stopLog.Errorf("query appset by name failed, %v", err)
		return COMMON_ERROR_DB, err
	}

	// if appset is empty, return error
	if len(appset.GetAllApps()) <= 0 {
		err = errors.New("can not start an empty appset")
		stopLog.Errorf("%v", err)
		errorCode = APPSET_ERR_NO_APPS
		return
	}

	// convert path first
	appset.ConvertPath()

	//delete groups
	// if the group is existed on marathon, delete it from marathon
	exist, err := common.UTIL.MarathonClient.HasGroup(appset.Group.ID)
	if err != nil {
		stopLog.Errorf("check group in marathon failed, %v", err)
		return COMMON_ERROR_MARATHON, err
	}
	if exist {
		stopLog.Infof("start to delete group from marathon")
		_, err := common.UTIL.MarathonClient.DeleteGroup(appset.Group.ID, true)
		if err != nil {
			stopLog.Errorf("delete group from marathon failed, %v", err)
			return COMMON_ERROR_MARATHON, err
		}
	}
	stopLog.Infof("appset is stopped")
	return
}

/**
 * get appset details
 * 		name:		the name of the appset you want to get
 * 		skipGroup:	including group info or not
 */
func (p *AppsetService) GetDetail(name string, skipGroup, displayMonitor bool) (
	appsetDetail *entity.AppsetDetailViewObj, errorCode string, err error) {
	appset := entity.Appset{Name: name}
	detailLog := logrus.WithFields(logrus.Fields{
		"operation": "getappsetdetail",
		"appset":    appset.Name})
	detailLog.Infof("received request to get appset details")
	// validate name first
	if !appset.IsValid() {
		detailLog.Errorf("invalid appset name.")
		return nil, COMMON_ERROR_INVALIDATE, errors.New("invalid name")
	}
	// get appset from db
	appset, errorCode, err = p.queryByName(name)
	if err != nil {
		detailLog.Errorf("query appset by name failed, %v", err)
		return nil, COMMON_ERROR_DB, err
	}

	// make sure all apps and groups using absolute path as its id
	// appset.ConvertPath()

	// parse all apps under appset and append it to appsetDetail.components
	appsetDetail = GenAppSetDetail(appset, skipGroup, displayMonitor)
	detailLog.Infof("get appset detail finished")
	return
}

/**
 * create appset, save appset in db
 * 		appset:	the appset object you want to create
 */
func (p *AppsetService) Update(appset entity.Appset) (newAppset *entity.Appset,
	errorCode string, err error) {
	// init log
	updateLog := logrus.WithFields(logrus.Fields{
		"operation": "updateappset",
		"appset":    appset.Name})
	updateLog.Infof("received request")
	updateLog.Debugf("appset is %v", appset)

	// check each app and inject uri field if need, for marathon to auth
	appsetBackup := appset
	err = checkAndInjectUriAppset(&appset)
	if err != nil {
		updateLog.Errorf("inject uri for appset error: %v", err)
		appset = appsetBackup
		updateLog.Infoln("appset restored to init state, nothing changed")
	}

	// validate name firest, name should be unique in db.
	if !appset.IsValid() {
		updateLog.Errorf("invalid appset.")
		return nil, COMMON_ERROR_INVALIDATE, errors.New("invalid appset")
	}

	oldAppset, errorCode, err := p.queryByName(appset.Name)
	if err != nil {
		return
	}

	// if oldAppset is created by json, validate json's group_id, must equal to appset's name
	if oldAppset.CreatedByJson {
		if len(strings.TrimSpace(appset.Group.ID)) == 0 ||
			strings.TrimLeft(strings.TrimSpace(appset.Group.ID), "/") != appset.Name {
			updateLog.Errorf("the id of appset's group must equal to appset's name.")
			return nil, APPSET_ERR_IDS_INVALID, errors.New("invalid group id.")
		}

		// all apps and groups belongs to this group must have save rootid
		groups := appset.GetAllGroups()
		for _, group := range groups {
			if !isBelongsToGroup(group.ID, appset.Group.ID) {
				err = errors.New(fmt.Sprintf("the id of group [%v] is not belongs to the root group.", group.ID))
				updateLog.Errorf("%v", err)
				return nil, APPSET_ERR_IDS_INVALID, err
			}
		}
		apps := appset.GetAllApps()
		for _, app := range apps {
			if !isBelongsToGroup(app.ID, appset.Group.ID) {
				err = errors.New(fmt.Sprintf("the id of app [%v] is not belongs to the root group.", app.ID))
				updateLog.Errorf("%v", err)
				return nil, APPSET_ERR_IDS_INVALID, err
			}
			// check serviceport unique
			// self check first
			if isServicePortUsedByAppset(*app, appset) {
				err = errors.New("has service port conflict")
				updateLog.Errorf("%v", err)
				return nil, APPSET_ERR_PORT_CONFLICT, err
			}
			// check other appsets from db
			isused, errorCode, err := isServicePortUsed(*app)
			if err != nil {
				updateLog.Errorf("check port failed: %v", err)
				return nil, errorCode, err
			}
			if isused {
				err = errors.New("has service port conflict")
				updateLog.Errorf("%v", err)
				return nil, APPSET_ERR_PORT_CONFLICT, err
			}
		}
		// update appset.group
		// deepCopy(&oldAppset.Group, &appset.Group)
		oldAppset.Group = appset.Group
	}
	updateLog.Infof("appset is validated, will save it into db.")
	// save appset into db.
	oldAppset.Description = appset.Description
	oldAppset.TimeUpdate = dao.GetCurrentTime()
	// convert ids to absolute path
	oldAppset.ConvertPath()
	// change each app's dependencies to the absolute path
	oldAppset.OperateOnAllApps(convertDependency)
	oldAppset.OperateOnAllApps(refineEnv, oldAppset)

	// if appset is created by json, and it's status is not IDLE,
	// call update group api to update group on marathon.
	if oldAppset.CreatedByJson {
		status, _, errorCode, err := getAppsetStatus(oldAppset)
		if err != nil {
			updateLog.Errorf("get appset status failed, %v", err)
			return nil, errorCode, err
		}
		if status != APPSET_STATUS_IDLE {
			_, err = common.UTIL.MarathonClient.UpdateGroup(oldAppset.Group.ID, &oldAppset.Group, true)
			if err != nil {
				return nil, COMMON_ERROR_MARATHON, err
			}
		}
	}

	//save to db
	err = p.syncToDB(&oldAppset)
	if err != nil {
		errorCode = COMMON_ERROR_DB
		return
	}

	updateLog.Infof("appset is saved into db")
	newAppset = &oldAppset
	return
}

/**
 * save appset to db
 */
func (p *AppsetService) syncToDB(appset *entity.Appset) (err error) {
	dbLog := logrus.WithFields(logrus.Fields{"appset": appset.Name})
	dbLog.Infof("will save appset into db")
	var selector = bson.M{}
	selector["_id"] = appset.ObjectId
	// convert ids to absolute path
	appset.ConvertPath()
	_, err = dao.HandleUpdateOne(appset, dao.QueryStruct{p.collectionName,
		selector, 0, 0, ""})
	if err != nil {
		dbLog.Errorf("save appset into db failed, %v", err)
	}
	dbLog.Infof("saved appset into db")
	return
}

func (p *AppsetService) isAppsetNameConflict(appsetName string) (conflict bool) {
	logrus.Debugf("will check appset [%v] whether exist in db", appsetName)
	selector := bson.M{}
	selector["name"] = appsetName
	n, _, _, err := p.query(selector, 0, 0, "")
	if err != nil {
		logrus.Errorf("query db for appsets failed, %v", err)
		return true
	}
	//found conflict
	if n > 0 {
		return true
	}
	return false
}

func (p *AppsetService) query(selector bson.M, skip int, limit int, sort string) (
	total int, appsets []entity.Appset, errorCode string, err error) {
	appsets = []entity.Appset{}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}
	total, err = dao.HandleQueryAll(&appsets, queryStruct)
	if err != nil {
		// logrus.Errorf("query appsets error is %v", err)
		errorCode = COMMON_ERROR_DB
		return
	}
	return
}

func (p *AppsetService) queryByName(name string) (appset entity.Appset, errorCode string, err error) {
	selector := bson.M{}
	selector["name"] = name
	total, appsets, errorCode, err := p.query(selector, 0, 0, "")
	if err != nil {
		return
	}
	if total > 0 {
		appset = appsets[0]
	} else {
		// logrus.Errorf("can not find appset with [name=%v]", name)
		err = errors.New("not found")
		errorCode = COMMON_ERROR_DB
	}
	return
}

func (p *AppsetService) getAll(skip int, limit int, sort string) (
	total int, appsets []entity.Appset, errorCode string, err error) {
	// get appsets from db first
	selector := bson.M{}
	total, appsets, errorCode, err = p.query(selector, skip, limit, sort)
	if err != nil {
		errorCode = COMMON_ERROR_DB
		return
	}
	return
}

func checkAndInjectUriAppset(appset *entity.Appset) (err error) {
	// get all registry url
	registryList, err := getRegistryList()
	if err != nil {
		logrus.Errorf("get registry list error: %v", err)
		return
	}
	appset.OperateOnAllApps(checkAndInjectUri, registryList)
	return
}

func (p *AppsetService) GetAppName(name string) (names []string, errcode string, err error) {
	appset, code, err := p.queryByName(name)
	if err != nil {
		logrus.Errorf("query appset err is %v", err)
		return nil, code, err
	}

	apps := appset.GetAllApps()
	for _, app := range apps {
		names = append(names, app.ID)
	}
	return

}
