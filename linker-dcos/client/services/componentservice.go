package services

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/mgo.v2/bson"

	marathon "github.com/LinkerNetworks/go-marathon"
	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var (
	componentService     *ComponentService = nil
	onceComponentService sync.Once
)

type ComponentService struct {
}

const (
	//E70100~E70199  Component
	COMPONENT_ERR_APP_INVALID       string = "E70106"
	COMPONENT_ERR_APP_ABSOLUTE_ID   string = "E70111"
	COMPONENT_ERR_APP_CONFLICT      string = "E70102"
	COMPONENT_ERR_APP_PORT_CONFLICT string = "E70103"

	COMPONENT_STATUS_IDLE      string = "IDLE"
	COMPONENT_STATUS_DEPLOYING string = "DEPLOYING"
	COMPONENT_STATUS_RUNNING   string = "RUNNING"
	COMPONENT_STATUS_FAILED    string = "FAILED"
	COMPONENT_STATUS_WAITING   string = "WAITING"
	COMPONENT_STATUS_SUSPENDED string = "SUSPENDED"
	COMPONENT_STATUS_UNKNOWN   string = "UNKNOWN"
)

func GetComponentService() *ComponentService {
	onceComponentService.Do(func() {
		logrus.Debugf("Once called from componentService ......................................")
		componentService = &ComponentService{}
	})
	return componentService
}

func (c *ComponentService) Create(component entity.ComponentViewObj) (
	newComponent *entity.ComponentViewObj, errorCode string, err error) {
	// init log
	createLog := logrus.WithFields(logrus.Fields{
		"operation": "createcomponent",
		"component": component.App.ID,
	})
	createLog.Infof("received request")

	// check app and inject uri field if need, for marathon to auth
	componentBackup := component
	err = checkAndInjectUriComponent(&component)
	if err != nil {
		createLog.Errorf("inject uri for component error: %v", err)
		component = componentBackup
		createLog.Infoln("component restored to init state, nothing changed")
	}

	//validate component first
	if !component.IsValid() {
		createLog.Errorf("invalid component.")
		return nil, COMMON_ERROR_INVALIDATE, errors.New("invalid component")
	} else if !isBelongsToGroup(component.App.ID, component.AppsetName) {
		createLog.Errorf("the app is not belongs to the appset")
		return nil, COMPONENT_ERR_APP_INVALID, errors.New("invalid component")
	} else {
		createLog.Infof("component is validated")
	}

	//check if appset exist
	appset, errorCode, err := GetAppsetService().queryByName(component.AppsetName)
	if err != nil {
		createLog.Errorf("query appset from db failed, %v", err)
		return nil, errorCode, err
	}

	// check app conflict
	if app, _ := getAppFromGroup(&appset.Group, component.App.ID); app != nil {
		createLog.Errorf("app has already exist")
		return nil, COMPONENT_ERR_APP_CONFLICT, errors.New("app has already exist")
	}
	// service port validate
	// self check first
	if isServicePortUsedByAppset(component.App, appset) {
		err = errors.New("has service port conflict")
		createLog.Errorf("%v", err)
		return nil, COMPONENT_ERR_APP_PORT_CONFLICT, err
	}
	// check other appsets from db
	isused, errorCode, err := isServicePortUsed(component.App)
	if err != nil {
		createLog.Errorf("check port failed: %v", err)
		return nil, errorCode, err
	}
	if isused {
		err = errors.New("has service port conflict")
		createLog.Errorf("%v", err)
		return nil, COMPONENT_ERR_APP_PORT_CONFLICT, err
	}
	// insert app into appset.Group
	_, err = insertAppIntoGroup(&appset.Group, component.App)
	if err != nil {
		createLog.Errorf("insert app into group failed, %v", err)
		return nil, COMMON_ERROR_INTERNAL, err
	}
	// change each app's dependencies to the absolute path
	convertDependency(&component.App)
	// save appset to db
	GetAppsetService().syncToDB(&appset)
	createLog.Infof("component is saved into db")

	// convert app to component, then return
	// appset.ConvertPath()
	app, err := getAppFromGroup(&appset.Group, component.App.ID)
	if err != nil {
		createLog.Errorf("get app from group failed, %v", err)
		errorCode = COMMON_ERROR_INTERNAL
		return
	}
	status := COMPONENT_STATUS_IDLE
	comp := convertAppToComponent(app, status)
	newComponent = &comp

	apps := appset.GetAllApps()
	for _, appTemp := range apps {
		env := appTemp.Env
		if env != nil {
			AlertEnable := (*env)["ALERT_ENABLE"]
			logrus.Infof("alert enable is %v", AlertEnable)
			if AlertEnable == "true" {
				scaleNum := (*env)["SCALE_NUMBER"]
				if scaleNum == "" {
					scaleNum = "1"
				}
				AppContainerIdEnv := (*env)["SCALED_APP_ID"]

				appContId := appTemp.ID
				if !strings.HasPrefix(appTemp.ID, "/") {
					appContId = "/" + appTemp.ID
				}
				var appContainerId string
				if AppContainerIdEnv == "" {
					appContainerId = appContId
				} else {
					appContainerId = AppContainerIdEnv
				}

				InsMaxNum := (*env)["INSTANCE_MAX_NUM"]
				InsMinNum := (*env)["INSTANCE_MIN_NUM"]

				repairpolicy := entity.RepairPolicy{}
				repairpolicy.RepairTemplateId = appset.Group.ID
				repairpolicy.ServiceGroupId = appset.Group.ID
				repairpolicy.AppCointainerId = appContId
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

				query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
				query["service_group_id"] = appset.Group.ID
				query["app_container_id"] = appContId
				query["$and"] = []bson.M{query1, query2}
				queryStruct := dao.QueryStruct{"repairPolicy", query, 0, 0, ""}
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
				_, _, errR := GetRepairPolicyService().Create(repairpolicy, "")
				if errR != nil {
					logrus.Errorf("create repairpolicy err is %v", errR)
				}

			}
		}

	}

	return
}

func (c *ComponentService) Detail(name string) (
	component *entity.ComponentViewObj, errorCode string, err error) {
	// init log
	detailLog := logrus.WithFields(logrus.Fields{
		"operation": "getcomponent",
		"component": name,
	})
	detailLog.Infof("received request")

	app, _, errorCode, err := c.getAppwithSet(name)
	if err != nil {
		detailLog.Errorf("get app from appset failed, %v", err)
		return
	}

	// get app status from marathon, and merge together
	status, application, _, _ := getComponentStatus(app)
	detailLog.Debugf("get app status is %v", status)
	comp := convertAppToComponent(application, status)

	detailLog.Infof("get component detail finished")
	component = &comp
	return
}

func (c *ComponentService) Delete(name string) (
	errorCode string, err error) {
	// init log
	deleteLog := logrus.WithFields(logrus.Fields{
		"operation": "deletecomponent",
		"component": name,
	})
	deleteLog.Infof("received request")

	app, appset, errorCode, err := c.getAppwithSet(name)
	if err != nil {
		deleteLog.Errorf("get app from appset failed, %v", err)
		return
	}

	// delete app from marathon if exist
	application, err := common.UTIL.MarathonClient.Application(app.ID)
	if err != nil {
		if isNotFound(err) {
			deleteLog.Infof("app is not in marathon, no need to delete")
		} else {
			deleteLog.Errorf("get app from marathon failed, %v", err)
			errorCode = COMMON_ERROR_MARATHON
			return
		}
	} else {
		_, err = common.UTIL.MarathonClient.DeleteApplication(application.ID, true)
		if err != nil {
			deleteLog.Errorf("delete app from marathon failed, %v", err)
			return COMMON_ERROR_MARATHON, err
		}
		deleteLog.Infof("app is deleted from marathon")
	}

	// remove app from group in appset
	_, err = deleteAppFromGroup(&appset.Group, app.ID)
	if err != nil {
		deleteLog.Errorf("delete app from group failed, %v", err)
		return COMMON_ERROR_INTERNAL, err
	}

	// delete other app's dependencies
	appset.OperateOnAllApps(delDependency, *app)

	// save appset into db
	GetAppsetService().syncToDB(&appset)
	deleteLog.Infof("app is deleted from db")

	deleteLog.Infof("start to selete app repairpolicy")

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
	return
}

func (c *ComponentService) Start(name string) (
	component *entity.ComponentViewObj, errorCode string, err error) {
	// init log
	startLog := logrus.WithFields(logrus.Fields{
		"operation": "startcomponent",
		"component": name,
	})
	startLog.Infof("received request")

	app, appset, errorCode, err := c.getAppwithSet(name)
	if err != nil {
		startLog.Errorf("get app from appset failed, %v", err)
		return
	}

	// add ENVs in app for monitor and autoscaling
	refineEnv(app, appset)

	// post app to marathon
	_, err = common.UTIL.MarathonClient.CreateApplication(app)
	if err != nil {
		startLog.Errorf("create app into marathon failed, %v", err)
		errorCode = COMMON_ERROR_MARATHON
		return
	}
	startLog.Infof("app is created on marathon")
	status, application, _, _ := getComponentStatus(app)
	comp := convertAppToComponent(application, status)
	component = &comp
	return
}

func (c *ComponentService) Stop(name string) (
	errorCode string, err error) {
	// init log
	stopLog := logrus.WithFields(logrus.Fields{
		"operation": "stopcomponent",
		"component": name,
	})
	stopLog.Infof("received request")

	app, _, errorCode, err := c.getAppwithSet(name)
	if err != nil {
		stopLog.Errorf("get app from appset failed, %v", err)
		return
	}

	// delete app from marathon if exist
	_, err = common.UTIL.MarathonClient.Application(app.ID)
	if err != nil {
		if isNotFound(err) {
			stopLog.Infof("app is not in marathon, no need to delete")
		} else {
			stopLog.Errorf("get app from marathon failed, %v", err)
			errorCode = COMMON_ERROR_MARATHON
			return
		}
	} else {
		_, err = common.UTIL.MarathonClient.DeleteApplication(app.ID, true)
		if err != nil {
			stopLog.Errorf("delete app from marathon failed, %v", err)
			errorCode = COMMON_ERROR_MARATHON
		}
		stopLog.Infof("app is deleted from marathon")
	}
	return
}

// deploymentId is used by repairpolicyservice.
func (c *ComponentService) Scale(name, scaleTo string) (
	deploymentId, errorCode string, err error) {
	// init log
	scaleLog := logrus.WithFields(logrus.Fields{
		"operation": "scalecomponent",
		"component": name,
		"scaleto":   scaleTo,
	})
	scaleLog.Infof("received request")
	// validate scaleTo
	toNum, err := strconv.Atoi(scaleTo)
	if err != nil {
		scaleLog.Errorf("convert app num failed, %v", err)
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}
	if toNum < 0 {
		scaleLog.Errorf("scaleTo should greater then 1")
		err = errors.New("scaleTo should greater then 1")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	app, appset, errorCode, err := c.getAppwithSet(name)
	if err != nil {
		scaleLog.Errorf("get app from appset failed, %v", err)
		return
	}

	if *app.Instances == toNum {
		scaleLog.Errorf("app.Instances already equal to scaleTo")
		err = errors.New("app.Instances already equal to scaleTo")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// do scale
	app.Instances = &toNum

	scaleLog.Debugf("will scale %v number to %v on marathon", app.ID, toNum)
	depId, err := common.UTIL.MarathonClient.ScaleApplicationInstances(app.ID, toNum, true)
	if err != nil {
		scaleLog.Errorf("scale app on marathon failed, %v", err)
		errorCode = COMMON_ERROR_MARATHON
		return
	}
	scaleLog.Infof("scale app on marathon finished")
	err = common.UTIL.MarathonClient.WaitOnDeployment(depId.DeploymentID, 5)
	if err != nil {
		scaleLog.Errorf("waiton deployment %v failed, %v", depId.DeploymentID, err)
		scaleLog.Errorf("but scale is success, so return success.")
		err = nil
	}
	deploymentId = depId.DeploymentID
	// save appset into db
	GetAppsetService().syncToDB(&appset)
	scaleLog.Infof("scale app in db finished")
	return
}

func (c *ComponentService) Update(component entity.ComponentViewObj) (
	newComponent *entity.ComponentViewObj, errorCode string, err error) {
	// init log
	updateLog := logrus.WithFields(logrus.Fields{
		"operation": "updatecomponent",
		"component": component.App.ID,
	})
	updateLog.Infof("received request")

	// check app and inject uri field if need, for marathon to auth
	componentBackup := component
	err = checkAndInjectUriComponent(&component)
	if err != nil {
		updateLog.Errorf("inject uri for component error: %v", err)
		component = componentBackup
		updateLog.Infoln("component restored to init state, nothing changed")
	}

	//validate component first
	if !component.IsValid() {
		updateLog.Errorf("invalid component.")
		return nil, COMMON_ERROR_INVALIDATE, errors.New("invalid component")
	} else if !isBelongsToGroup(component.App.ID, component.AppsetName) {
		updateLog.Errorf("the app is not belongs to the appset")
		return nil, COMPONENT_ERR_APP_INVALID, errors.New("invalid component")
	} else {
		updateLog.Infof("component is validated")
	}

	//check if appset exist
	appset, errorCode, err := GetAppsetService().queryByName(component.AppsetName)
	if err != nil {
		updateLog.Errorf("query appset from db failed, %v", err)
		return nil, errorCode, err
	}

	// service port validate
	// self check first
	if isServicePortUsedByAppset(component.App, appset) {
		err = errors.New("has service port conflict")
		updateLog.Errorf("%v", err)
		return nil, COMPONENT_ERR_APP_PORT_CONFLICT, err
	}
	// check other appsets from db
	isused, errorCode, err := isServicePortUsed(component.App)
	if err != nil {
		updateLog.Errorf("check port failed: %v", err)
		return nil, errorCode, err
	}
	if isused {
		err = errors.New("has service port conflict")
		updateLog.Errorf("%v", err)
		return nil, COMPONENT_ERR_APP_PORT_CONFLICT, err
	}
	// update app to appset
	_, err = updateAppInGroup(&appset.Group, component.App)
	if err != nil {
		updateLog.Errorf("update app in group failed, %v", err)
		return nil, COMMON_ERROR_INTERNAL, err
	}

	appset.ConvertPath()
	app, err := getAppFromGroup(&appset.Group, component.App.ID)
	if err != nil {
		updateLog.Errorf("failed to get app from group, %v", err)
		errorCode = COMMON_ERROR_INTERNAL
		return
	}
	// change each app's dependencies to the absolute path
	convertDependency(app)

	// updateLog.Debugf("app=%v", app)
	status, _, _, _ := getComponentStatus(app)
	// updateLog.Debugf("application=%v\napp=%v", application, app)
	if status != COMPONENT_STATUS_IDLE {
		updateLog.Infof("component is not IDLE, should update it on marathon")
		// update app in marathon
		// add ENVs, and put app to marathon.
		// here using a copy of app to avoid add env into db data.
		marathonApp := marathon.Application{}
		err = deepCopy(&marathonApp, app)
		if err != nil {
			updateLog.Errorf("deepCopy app failed, %v", err)
			errorCode = COMMON_ERROR_INTERNAL
			return
		}
		marathonApp.Instances = app.Instances
		refineEnv(&marathonApp, appset)

		// updateLog.Debugf("marathonApp=%v, app=%v", &marathonApp, app)
		// updateLog.Debugf("marathonApp=%v, app=%v", marathonApp, *app)
		// updateLog.Debugf("app post to marathon is %v", marathonApp)
		depId, err := common.UTIL.MarathonClient.UpdateApplication(&marathonApp, true)
		if err != nil {
			updateLog.Errorf("update app in marathon failed, %v", err)
			return nil, COMMON_ERROR_MARATHON, err
		}
		updateLog.Infof("update app on marathon finished")
		err = common.UTIL.MarathonClient.WaitOnDeployment(depId.DeploymentID, 5)
		if err != nil {
			updateLog.Errorf("waiton deployment %v failed, %v", depId.DeploymentID, err)
			updateLog.Errorf("but update is success, so return success.")
			err = nil
		}
	}
	// save appset to db
	GetAppsetService().syncToDB(&appset)

	appRes := appset.GetAllApps()
	for _, appRe := range appRes {
		env := appRe.Env
		if env != nil {
			alertEnable := (*env)["ALERT_ENABLE"]
			AppContainerIdEnv := (*env)["SCALED_APP_ID"]
			if alertEnable == "true" {
				scaleNum := (*env)["SCALE_NUMBER"]
				if scaleNum == "" {
					scaleNum = "1"
				}
				InsMaxNum := (*env)["INSTANCE_MAX_NUM"]
				InsMinNum := (*env)["INSTANCE_MIN_NUM"]
				var appContainerId string
				if AppContainerIdEnv == "" {
					appContainerId = appRe.ID
				} else {
					appContainerId = AppContainerIdEnv
				}
				repairpolicy := entity.RepairPolicy{}
				repairpolicy.RepairTemplateId = appset.Group.ID
				repairpolicy.ServiceGroupId = appset.Group.ID
				repairpolicy.AppCointainerId = appRe.ID
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

				query, query1, query2 := bson.M{}, bson.M{}, bson.M{}
				query["service_group_id"] = appset.Group.ID
				query["app_container_id"] = appRe.ID
				query["$and"] = []bson.M{query1, query2}
				queryStruct := dao.QueryStruct{"repairPolicy", query, 0, 0, ""}
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
				_, _, errR := GetRepairPolicyService().Create(repairpolicy, "")
				if errR != nil {
					logrus.Errorf("create repairpolicy err is %v", errR)
				}
			}
		}

	}

	updateLog.Infof("update app in db finished")
	status, _, _, _ = getComponentStatus(app)
	comp := convertAppToComponent(app, status)
	newComponent = &comp
	return
}

func (c *ComponentService) getAppwithSet(appID string) (
	app *marathon.Application, appset entity.Appset, errorCode string, err error) {
	// componentName must be absolute path
	appsetName, errorCode, err := getRootGroupName(appID)
	if err != nil {
		return
	}
	// get appset
	appset, errorCode, err = GetAppsetService().queryByName(appsetName)
	if err != nil {
		return
	}

	// get app from appset
	appset.ConvertPath()
	app, err = getAppFromGroup(&appset.Group, appID)
	if err != nil {
		errorCode = COMMON_ERROR_INTERNAL
		return
	}
	return
}

func checkAndInjectUriComponent(component *entity.ComponentViewObj) (err error) {
	// get all registry url
	registryList, err := getRegistryList()
	if err != nil {
		logrus.Errorf("get registry list error: %v", err)
		return
	}
	checkAndInjectUri(&component.App, registryList)
	return
}
