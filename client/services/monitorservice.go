package services

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

var (
	monitorService *MonitorService = nil
	onceMonitor    sync.Once
)

type MonitorService struct {
}

func GetMonitorService() *MonitorService {
	onceMonitor.Do(func() {
		logrus.Debugf("Once called from monitorService ......................................")
	})
	return monitorService
}

//func (p *MonitorService) GetAllContainers() (tasks []entity.Task, err error) {

//	return
//}
