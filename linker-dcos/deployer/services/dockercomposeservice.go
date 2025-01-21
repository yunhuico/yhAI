package services

import (
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/deployer/command"
)

type DockerComposeService struct {
	serviceName string
}

const HOSTNAME_PREFIX string = "linker_hostname_"

var (
	dockerComposeService *DockerComposeService = nil
	onceDockerCompose    sync.Once
)

func GetDockerComposeService() *DockerComposeService {
	onceDockerCompose.Do(func() {
		logrus.Debugf("Once called from DockerComposeService ......................................")
		dockerComposeService = &DockerComposeService{"DockerComposeService"}
	})
	return dockerComposeService

}

func (p *DockerComposeService) Add(username string, clusterName string, privateNicName []string, addServer []entity.Server, curScale int, swarmMaster string, isHA bool, registries []entity.DockerRegistry) error {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clusterName + ""
	nodeList := []string{}
	for _, tmpHost := range addServer {
		strTmp := strings.Replace(tmpHost.Hostname, ".", "_", -1)
		strTmp = strings.Replace(strTmp, "-", "_", -1)
		tmpStr := HOSTNAME_PREFIX + strTmp + "=" + tmpHost.IpAddress
		nodeList = append(nodeList, tmpStr)
	}

	//get system component registry if exists
	imageRegistry := ""
	exist, registry := GetSystemRegistry(registries)
	if exist {
		imageRegistry = registry.Registry
	}

	return command.AddSlaveToCluster(username, clusterName, privateNicName, swarmMaster, storagePath, nodeList, curScale, isHA, imageRegistry)
}

func (p *DockerComposeService) Remove(username, clusterName, serverName string) error {
	strTmp := strings.Replace(serverName, ".", "_", -1)
	strTmp = strings.Replace(strTmp, "-", "_", -1)
	tmpStr := HOSTNAME_PREFIX + strTmp
	return command.RemoveSlaveFromCluster(username, clusterName, tmpStr)
}

func (p *DockerComposeService) Create(username string, clusterName string, privateNicName []string, allServers []entity.Server, scale int, isLinkerMgmt, isHA bool, registries []entity.DockerRegistry) error {
	nodeList := []string{}
	masterList := []string{}
	swarmMaster := ""
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clusterName + ""
	for _, tmpHost := range allServers {
		if tmpHost.IsMaster {
			swarmMaster = tmpHost.Hostname
		}
		if tmpHost.IsMaster {
			masterList = append(masterList, tmpHost.PrivateIpAddress)
		}
		// if tmpHost.IsSlave {
		strTmp := strings.Replace(tmpHost.Hostname, ".", "_", -1)
		strTmp = strings.Replace(strTmp, "-", "_", -1)
		tmpStr := HOSTNAME_PREFIX + strTmp + "=" + tmpHost.IpAddress
		nodeList = append(nodeList, tmpStr)
		// }
	}

	//get system component registry if exists
	imageRegistry := ""
	exist, registry := GetSystemRegistry(registries)
	if exist {
		imageRegistry = registry.Registry
	}
	return command.InstallCluster(username, clusterName, privateNicName, swarmMaster, storagePath, masterList, nodeList, scale, isLinkerMgmt, isHA, imageRegistry)
}

func (p *DockerComposeService) Restart(userName, clusterName, swarmMaster, serviceName, ymlFileName string) error {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + userName + "/" + clusterName + ""
	return command.RestartService(userName, clusterName, swarmMaster, serviceName, ymlFileName, storagePath)
}
