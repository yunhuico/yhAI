package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	marathon "github.com/LinkerNetworks/go-marathon"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"

	"github.com/Sirupsen/logrus"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"

	COMMON_ERROR_MARATHON = "E12005" // call marathon failed.
	COMMON_ERROR_DB       = "E12006" // db operate failed.
)

const (
	PATH_DOCKER_CONFIG_JSON = "/root/.docker/config.json"
	URI_DOCKER_TAR_GZ       = "file:///var/lib/mesos/docker.tar.gz"
)

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "does not exist")
}

/**
 * return the root group name of an app
 *      appID:			the id of the app (absolute path)
 */
func getRootGroupName(appID string) (
	appPath string, errorCode string, err error) {
	if !strings.HasPrefix(appID, "/") {
		return "", COMPONENT_ERR_APP_ABSOLUTE_ID, errors.New("appID must be absolute path.")
	}
	appPaths := strings.Split(appID, "/")
	if len(appPaths) < 3 {
		return "", COMPONENT_ERR_APP_INVALID, errors.New("Invalid appID")
	}
	return appPaths[1], "", nil
}

/**
 * return the parent group name of an app
 *      appID:			the id of the app (absolute path)
 *		example: getParent("/iot/server/server")="/iot/server/"
 */
func getParent(appID string) (
	appPath string, errorCode string, err error) {
	if !strings.HasPrefix(appID, "/") {
		return "", COMPONENT_ERR_APP_ABSOLUTE_ID, errors.New("appID must be absolute path.")
	}
	appPaths := strings.Split(appID, "/")
	if len(appPaths) < 3 {
		return "", COMPONENT_ERR_APP_INVALID, errors.New("Invalid appID")
	}
	appPaths[len(appPaths)-1] = ""
	return strings.Join(appPaths, "/"), "", nil
}

/**
 * return the absolute appId from taskId
 *      taskId:			the id of the task
 *						format is <groupId>[_<subGroupId>]_<appId>.uuid
 */
func getAppIdFromTaskId(taskId string) string {
	withoutUUIDs := strings.Split(taskId, ".")
	withoutUUID := strings.Join(withoutUUIDs[:len(withoutUUIDs)-1], ".")
	appId := strings.Join(append([]string{""}, strings.Split(withoutUUID, "_")...), "/")
	return appId
}

/**
 * check wheter the app or group is belongs to the group
 *      id:			the id of the app or group
 *      groupId:	the absolute path of the group
 */
func isBelongsToGroup(id, groupId string) bool {
	// make sure groupId is the absolute path
	if !strings.HasPrefix(groupId, "/") {
		groupId = "/" + groupId
	}
	if strings.HasPrefix(id, "/") {
		// if id is the absolute path, it must start with groupId
		return strings.HasPrefix(id, groupId)
	} else {
		// if the id is not the absolute path, means it belongs to the group.
		return true
	}
}

/**
 * insert an app into the group, consider both relative path and absolute path
 * 		group:			the group you want to modify
 *		application:	the app you want to insert into the group
 */
func insertAppIntoGroup(group *marathon.Group, application marathon.Application) (
	bool, error) {
	if !strings.HasPrefix(application.ID, "/") {
		// appID is relative path
		if strings.Contains(application.ID, "/") {
			// appID is like a/b/c...
			appPath := strings.Split(application.ID, "/")
			for _, subGroup := range group.Groups {
				_subGroupId := subGroup.ID
				if strings.Contains(subGroup.ID, "/") {
					_subGroupIds := strings.Split(subGroup.ID, "/")
					_subGroupId = _subGroupIds[len(_subGroupIds)-1]
				}
				if _subGroupId == appPath[0] {
					application.ID = strings.Join(appPath[1:], "/")
					return insertAppIntoGroup(subGroup, application)
				}
			}
			subGroup := marathon.Group{ID: appPath[0]}
			group.Groups = append(group.Groups, &subGroup)
			application.ID = strings.Join(appPath[1:], "/")
			return insertAppIntoGroup(&subGroup, application)
			// return false, errors.New("No matched app path.")
		} else {
			// insert it to group.Apps
			group.App(&application)
			return true, nil
		}
	} else {
		// appID is the absolute path, find it from full pathes
		// appPath[0]="", appPath[1]=groupID.... appPath[len(appPath)]=appId
		appPath := strings.Split(application.ID, "/")
		if len(appPath) < 3 {
			return false, errors.New("invalid appID")
		} else if len(appPath) == 3 {
			if appPath[1] == strings.Trim(group.ID, "/") {
				// insert it to group.Apps
				group.App(&application)
				return true, nil
			}
			return false, errors.New("No matched app path.")
		} else {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for _, subGroup := range group.Groups {
					_subGroupId := subGroup.ID
					if strings.Contains(subGroup.ID, "/") {
						_subGroupIds := strings.Split(subGroup.ID, "/")
						_subGroupId = _subGroupIds[len(_subGroupIds)-1]
					}
					if _subGroupId == appPath[2] {
						application.ID = strings.Join(appPath[3:], "/")
						return insertAppIntoGroup(subGroup, application)
					}
				}
				subGroup := marathon.Group{ID: appPath[2]}
				group.Groups = append(group.Groups, &subGroup)
				application.ID = strings.Join(appPath[3:], "/")
				return insertAppIntoGroup(&subGroup, application)
			}
			return false, errors.New("No matched app path.")
		}
	}
}

/**
 * get app object from one group
 * 		group:		the group you want to loop
 * 		appID:		the path of the app you want to get
 */
func getAppFromGroup(group *marathon.Group, appID string) (
	*marathon.Application, error) {
	// logrus.Debugf("find app[%v] from [%v]", appID, group.ID)
	// logrus.Debugf("group=%v", group)
	if !strings.HasPrefix(appID, "/") {
		// appID is relative path
		if strings.Contains(appID, "/") {
			// appID is like a/b/c...
			appPath := strings.Split(appID, "/")
			for _, subGroup := range group.Groups {
				_subGroupId := subGroup.ID
				if strings.Contains(subGroup.ID, "/") {
					_subGroupIds := strings.Split(subGroup.ID, "/")
					_subGroupId = _subGroupIds[len(_subGroupIds)-1]
				}
				if _subGroupId == appPath[0] {
					return getAppFromGroup(subGroup, strings.Join(appPath[1:], "/"))
				}
			}
		} else {
			// find it only from root group
			for _, app := range group.Apps {
				_appID := app.ID
				if strings.Contains(app.ID, "/") {
					_appIDs := strings.Split(app.ID, "/")
					_appID = _appIDs[len(_appIDs)-1]
				}
				if _appID == appID {
					return app, nil
				}
			}
		}
		return nil, errors.New("not found")
	} else {
		// appID is the absolute path, find it from full pathes
		// appPath[0]="", appPath[1]=groupID.... appPath[len(appPath)]=appId
		appPath := strings.Split(appID, "/")
		if len(appPath) < 3 {
			return nil, errors.New("invalid appID")
		} else if len(appPath) == 3 {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for _, app := range group.Apps {
					_appID := app.ID
					if strings.Contains(app.ID, "/") {
						_appIDs := strings.Split(app.ID, "/")
						_appID = _appIDs[len(_appIDs)-1]
					}
					// if _appID == appID {
					if _appID == appPath[2] {
						return app, nil
					}
				}
			}
			return nil, errors.New("not found")
		} else {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for _, subGroup := range group.Groups {
					_subGroupId := subGroup.ID
					if strings.Contains(subGroup.ID, "/") {
						_subGroupIds := strings.Split(subGroup.ID, "/")
						_subGroupId = _subGroupIds[len(_subGroupIds)-1]
					}
					if _subGroupId == appPath[2] {
						return getAppFromGroup(subGroup, strings.Join(appPath[3:], "/"))
					}
				}
			}
			return nil, errors.New("not found")
		}
	}
}

/**
 * update app object in one group
 * 		group:			the group you want to update
 * 		application:	the app you want to update
 */
func updateAppInGroup(group *marathon.Group, application marathon.Application) (
	bool, error) {
	if !strings.HasPrefix(application.ID, "/") {
		// appID is relative path
		if strings.Contains(application.ID, "/") {
			// appID is like a/b/c...
			appPath := strings.Split(application.ID, "/")
			for _, subGroup := range group.Groups {
				if subGroup.ID == appPath[0] {
					application.ID = strings.Join(appPath[1:], "/")
					return updateAppInGroup(subGroup, application)
				}
			}
			return false, errors.New("No matched app.")
		} else {
			// update it from group.Apps
			for i, app := range group.Apps {
				_appID := app.ID
				if strings.Contains(app.ID, "/") {
					_appIDs := strings.Split(app.ID, "/")
					_appID = _appIDs[len(_appIDs)-1]
				}
				if _appID == application.ID {
					group.Apps[i] = &application
					return true, nil
				}
			}
			return false, errors.New("No matched app.")
		}
	} else {
		// appID is the absolute path, find it from full pathes
		// appPath[0]="", appPath[1]=groupID.... appPath[len(appPath)]=appId
		appPath := strings.Split(application.ID, "/")
		if len(appPath) < 3 {
			return false, errors.New("invalid appID")
		} else if len(appPath) == 3 {
			if appPath[1] == strings.Trim(group.ID, "/") {
				// find it in group.Apps
				for i, app := range group.Apps {
					_appID := app.ID
					if strings.Contains(app.ID, "/") {
						_appIDs := strings.Split(app.ID, "/")
						_appID = _appIDs[len(_appIDs)-1]
					}
					if _appID == appPath[2] {
						application.ID = appPath[2]
						group.Apps[i] = &application
						return true, nil
					}
				}
			}
			return false, errors.New("No matched app.")
		} else {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for _, subGroup := range group.Groups {
					if subGroup.ID == appPath[2] {
						application.ID = strings.Join(appPath[3:], "/")
						return updateAppInGroup(subGroup, application)
					}
				}

			}
			return false, errors.New("No matched app.")
		}
	}
}

/**
 * delete app by appID from a group
 *		group: 		the group you want to modify
 *		appID:		the id of the app you want to remove
 */
func deleteAppFromGroup(group *marathon.Group, appID string) (bool, error) {
	logrus.Debugf("delete app[%v] from [%v]", appID, group.ID)
	if !strings.HasPrefix(appID, "/") {
		// appID is relative path
		if strings.Contains(appID, "/") {
			// appID is like a/b/c...
			appPath := strings.Split(appID, "/")
			for _, subGroup := range group.Groups {
				_subGroupId := subGroup.ID
				if strings.Contains(subGroup.ID, "/") {
					_subGroupIds := strings.Split(subGroup.ID, "/")
					_subGroupId = _subGroupIds[len(_subGroupIds)-1]
				}
				if _subGroupId == appPath[0] {
					return deleteAppFromGroup(subGroup, strings.Join(appPath[1:], "/"))
				}
			}
		} else {
			// find it only from root group
			for i, app := range group.Apps {
				_appID := app.ID
				if strings.Contains(app.ID, "/") {
					_appIDs := strings.Split(app.ID, "/")
					_appID = _appIDs[len(_appIDs)-1]
				}
				if _appID == appID {
					group.Apps = append(group.Apps[:i], group.Apps[i+1:]...)
					// delete group.Apps[i]
					// return app, nil
					return true, nil
				}
			}
		}
		return false, errors.New("not found")
	} else {
		// appID is the absolute path, find it from full pathes
		// appPath[0]="", appPath[1]=groupID.... appPath[len(appPath)]=appId
		appPath := strings.Split(appID, "/")
		if len(appPath) < 3 {
			return false, errors.New("invalid appID")
		} else if len(appPath) == 3 {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for i, app := range group.Apps {
					_appID := app.ID
					if strings.Contains(app.ID, "/") {
						_appIDs := strings.Split(app.ID, "/")
						_appID = _appIDs[len(_appIDs)-1]
					}
					if _appID == appPath[2] {
						group.Apps = append(group.Apps[:i], group.Apps[i+1:]...)
						return true, nil
					}
				}
			}
			return false, errors.New("not found")
		} else {
			if appPath[1] == strings.Trim(group.ID, "/") {
				for _, subGroup := range group.Groups {
					_subGroupId := subGroup.ID
					if strings.Contains(subGroup.ID, "/") {
						_subGroupIds := strings.Split(subGroup.ID, "/")
						_subGroupId = _subGroupIds[len(_subGroupIds)-1]
					}
					if _subGroupId == appPath[2] {
						return deleteAppFromGroup(subGroup, strings.Join(appPath[3:], "/"))
					}
				}
			}
			return false, errors.New("not found")
		}
	}
}

/**
 *
 */
func GenAppSetDetail(appset entity.Appset, skipGroup, displayMonitor bool) *entity.AppsetDetailViewObj {
	appsetDetail := entity.AppsetDetailViewObj{
		ObjectId:      appset.ObjectId,
		Name:          appset.Name,
		Status:        appset.Status,
		Description:   appset.Description,
		TemplateId:    appset.TemplateId,
		CreatedByJson: appset.CreatedByJson,
		TimeDeployed:  appset.TimeDeployed,
		TimeCreate:    appset.TimeCreate,
		TimeUpdate:    appset.TimeUpdate,
		Components:    []entity.ComponentViewObj{},
	}

	if !skipGroup {
		appsetDetail.Group = appset.Group
	}

	status, components, _, _ := getAppsetStatus(appset)
	appsetDetail.Status = status
	for _, component := range components {
		if component == nil {
			continue
		}
		appsetDetail.TotalContainer += *component.App.Instances
		appsetDetail.TotalCpu += component.TotalCpu
		appsetDetail.TotalMem += component.TotalMem
		appsetDetail.TotalGpu += component.TotalGpu
		if component.Status != COMPONENT_STATUS_IDLE && component.Status != COMPONENT_STATUS_UNKNOWN {
			appsetDetail.RunningContainer += component.App.TasksRunning
			appsetDetail.RunningCpu += (component.App.CPUs * float64(component.App.TasksRunning))
			appsetDetail.RunningMem += (*component.App.Mem * float64(component.App.TasksRunning))
			appsetDetail.RunningGpu += (*component.App.GPUs * float64(component.App.TasksRunning))
		}
		tasks := component.App.Tasks
		wg := &sync.WaitGroup{}
		kvCh := make(chan *[2]string, len(tasks))
		errCh := make(chan *error, len(tasks))
		for _, t := range tasks {
			wg.Add(1)
			// call cAdvisor API
			// GET /api/linker/dockerid?taskid=123
			// example resp: ebd123f2-2675-4087-828b-420cad85ae8d
			//
			// container name: mesos-{task.SlaveID}.ebd123f2-2675-4087-828b-420cad85ae8d

			// fmt.Printf("%+v\n", t)
			go getMonitorURL(t.Host, t.ID, t.SlaveID, wg, kvCh, errCh)
		}
		go func() {
			wg.Add(1)
			defer wg.Done()

			m := make(map[string]string)
			for i := 0; i < len(tasks); i++ {
				select {
				case kv, ok := <-kvCh:
					if ok {
						k := kv[0]
						v := kv[1]
						m[k] = v
						// fmt.Printf("k: %s, v: %s\n", k, v)
					}
				case e, ok := <-errCh:
					if ok {
						logrus.Errorf("get container name error: %v", *e)
					}
				}
			}
			component.MonitorURLMap = m
		}()
		defer close(kvCh)
		defer close(errCh)
		wg.Wait()

		appsetDetail.Components = append(appsetDetail.Components, *component)
	}
	return &appsetDetail
}

func getMonitorURL(cadvisorHost, taskID, slaveID string, wg *sync.WaitGroup, kvCh chan *[2]string, errCh chan *error) {
	defer wg.Done()

	port := "10000"
	schema := "/api/linker/dockerid"

	baseURL := fmt.Sprintf("http://%s:%s%s", cadvisorHost, port, schema)

	u, err := url.Parse(baseURL)
	if err != nil {
		// log.Printf("parse %s to url error: %v\n", baseURL, err)
		errCh <- &err
		return
	}
	q := u.Query()
	q.Set("taskid", taskID)
	u.RawQuery = q.Encode()

	fullURL := u.String()
	// fmt.Printf("fullURL: %s\n", fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		// log.Printf("call cadvisor to get container name error: %v\n", err)
		errCh <- &err
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// log.Printf("read body error: %v\n", err)
		errCh <- &err
		return
	}

	containerName := fmt.Sprintf("mesos-%s.%s", slaveID, string(data))
	schema2 := "/api/v1.2/docker"
	// e.g. http://192.168.7.72:10000/api/v1.2/docker/mesos-bf2b618e-15f0-4c88-8193-120f29b5d7a0-S2.dc72bab0-3744-4a1a-a27c-f8810954c21b
	monitorURL := fmt.Sprintf("http://%s:%s%s/%s", cadvisorHost, port, schema2, containerName)
	kv := [2]string{taskID, monitorURL}
	kvCh <- &kv
	return
}

/**
 *
 */
func convertAppToComponent(app *marathon.Application, status string) entity.ComponentViewObj {
	appsetName, _, _ := getRootGroupName(app.ID)
	component := entity.ComponentViewObj{
		AppsetName: appsetName,
		Status:     status}
	switch status {
	case COMPONENT_STATUS_IDLE:
		component.CanStart = true
		component.CanDelete = true
		component.CanStop = false
		component.CanModify = true
		component.CanScale = false
		component.CanShell = false
	case COMPONENT_STATUS_RUNNING:
		component.CanStart = false
		component.CanDelete = true
		component.CanStop = true
		component.CanModify = true
		component.CanScale = true
		component.CanShell = true
	case COMPONENT_STATUS_FAILED:
		component.CanStart = false
		component.CanDelete = true
		component.CanStop = true
		component.CanModify = true
		component.CanScale = false
		component.CanShell = false
	case COMPONENT_STATUS_DEPLOYING:
		component.CanStart = false
		component.CanDelete = true
		component.CanStop = true
		component.CanModify = true
		component.CanScale = false
		component.CanShell = false
	case COMPONENT_STATUS_WAITING:
		component.CanStart = false
		component.CanDelete = true
		component.CanStop = true
		component.CanModify = true
		component.CanScale = true
		component.CanShell = false
	case COMPONENT_STATUS_SUSPENDED:
		component.CanStart = false
		component.CanDelete = true
		component.CanStop = true
		component.CanModify = true
		component.CanScale = true
		component.CanShell = false
	default:
		logrus.Warningf("unknown component status %v", status)
	}
	component.App = *app
	component.TotalCpu = app.CPUs * float64(*app.Instances)
	component.TotalMem = *app.Mem * float64(*app.Instances)
	component.TotalGpu = *app.GPUs * float64(*app.Instances)
	return component
}

/**
 * reference marathon group's status to appset status
 */
func getAppsetStatus(appset entity.Appset) (
	status string, components []*entity.ComponentViewObj, errorCode string, err error) {
	log := logrus.WithFields(logrus.Fields{
		"appset": appset.Name,
		"func":   "getAppsetStatus",
	})

	var finalStatus, tmpStatus, allStatus string
	components = []*entity.ComponentViewObj{}
	// here should check app in the appset, not in the marathon group
	apps := appset.GetAllApps()
	if len(apps) == 0 {
		finalStatus = APPSET_STATUS_IDLE
	} else {
		exist, err := common.UTIL.MarathonClient.HasGroup(appset.Group.ID)
		if err != nil {
			log.Errorf("check group exist on marathon failed, %v", err)
			finalStatus = APPSET_STATUS_UNKNOWN
			log.Infof("appset.Status=%v", finalStatus)
			return finalStatus, components, COMMON_ERROR_MARATHON, err
		}
		if !exist {
			finalStatus = APPSET_STATUS_IDLE
		}
	}
	if finalStatus == APPSET_STATUS_IDLE {
		for _, app := range apps {
			component := convertAppToComponent(app, COMPONENT_STATUS_IDLE)
			components = append(components, &component)
		}
		log.Infof("appset.Status=%v", finalStatus)
		return finalStatus, components, "", nil
	}

	// improve performance, decrease the requests to marathon
	group, err := common.UTIL.MarathonClient.GroupBy(appset.Group.ID,
		&marathon.GetGroupOpts{
			Embed: []string{
				"group.groups",
				"group.apps",
				"group.apps.tasks",
				"group.apps.counts",
				"group.apps.deployments",
				"group.apps.lastTaskFailure",
			},
		})
	if err != nil {
		log.Errorf("get group from marathon failed, %v", err)
		finalStatus = APPSET_STATUS_UNKNOWN
		log.Infof("appset.Status=%v", finalStatus)
		return finalStatus, components, COMMON_ERROR_MARATHON, err
	}
	marathonAppset := appset
	marathonAppset.Group = *group
	marathonApps := marathonAppset.GetAllApps()
	for _, app := range apps {
		// status, application, _, _ := getComponentStatus(app)
		var status string
		var application *marathon.Application
		if mapp := getApp(marathonApps, app); mapp != nil {
			status, application, _, _ = getComponentStatus(mapp, true)
			log.Debugf("group contains app[%v], app.Status=%v", mapp.ID, status)
		} else {
			// status, application, _, _ = getComponentStatus(app, false)
			status = COMPONENT_STATUS_IDLE
			application = app
			log.Debugf("group does not contain app[%v], app.Status=%v", app.ID, status)
		}
		switch {
		case status == COMPONENT_STATUS_UNKNOWN:
			log.Warningf("skip unknown component [%v]", app.ID)
		case status == COMPONENT_STATUS_FAILED:
			finalStatus = APPSET_STATUS_FAILED
		case status == COMPONENT_STATUS_WAITING:
			tmpStatus = APPSET_STATUS_WAITING
		case status == COMPONENT_STATUS_SUSPENDED:
			finalStatus = APPSET_STATUS_INCOMPLETE
		case status == COMPONENT_STATUS_IDLE || status == APPSET_STATUS_RUNNING:
			if allStatus == "" {
				allStatus = status
			} else if allStatus != status {
				allStatus = APPSET_STATUS_INCOMPLETE
			}
		case status == COMPONENT_STATUS_DEPLOYING:
			if tmpStatus == "" {
				tmpStatus = APPSET_STATUS_DEPLOYING
			}
		}
		component := convertAppToComponent(application, status)
		components = append(components, &component)
	}
	if finalStatus != "" {
		log.Infof("appset.Status=%v", finalStatus)
		return finalStatus, components, "", nil
	} else if tmpStatus != "" {
		log.Infof("appset.Status=%v", tmpStatus)
		return tmpStatus, components, "", nil
	} else {
		log.Infof("appset.Status=%v", allStatus)
		return allStatus, components, "", nil
	}
}

func getApp(apps []*marathon.Application,
	app *marathon.Application) *marathon.Application {
	for _, mApp := range apps {
		if app.ID == mApp.ID {
			return mApp
		}
	}
	return nil
}

/**
 * reference marathon app's status to component status.
 *         appId:	the absolute path of the app
 */
func getComponentStatus(app *marathon.Application, callMarathon ...bool) (
	status string, application *marathon.Application, errorCode string, err error) {
	log := logrus.WithFields(logrus.Fields{
		"app":          app,
		"callMarathon": callMarathon,
		"func":         "getComponentStatus",
	})
	status = COMPONENT_STATUS_UNKNOWN
	isCallMarathon := true
	if len(callMarathon) > 0 {
		isCallMarathon = callMarathon[0]
	}
	if !isCallMarathon {
		application = app
		// log.Infof("not call marathon, application=%v", application)
	} else {
		application, err = common.UTIL.MarathonClient.Application(app.ID)
		if err != nil {
			// log.Debugf("check app on marathon failed, %v", err)
			if isNotFound(err) {
				// app is not exist on marathon
				status = COMPONENT_STATUS_IDLE
				application = app
				return
			} else {
				// other error
				application = app
				errorCode = COMMON_ERROR_MARATHON
				return
			}
		}
		// log.Infof("call marathon, application=%v", application)
	}
	// Marathon Application Status Reference: https://mesosphere.github.io/marathon/docs/marathon-ui.html
	// do not change checking order below
	if isCallMarathon {
		queue, err := common.UTIL.MarathonClient.Queue()
		if err != nil {
			log.Printf("get marathon queue error: %v", err)
			return status, application, COMMON_ERROR_MARATHON, err
		}
		for _, item := range queue.Items {
			if item.Application.ID == application.ID {
				// WAITING
				if item.Delay.Overdue == true {
					status = COMPONENT_STATUS_WAITING
					return status, application, "", nil
				} else {
					// DELAYED
					status = COMPONENT_STATUS_FAILED
					return status, application, "", nil
				}
			}
		}
	}
	// DEPLOYING
	if len(application.Deployments) > 0 || application.TasksStaged > 0 {
		status = COMPONENT_STATUS_DEPLOYING
		return status, application, "", nil
	}
	// SUSPENDED
	if application.TasksRunning == 0 && *application.Instances == 0 {
		status = COMPONENT_STATUS_SUSPENDED
		return
	}
	// RUNNING
	if application.TasksRunning >= *application.Instances {
		status = COMPONENT_STATUS_RUNNING
		return status, application, "", nil
	}
	return status, application, "", nil
}

/**
 * for each app running on marathon, we should add below ENVs for monitor and auto-scaling
 *  ENV_APPSET_OBJ_ID, appset.ObjectId.Hex()
 * 	ENV_APPSET_GROUP_ID, appset.Group.Id
 * 	ENV_APPSET_APP_ID, group.Apps[i].Id
 * 	ENV_APPSET_TEMPLATE_ID, appset.TemplateId
 */
func refineEnv(app *marathon.Application, appset interface{}) {
	switch appset.(type) {
	case entity.Appset:
		app.AddEnv(ENV_APPSET_OBJ_ID, appset.(entity.Appset).ObjectId.Hex()).
			AddEnv(ENV_APPSET_GROUP_ID, appset.(entity.Appset).Group.ID).
			AddEnv(ENV_APPSET_APP_ID, app.ID).
			AddEnv(ENV_APPSET_TEMPLATE_ID, appset.(entity.Appset).TemplateId)
	default:
		return
	}
}

/**
 * for each app running on marathon, we should delete the dependency when we delete the application
 */
func delDependency(app *marathon.Application, deledApp interface{}) {
	switch deledApp.(type) {
	case marathon.Application:
		for i, dependency := range app.Dependencies {
			if dependency == deledApp.(marathon.Application).ID {
				// delete the dependency
				app.Dependencies = append(app.Dependencies[:i], app.Dependencies[i+1:]...)
			}
		}
	default:
		return
	}
}

/**
 * for each app running on marathon, we should delete the dependency when we delete the application
 */
func convertDependency(app *marathon.Application) {
	for i, dependency := range app.Dependencies {
		if !strings.HasPrefix(dependency, "/") {
			parentPath, _, _ := getParent(app.ID)
			app.Dependencies[i] = parentPath + dependency
		}
	}
}

func getErrorFromResponse(data []byte) (errorCode string, err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return COMMON_ERROR_INTERNAL, err
	}

	errorCode = resp.Error.Code
	err = errors.New(resp.Error.ErrorMsg)
	return
}

func getRetFromResponse(data []byte, obj interface{}) (err error) {
	err = json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	return
}

func getIdFromResponse(data []byte) (tokenId string, err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return "", err
	}

	json := resp.Data.(map[string]interface{})
	idobj := json["id"]
	if idobj == nil {
		logrus.Errorln("no id field")
		return "", errors.New("no id field in response!")
	}
	return idobj.(string), nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func exceptStatus(selector bson.M, status string) (newSelector bson.M) {
	newSelector = bson.M{}
	s := bson.M{}
	s["status"] = bson.M{"$ne": status}
	newSelector["$and"] = []bson.M{selector, s}
	return newSelector
}

//for debugging
func printPretty(v interface{}, mark string) (err error) {
	fmt.Printf("*********%s\n", mark)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	data = append(data, '\n')
	os.Stdout.Write(data)
	return
}

func deepCopy(dst, src interface{}) error {
	srcbytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(srcbytes, dst)
}

// has bug of copy pointer to 0, "", false, or []
//
// func deepCopy(dst, src interface{}) error {
// 	var buf bytes.Buffer
// 	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
// 		return err
// 	}
// 	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
// }

func getContainerViewObjByTask(task *marathon.Task) *entity.ContainerViewObj {
	taskLog := logrus.WithFields(logrus.Fields{
		"task":      task,
		"operation": "getContainerViewObjByTask",
	})
	container := entity.ContainerViewObj{
		Task: *task,
		Name: "UNKNOWN",
	}
	// for local test
	// endPoint := fmt.Sprintf("%s/cadvisor/%s/api/linker/dockerid?taskid=%s", "54.238.222.69", task.SlaveID, task.ID)

	endPoint := fmt.Sprintf("%s:10000/api/linker/dockerid?taskid=%s", task.Host, task.ID)
	taskLog.Debugf("will call api %v to get container name", endPoint)
	resp, err := httpclient.Http_get(endPoint, "",
		httpclient.Header{
			"Content-Type", "application/json",
		})
	if err != nil {
		logrus.Errorf("get container name from cadvisor failed, %v", err)
		return &container
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		nameBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			taskLog.Errorf("read container id from response failed, %v", err)
			return &container
		}
		containerName := nameBytes[:len(nameBytes)]
		taskLog.Debugf("received containerId=%v", string(containerName))
		container.Name = fmt.Sprintf("mesos-%s.%s", container.SlaveID, string(containerName))
		return &container
	} else {
		err = errors.New("failed to get container name from cadvisor")
		return &container
	}
}

func isServicePortUsed(app marathon.Application) (
	isUsed bool, errorCode string, err error) {
	portMappings := []marathon.PortMapping{}
	// if app has no portMapping, return true
	if app.Container != nil &&
		app.Container.Docker != nil &&
		app.Container.Docker.PortMappings != nil {

		portMappings = *app.Container.Docker.PortMappings
		if len(portMappings) == 0 {
			return false, "", nil
		}
	} else {
		return false, "", nil
	}

	allocatedServicePort := map[int]*int{}
	for _, portMapping := range portMappings {
		if portMapping.ServicePort != 0 {
			allocatedServicePort[portMapping.ServicePort] = &portMapping.ContainerPort
		}
	}
	// check every app of each appset, except app itself
	_, appsets, errorCode, err := GetAppsetService().getAll(0, 0, "")
	if err != nil {
		return
	}
	appsetOfApp, _, _ := getRootGroupName(app.ID)
	for _, appset := range appsets {
		// skip the appset which app is belongs to, to avoid update check failure
		if appset.Name == appsetOfApp {
			continue
		}
		if isServicePortUsedByAppset(app, appset) {
			return true, "", nil
		}
	}
	return false, "", nil
}

func isServicePortUsedByAppset(app marathon.Application, appset entity.Appset) bool {
	portMappings := []marathon.PortMapping{}
	// if app has no portMapping, return true
	if app.Container != nil &&
		app.Container.Docker != nil &&
		app.Container.Docker.PortMappings != nil {
		portMappings = *app.Container.Docker.PortMappings
		if len(portMappings) == 0 {
			return false
		}
	} else {
		return false
	}

	allocatedServicePort := map[int]*int{}
	for _, portMapping := range portMappings {
		if portMapping.ServicePort != 0 {
			allocatedServicePort[portMapping.ServicePort] = &portMapping.ContainerPort
		}
	}

	apps := appset.GetAllApps()
	for _, appInAppset := range apps {
		if appInAppset.ID != app.ID {
			if appInAppset.Container != nil &&
				appInAppset.Container.Docker != nil &&
				appInAppset.Container.Docker.PortMappings != nil {

				for _, portMappingInApp := range *appInAppset.Container.Docker.PortMappings {
					if allocatedServicePort[portMappingInApp.ServicePort] != nil {
						return true
					}
				}
			} else {
				continue
			}
		}
	}
	return false
}

// inject uri field in app json because marathon need it to auth, otherwise pulling image is impossible
// check if docker image belong to local regitries, if it is, inject uri in app json
func checkAndInjectUri(app *marathon.Application, registryList interface{}) {
	switch registryList.(type) {
	case entity.RegistryList:
		if isLocalRegistryImage(app.Container.Docker.Image, registryList.(entity.RegistryList).RegistryList) {
			app.AddUris(URI_DOCKER_TAR_GZ)
		}
	default:
		return
	}
}

// get registry list from config file
func getRegistryList() (registryList entity.RegistryList, err error) {
	registryList = entity.RegistryList{}
	// read docker config.json
	// dont worry about busy disk operation, io cache will work
	dockerAuthConfig, err := readDockerConfigJson(PATH_DOCKER_CONFIG_JSON)
	if err != nil {
		logrus.Errorf("read docker auth file error: %v", err)
		return
	}
	for key := range dockerAuthConfig.Auths {
		registryList.RegistryList = append(registryList.RegistryList, key)
	}
	return
}

// check if the image belongs to one of local registries
func isLocalRegistryImage(image string, localRegistryList []string) bool {
	for _, regUrl := range localRegistryList {
		if strings.Contains(image, regUrl) {
			return true
		}
	}
	return false
}

// read docker config.json and convert to entity
func readDockerConfigJson(path string) (config entity.DockerAuthConfig, err error) {
	if _, err = os.Stat(path); err != nil {
		logrus.Errorf("%s not exist: %v", path, err)
		return
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("read file error: %v", err)
		return
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		logrus.Errorf("parse file to entity error: %v", err)
		return
	}
	return
}
