package services

import (
	"sync"

	marathon "github.com/LinkerNetworks/go-marathon"
	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var (
	containerService     *ContainerService = nil
	onceContainerService sync.Once
)

type ContainerService struct {
	collectionName string
}

const ()

func GetContainerService() *ContainerService {
	onceContainerService.Do(func() {
		logrus.Debugf("Once called from containerService ......................................")
		containerService = &ContainerService{"component"}
	})
	return containerService
}

func (c *ContainerService) GetContainersByHost(host_ip string) (
	total int, containers []*entity.ContainerViewObj, errorCode string, err error) {
	containers = []*entity.ContainerViewObj{}

	_, allTasks, errorCode, err := c.getAllTasks()
	if err != nil {
		return
	}
	for _, task := range allTasks {
		if task.Host == host_ip {
			// call Cadvisor api to get containerName
			container := getContainerViewObjByTask(task)
			containers = append(containers, container)
		}
	}
	total = len(containers)
	return
}

// func (c *ContainerService) GetAllContainers() (
// 	total int, containers []*entity.ContainerViewObj, errorCode string, err error) {
// 	// return 0, nil, "", nil
// 	// get all appset from db
// 	_, appsets, _, err := GetAppsetService().List(0, 0, "name")
// 	if err != nil {
// 		errorCode = COMMON_ERROR_INTERNAL
// 		return
// 	}
// 	// for each appset, get its detail, and each app's tasks to the returned list
// 	for _, appset := range appsets {
// 		subTotal, subTasks, _, suberr := c.GetContainersByAppSet(appset.Name)
// 		if suberr != nil {
// 			// skip the failed appset
// 			continue
// 		}
// 		total += subTotal
// 		// logrus.Debugf("append tasks [%v]", subTasks)
// 		tasks = append(tasks, subTasks...)
// 	}
// 	return
// }

func (c *ContainerService) GetContainersByAppSet(appsetName string) (
	total int, containers []*entity.ContainerViewObj, errorCode string, err error) {
	containers = []*entity.ContainerViewObj{}
	total, tasks, errorCode, err := c.getTasksByAppSet(appsetName)
	if err != nil {
		return
	}
	// convert task to containerviewobj
	for _, task := range tasks {
		container := getContainerViewObjByTask(task)
		containers = append(containers, container)
	}
	return
}

func (c *ContainerService) getAllTasks() (
	total int, tasks []*marathon.Task, errorCode string, err error) {
	// return 0, nil, "", nil
	// get all appset from db
	_, appsets, _, err := GetAppsetService().List(0, 0, "name")
	if err != nil {
		errorCode = COMMON_ERROR_INTERNAL
		return
	}
	// for each appset, get its detail, and each app's tasks to the returned list
	for _, appset := range appsets {
		subTotal, subTasks, _, suberr := c.getTasksByAppSet(appset.Name)
		if suberr != nil {
			// skip the failed appset
			continue
		}
		total += subTotal
		// logrus.Debugf("append tasks [%v]", subTasks)
		tasks = append(tasks, subTasks...)
	}
	return
}

func (c *ContainerService) getTasksByAppSet(appsetName string) (
	total int, tasks []*marathon.Task, errorCode string, err error) {
	// return 0, nil, "", nil
	appsetDetail, _, err := GetAppsetService().GetDetail(appsetName, true, false)
	if err != nil {
		logrus.Errorf("get tasks of %v failed, %v", appsetName, err)
		errorCode = COMMON_ERROR_INTERNAL
		return
	}
	for _, component := range appsetDetail.Components {
		if component.Status != COMPONENT_STATUS_IDLE {
			// logrus.Debugf("append tasks [%v]", component.App.Tasks)
			tasks = append(tasks, component.App.Tasks...)
		}
	}
	total = len(tasks)
	return
}

func (c *ContainerService) Kill(taskID string, doscale bool) (
	errorCode string, err error) {
	killLog := logrus.WithFields(logrus.Fields{
		"taskId":    taskID,
		"doscale":   doscale,
		"operation": "killTask",
	})

	// if doscale is true, update appset
	// need to get appId from taskId
	// taskId format is <groupId>[_<subGroupId>]_<appId>.uuid
	if doscale {
		// appId is the absolute path
		appId := getAppIdFromTaskId(taskID)
		app, appset, errorCode, err := GetComponentService().getAppwithSet(appId)
		if err != nil {
			killLog.Errorf("get appset failed, %v", err)
			return errorCode, err
		}

		if *app.Instances == 1 {
			// stop the component
			killLog.Infof("will stop the component %v", app.ID)
			_, err = common.UTIL.MarathonClient.DeleteApplication(app.ID, true)
			if err != nil {
				killLog.Errorf("delete app from marathon failed, %v", err)
				return COMMON_ERROR_MARATHON, err
			}
		} else {
			killLog.Infof("will kill the task %v", taskID)
			// scalein the component
			errorCode, err = c.killTask(taskID, doscale)
			if err != nil {
				return errorCode, err
			}
			*app.Instances -= 1
			killLog.Infof("update appset")
			GetAppsetService().syncToDB(&appset)
		}
		return "", nil
	} else {
		return c.killTask(taskID, doscale)
	}
}

func (c *ContainerService) killTask(taskID string, doscale bool) (
	errorCode string, err error) {
	killLog := logrus.WithFields(logrus.Fields{
		"taskId":    taskID,
		"doscale":   doscale,
		"operation": "killTask",
	})

	_, err = common.UTIL.MarathonClient.KillTask(taskID, &marathon.KillTaskOpts{
		Scale: doscale,
		Force: true,
	})
	if err != nil {
		killLog.Errorf("kill task on marathon failed, %v", err)
		errorCode = COMMON_ERROR_MARATHON
	}

	return
}
