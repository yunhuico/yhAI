package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/jmoiron/jsonq"

	"github.com/Sirupsen/logrus"
	cmd "linkernetworks.com/dcos-backend/common/common"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/deployer/command"
	"linkernetworks.com/dcos-backend/deployer/common"

	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	marathon "github.com/LinkerNetworks/go-marathon"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/utils"
)

const (
	DEPLOY_STATUS_TERMINATED = "TERMINATED"
	DEPLOY_STATUS_DEPLOYED   = "RUNNING"
	DEPLOY_STATUS_DEPLOYING  = "DEPLOYING"
	DEPLOY_STATUS_FAILED     = "FAILED"

	DEPLOY_ERROR_CREATE string = "E60000"
	DEPLOY_ERROR_SCALE  string = "E60001"
	DEPLOY_ERROR_DELETE string = "E60002"
	DEPLOY_ERROR_QUERY  string = "E60003"

	DEPLOY_ERROR_CHANGE_HOST              string = "E60010"
	DEPLOY_ERROR_CHANGE_NAMESERVER        string = "E60011"
	DEPLOY_ERROR_CHANGE_DNSCONFIG         string = "E60012"
	DEPLOY_ERROR_COPY_CONFIG_FILE         string = "E60013"
	DEPLOY_ERROR_COPY_DOCKER_MACHINE_FILE string = "E60014"

	DEPLOY_ERROR_DELETE_NODE    string = "E60020"
	DEPLOY_ERROR_DELETE_CLUSTER string = "E60021"

	DEPLOY_ERROR_ADD_NODE_DOCKER_MACHINE string = "E60030"
	DEPLOY_ERROR_ADD_NODE_DOCKER_COMPOSE string = "E60031"
	DEPLOY_ERROR_CALLBACK                string = "E60032"
	DEPLOY_ERROR_CREATEZKSWARM           string = "E60033"
	DEPLOY_ERROR_NUMBER                  string = "E60034"
	COMPONENT_ERROR_CREATE_FILE          string = "E60035"
	COMPONENT_ERROR_GET_COMPONENTID      string = "E60036"

	DEPLOY_ERROR_DOCKERMACHINECREATE string = "installation_failed"
	DEPLOY_ERROR_DOCKERCOMPOSE       string = "systemComponent_failed"
	DEPLOY_ERROR_INSTALLCOMPONENT    string = "service_failed"

	MESOS_ATTRIBUTE_LABEL_PREFIX string = "LINKER_MESOS_ATTRIBUTE"
	MESOS_ATTRIBUTE_LB           string = "lb:enable;public_ip:true"

	CLUSTER_CATEGORY_COMPACT string = "compact"
	CLUSTER_CATEGORY_HA      string = "ha"

	MASTER_NODE_NUMBER_HA      int = 3
	MASTER_NODE_NUMBER_COMPACT int = 1

	MINMUM_NODE_NUMBER_HA      int = 5
	MINMUM_NODE_NUMBER_COMPACT int = 2

	GOOGLE_CREDENTIAL_FILE string = "/googlecredentials/service-account.json"
)

var (
	deployService *DeployService = nil
	onceDeploy    sync.Once
	RetryTime     = 100
)

type DeployService struct {
	serviceName string
}

func GetDeployService() *DeployService {
	onceDeploy.Do(func() {
		logrus.Debugf("Once called from DeployService ......................................")
		deployService = &DeployService{"DeployService"}
	})
	return deployService
}

func (p *DeployService) CreateCluster(request entity.Request) (err error) {
	createLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	createLog.Infof("start to Deploy Docker Cluster...")
	clustername := request.ClusterName

	category := request.CreateCategory
	if !strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) && !strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
		createLog.Warnf("cluster create category is invalid %s, will use defalut value %s", category, CLUSTER_CATEGORY_COMPACT)
		request.CreateCategory = CLUSTER_CATEGORY_COMPACT
	}

	//Call the Docker Machines to create machines, change /etc/hosts and Replace PubKey File
	servers, swarmServers, mgmtServers, slaveServers, _, _, err := dockerMachineCreateCluster(request)
	if err != nil {
		createLog.Errorf("Call docker-machine to create cluster failed , err is %v", err)

		// CALLBACK :docker machine create is err so callback to cluster to change cluster and host status to failed
		_, errC := CallbackCluster(clustername, request.UserName, false, "install", request.LogId, DEPLOY_ERROR_DOCKERMACHINECREATE, request.XAuthToken)
		if errC != nil {
			createLog.Errorf("callback to cluster is err")
		}

		_, errH := CallbackHost(clustername, request.UserName, false, "install", servers, request.XAuthToken)
		if errH != nil {
			createLog.Errorf("callback host is err")
		}

		return
	}

	count := request.MasterCount + request.SharedCount + request.PureSlaveCount
	clusterSlaveSize := count
	failedHostSize := clusterSlaveSize - len(servers)
	if request.IsLinkerMgmt == false {
		if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
			clusterSlaveSize = count - 1 - failedHostSize
		} else {
			clusterSlaveSize = count - 3 - failedHostSize
		}

	}

	//waiting 120s for swarming cluster startup and configuration
	//TODO need to be replaced by api check
	createLog.Debugf("waiting 120s.....")
	time.Sleep(120 * time.Second)

	isHA := strings.EqualFold(category, CLUSTER_CATEGORY_HA)

	createLog.Infof("start to Call Docker Compose")

	nodes := entity.GetRequestNodes(request)
	var NicName []string
	if len(nodes) > 0 {
		NicName = getAllPrivateNicName(nodes, swarmServers)
	}

	err = GetDockerComposeService().Create(request.UserName, request.ClusterName, NicName, swarmServers, clusterSlaveSize, request.IsLinkerMgmt, isHA, request.DockerRegistries)
	if err != nil {
		createLog.Errorf("Call docker-compose to create cluster failed , err is %v", err)
		_, errC := CallbackCluster(clustername, request.UserName, false, "install", request.LogId, DEPLOY_ERROR_DOCKERCOMPOSE, request.XAuthToken)
		if errC != nil {
			createLog.Errorf("callback to cluster is err")
		}

		_, errH := CallbackHost(clustername, request.UserName, false, "install", servers, request.XAuthToken)
		if errH != nil {
			createLog.Errorf("callback host is err")
		}

		return
	}

	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName

	//start mesos-agent on userCluster
	if !request.IsLinkerMgmt {
		bootUpMesosAgent(slaveServers, request.UserName, request.ClusterName)
	}

	//set universe registry for user cluster
	if !request.IsLinkerMgmt {
		configUniverseRegistry(servers, request.ClusterName, storagePath)
	}

	err = installComponent(servers, swarmServers, mgmtServers, request)
	if err != nil {
		_, err = CallbackCluster(clustername, request.UserName, false, "install", request.LogId, DEPLOY_ERROR_INSTALLCOMPONENT, request.XAuthToken)
		if err != nil {
			createLog.Errorf("callback to cluster is err")
		}

		_, err = CallbackHost(clustername, request.UserName, false, "install", servers, request.XAuthToken)
		if err != nil {
			createLog.Errorf("callback host is err")
		}

	}

	//callback cluster to change cluster and hosts status to running

	_, err = CallbackHost(clustername, request.UserName, true, "install", servers, request.XAuthToken)
	if err != nil {
		createLog.Errorf("callback host is err")
	}
	_, err = CallbackCluster(clustername, request.UserName, true, "install", request.LogId, "", request.XAuthToken)
	if err != nil {
		createLog.Errorf("callback to cluster is err")
	}

	createLog.Infof("callback servers is %v", servers)

	return
}

func getAllPrivateNicName(nodes []entity.Node, servers []entity.Server) (privateNicName []string) {
	for _, server := range servers {
		for _, node := range nodes {
			if server.IpAddress == node.IP {
				if node.PrivateNicName != "" {
					strTmp := strings.Replace(server.Hostname, ".", "_", -1)
					strTmp = strings.Replace(strTmp, "-", "_", -1)
					nicname := "ENNAME_" + strTmp + "=" + node.PrivateNicName
					privateNicName = append(privateNicName, nicname)
				} else {
					strTmp := strings.Replace(server.Hostname, ".", "_", -1)
					strTmp = strings.Replace(strTmp, "-", "_", -1)
					nicname := "ENNAME_" + strTmp + "=eth0"
					privateNicName = append(privateNicName, nicname)
				}
			}
		}
	}
	return
}

func bootUpMesosAgent(slaveServers []entity.Server, username, clustername string) {
	logrus.Infof("config and boot up mesos-agent process on each slave node")
	for _, slave := range slaveServers {
		hostname := slave.Hostname
		err := GetDockerMachineService().ConfigAndBootMesosAgent(hostname, username, clustername)
		if err != nil {
			logrus.Errorf("config and boot %s failed, error %v", hostname, err)
		}
	}
}

func configUniverseRegistry(servers []entity.Server, clustername, storagePath string) {
	log := logrus.WithFields(logrus.Fields{"clustername": clustername})
	log.Info("config universer registry certificate for cluster %s", clustername)

	cmd := "sudo mkdir -p /etc/docker/certs.d/master.mesos:5000 && sudo curl -o /etc/docker/certs.d/master.mesos:5000/ca.crt http://master.mesos:8082/certs/domain.crt"
	for _, server := range servers {
		_, _, err := command.ExecCommandOnMachine(server.Hostname, cmd, storagePath)
		if err != nil {
			log.Errorf("config host %s docker universe registry certificat error %v", server.Hostname, err)
		}
	}
}

func installComponent(servers, swarmServers, mgmtServers []entity.Server, request entity.Request) error {
	//get the first ip of mgmtServer as marathon Ip
	installLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	marathonEndpoint := fmt.Sprintf("%s/marathon", mgmtServers[0].IpAddress)
	installLog.Debugf("marathon endpoint is %s", marathonEndpoint)
	category := request.CreateCategory

	flag := waitForMarathonStartUp(marathonEndpoint)
	if !flag {
		installLog.Warnf("marathon failed to startup!")
		return errors.New("marathon failed to startup")
	}

	//get system component registry if exists
	imageRegistry := ""
	exist, registry := GetSystemRegistry(request.DockerRegistries)
	if exist {
		imageRegistry = registry.Registry
	}

	instance := getsharedInstanceInServer(servers)

	lbPayload := prepareLbJson(request.UserName, request.ClusterName, request.IsLinkerMgmt, category, imageRegistry, instance)

	if !request.IsLinkerMgmt {
		deploymentId, errC := GetMarathonService().CreateGroup(lbPayload, marathonEndpoint)
		if errC != nil {
			installLog.Warnf("create group to get deployment id id fail, err is %v", errC)
		}
		installLog.Infof("start to call marathon for lb with deployment Id: %s", deploymentId)
		flag, err := waitForMarathon(deploymentId, marathonEndpoint)
		if flag {
			if err != nil {
				installLog.Warnf("deploy the lb fail, err is %v", err)
			} else {
				installLog.Infof("deploy the lb finished successfully...")
			}
		} else {
			installLog.Warnf("deploy the lb fail because of timeout, err is %v", err)
		}
	}

	//Start create Linker Components
	if request.IsLinkerMgmt {
		//deploy marathonlb
		lbDeploymentId, errCG := GetMarathonService().CreateGroup(lbPayload, marathonEndpoint)
		if errCG != nil {
			installLog.Warnf("create group to get lb deployment is fail, err is %v", errCG)
		}
		installLog.Infof("start to call marathon for lb with deployment Id: %s", lbDeploymentId)
		flag, err := waitForMarathon(lbDeploymentId, marathonEndpoint)
		if flag {
			if err != nil {
				installLog.Errorf("deploy the lb fail, err is %v", err)
				return err
			} else {
				installLog.Infof("deploy the lb finished successfully...")
			}
		} else {
			installLog.Errorf("deploy the lb fail because of timeout, err is %v", err)
			return err
		}

		//call the marathon to deploy the linker service containers for Linker-Management Cluster
		payload := prepareLinkerComponents(mgmtServers, swarmServers[0], category, imageRegistry, request)
		deploymentId, _ := GetMarathonService().CreateGroup(payload, marathonEndpoint)
		flagDeploy, errDeploy := waitForMarathon(deploymentId, marathonEndpoint)
		if flagDeploy {
			if errDeploy != nil {
				installLog.Errorf("deploy the linker management components fail, err is %v", errDeploy)
				return errDeploy
			} else {
				installLog.Infof("deploy the linker management components finished successfully...")
			}
		} else {
			installLog.Errorf("deploy the linker management components fail because of timeout, err is %v", errDeploy)
			return errDeploy
		}
	}

	return nil
}

func prepareLinkerComponents(mgmtServers []entity.Server, swarmServer entity.Server, category string, imageRegistry string, request entity.Request) (payload []byte) {
	payload, err := ioutil.ReadFile("/linker/marathon/marathon-linkercomponents.json")

	payload = common.GenRegistry(imageRegistry, payload)

	if err != nil {
		logrus.Errorf("read linkercomponents.json failed, error is %v", err)
		return
	}

	// var serviceGroup *entity2.ServiceGroup
	var serviceGroup *marathon.Groups
	err = json.Unmarshal(payload, &serviceGroup)
	if err != nil {
		logrus.Errorf("Unmarshal linkercomponents.json failed, error is %v", err)
		return
	}

	mongoInstance := 3
	if category == CLUSTER_CATEGORY_COMPACT {
		mongoInstance = 1
	}

	mongoserverlist := ""
	var commandTextBuffer bytes.Buffer
	for index, server := range mgmtServers {
		commandTextBuffer.WriteString(server.PrivateIpAddress)
		if index != len(mgmtServers)-1 {
			commandTextBuffer.WriteString(",")
		}
	}

	mongoserverlist = commandTextBuffer.String()

	for j := range serviceGroup.Groups {
		group := serviceGroup.Groups[j]
		// Add constraints
		// There is no case for group embeded group.
		for i := range group.Apps {
			app := group.Apps[i]
			if app.Env != nil && (*app.Env)["MONGODB_NODES"] != "" {
				(*app.Env)["MONGODB_NODES"] = mongoserverlist
				for _, ser := range mgmtServers {
					nic := getNicAccIp(ser.IpAddress, request)
					logrus.Infof("nic name is %s ", nic)
					strTmp := strings.Replace(ser.Hostname, "-", "_", -1)
					strTmp = strings.Replace(strTmp, ".", "_", -1)
					envname := "ENNAME_" + strTmp
					logrus.Infof("envname is %s ", envname)
					(*app.Env)[envname] = nic
				}
			}

			if app.ID == "deployer" {
				constraint := []string{"hostname", "CLUSTER", swarmServer.PrivateIpAddress}
				// app.Constraints = [][]string{}
				// app.Constraints = append(app.Constraints, constraint)
				Constraints := [][]string{}
				Constraints = append(Constraints, constraint)
				app.Constraints = &Constraints

			}

			if app.ID == "mongodb" || app.ID == "ui" || app.ID == "clustermgmt" || app.ID == "usermgmt" || app.ID == "redis" {
				app.Instances = &mongoInstance
			}
		}
	}

	payload, err = json.Marshal(*serviceGroup)
	if err != nil {
		logrus.Errorf("Marshal linkercomponents err is %v", err)
		return
	}

	logrus.Debug("payload is : " + string(payload))

	return payload
}

func getNicAccIp(ip string, request entity.Request) (nicname string) {
	if request.CreateMode == "new" {
		logrus.Infof("using eth0 as default nic name for Iaas platform cluster node")
		nicname = "eth0"
	} else {
		nodes := entity.GetRequestNodes(request)
		for _, node := range nodes {
			if node.IP == ip {
				if node.PrivateNicName != "" {
					nicname = node.PrivateNicName
				} else {
					nicname = "eth0"
				}
			}
		}
	}

	return
}

func getsharedInstanceInServer(servers []entity.Server) (instance int) {
	if len(servers) == 0 {
		logrus.Errorf("there is no server")
		return
	}

	for _, server := range servers {
		if server.IsSharedServer {
			instance = instance + 1
		}
	}
	logrus.Infof("sharedserver in servers is %v", instance)
	return

}

func prepareLbJson(username, clustername string, isLinkerMgmt bool, category string, imageRegistry string, sharedinstance int) (payload []byte) {
	prepareLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	prepareLog.Info("Start to prepare the lb marathon json with linker management as %v", isLinkerMgmt)
	payload, err := ioutil.ReadFile("/linker/marathon/marathon-lb.json")

	//replace "{registry}" with imageRegistry
	payload = common.GenRegistry(imageRegistry, payload)

	constraint := []string{"lb", "CLUSTER", "enable"}
	constraintUnique := []string{"hostname", "UNIQUE"}

	if err != nil {
		prepareLog.Errorf("read marathon lb.json failed, error is %v", err)
		return
	}

	// var serviceGroup *entity2.ServiceGroup
	var serviceGroup *marathon.Groups
	err = json.Unmarshal(payload, &serviceGroup)
	if err != nil {
		prepareLog.Errorf("Unmarshal marathon lb.json failed, error is %v", err)
		return
	}

	instanceNum := 1
	if isLinkerMgmt {
		if category == CLUSTER_CATEGORY_COMPACT {
			instanceNum = 1
		} else if category == CLUSTER_CATEGORY_HA {
			instanceNum = 3
		} else {
			prepareLog.Warnf("not supported cluster category %s, will create 1 lb", category)
			instanceNum = 1
		}
	} else {
		if category == CLUSTER_CATEGORY_COMPACT {
			instanceNum = 1
		} else if category == CLUSTER_CATEGORY_HA {
			instanceNum = 2
		}
	}

	if sharedinstance > instanceNum {
		instanceNum = sharedinstance
	}

	for j := range serviceGroup.Groups {
		group := serviceGroup.Groups[j]
		// Add constraints
		// There is no case for group embeded group.
		for i := range group.Apps {
			app := group.Apps[i]
			prepareLog.Infof("Add constraint to LB for app:%s and constaint: %v", app.ID, constraint)
			// app.Constraints = [][]string{}
			// app.Constraints = append(app.Constraints, constraint)
			// app.Constraints = append(app.Constraints, constraintUnique)
			Constraints := [][]string{}
			Constraints = append(Constraints, constraint)
			Constraints = append(Constraints, constraintUnique)
			app.Constraints = &Constraints
			if app.ID == "marathonlb" {
				prepareLog.Infof("Change instance number of LB for app:%s to %s", app.ID, instanceNum)
				app.Instances = &instanceNum
			}
		}
	}

	payload, err = json.Marshal(*serviceGroup)
	if err != nil {
		prepareLog.Errorf("Marshal marathon lb err is %v", err)
		return
	}

	return
}

func waitForMarathonStartUp(marathonEndpoint string) (flag bool) {
	flag = false
	logrus.Debugf("check and wait marathon service available, endpoint %s", marathonEndpoint)
	for i := 0; i < RetryTime; i++ {
		flag = GetMarathonService().IsServiceAvaiable(marathonEndpoint)
		if flag {
			return
		} else {
			time.Sleep(10000 * time.Millisecond)
		}
	}
	return
}

func waitForMarathon(deploymentId, marathonEndpoint string) (flag bool, err error) {
	flag = false
	logrus.Debugf("check status with deploymentId [%v]", deploymentId)
	for i := 0; i < RetryTime; i++ {
		// get lock by service group instance id
		flag, err = GetMarathonService().IsDeploymentDone(deploymentId, marathonEndpoint)
		if err != nil {
			continue
		}
		if flag {
			return
		} else {
			time.Sleep(30000 * time.Millisecond)
		}
	}
	return false, errors.New("deployment has not been done!")
}

func changeDnsConfig(privateIP string, mgmtServers []entity.Server, storagePath string) (err error) {
	dat, err := ioutil.ReadFile(storagePath + "/dns-config.json")
	if err != nil {
		logrus.Errorf("read dns-config.json failed, error is %v", err)
		return
	}

	var dnsconfig *entity.DnsConfig
	err = json.Unmarshal(dat, &dnsconfig)

	if err != nil {
		logrus.Errorf("Unmarshal DnsConfig err is %v", err)
		return
	}

	//make zookeeper string
	var commandZkBuffer bytes.Buffer
	masterGroup := []string{}
	commandZkBuffer.WriteString("zk://")
	for i, server := range mgmtServers {
		commandZkBuffer.WriteString(server.PrivateIpAddress)
		commandZkBuffer.WriteString(":2181")
		if i < (len(mgmtServers) - 1) {
			commandZkBuffer.WriteString(",")
		}
		var commandMasterBuffer bytes.Buffer
		commandMasterBuffer.WriteString(server.PrivateIpAddress)
		commandMasterBuffer.WriteString(":5050")
		masterGroup = append(masterGroup, commandMasterBuffer.String())
	}
	commandZkBuffer.WriteString("/mesos")

	//make zookeeper
	dnsconfig.Zookeeper = commandZkBuffer.String()

	//make masters
	dnsconfig.Masters = append(masterGroup)

	//make listener
	dnsconfig.Listener = privateIP

	//write back to file
	jsonresult, err := json.Marshal(*dnsconfig)
	if err != nil {
		logrus.Errorf("Marshal DnsConfig err is %v", err)
		return
	}

	err = ioutil.WriteFile(storagePath+"/config.json", jsonresult, 0666)
	if err != nil {
		logrus.Errorf("write config.json failed, error is %v", err)
		return
	}
	return
}

func copyDockerMachineFiles(mgmtServers []entity.Server, storagePath, userPath string) (err error) {
	clustername := GetClusterName(storagePath)
	copyLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	for _, server := range mgmtServers {
		commandStr := fmt.Sprintf("sudo mkdir -p %s && sudo chown -R %s:%s /linker", userPath, server.SshUser, server.SshUser)
		_, _, err = command.ExecCommandOnMachine(server.Hostname, commandStr, storagePath)
		if err != nil {
			copyLog.Errorf("mkdir user path for docker machine failed: %v", err)
			return
		}
		_, _, err = command.ScpFolderToMachine(server.Hostname, storagePath, userPath, storagePath)
		if err != nil {
			copyLog.Errorf("copy docker machine files to target server %s fail, err is %v", server.Hostname, err)
			return
		}
	}
	return
}

func changeNameserver(servers, dnsServers []entity.Server, storagePath string, isLinkerMgmt bool) (err error) {
	clustername := GetClusterName(storagePath)
	changeLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	for _, server := range servers {
		if server.IsMaster {
			commandStr := fmt.Sprintf("sudo sed -i '1inameserver %s' /etc/resolv.conf", server.PrivateIpAddress)
			_, _, err = command.ExecCommandOnMachine(server.Hostname, commandStr, storagePath)
			if err != nil {
				changeLog.Errorf("change name server failed for server [%v], error is %v", server.PrivateIpAddress, err)
				return
			}
		} else {
			commandStr := fmt.Sprintf("sudo sed -i '1inameserver %s' /etc/resolv.conf", dnsServers[0].PrivateIpAddress)
			_, _, err = command.ExecCommandOnMachine(server.Hostname, commandStr, storagePath)
			if err != nil {
				changeLog.Errorf("change name server failed for server [%v], error is %v", dnsServers[0].PrivateIpAddress, err)
				return
			}
		}
	}
	return
}

func dockerMachineCreateCluster(request entity.Request) (servers, swarmServers, mgmtServers, slaveServers, monitorServers []entity.Server, errorCode string, err error) {
	dockerCLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	if request.CreateMode == "new" {
		servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, err = dockerMachineIaasCreateCluster(request)
	} else if request.CreateMode == "reuse" {
		servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, err = dockerMachineGenericCreateCluster(request)
	} else {
		dockerCLog.Errorf("not supported create mode %s", request.CreateMode)
		err = errors.New("not supported create mode for deployment!")
		return
	}
	if err != nil {
		dockerCLog.Errorf("docker-machine create machine error %v", err)
		return
	}

	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName

	//copy the dns config file to target dns server
	dockerCLog.Infof("start to config and copy the dns config file to target dns server")
	//1. copy dns config file to storage path folder
	//remove the file then copy, does not use "cp -n" due to this parameter may not exist in some os
	cmd.ExecCommand("rm -f " + storagePath + "/dns-config.json && cp /linker/config/dns-config.json " + storagePath + "/dns-config.json")
	//2. config and cp dns-config.json to each dns server
	for _, server := range mgmtServers {
		err = changeDnsConfig(server.PrivateIpAddress, mgmtServers, storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_CHANGE_DNSCONFIG
			dockerCLog.Errorf("change a dns config failed, err is %v", err)
			return
		}

		commandStr := fmt.Sprintf("sudo mkdir -p /linker/config && sudo chown -R %s:%s /linker", server.SshUser, server.SshUser)
		_, _, err = command.ExecCommandOnMachine(server.Hostname, commandStr, storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("mkdir /linker/config failed when copy dns config file: %v", err)
			return
		}
		_, _, err = command.ScpToMachine(server.Hostname, storagePath+"/config.json", "/linker/config/config.json", storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("copy dns config file to target server %s fail, err is %v", server.Hostname, err)
			return
		}
	}

	//Change "/etc/resolve.conf"
	dockerCLog.Infof("start to change name server for all nodes")
	err = changeNameserver(swarmServers, mgmtServers, storagePath, request.IsLinkerMgmt)
	if err != nil {
		errorCode = DEPLOY_ERROR_CHANGE_NAMESERVER
		return
	}

	//execute the replacement of the cadvisor server
	var commandCAdvisorBuffer bytes.Buffer
	for i, server := range slaveServers {
		commandCAdvisorBuffer.WriteString("'")
		commandCAdvisorBuffer.WriteString(server.PrivateIpAddress)
		commandCAdvisorBuffer.WriteString(":10000")
		commandCAdvisorBuffer.WriteString("'")
		if i < (len(slaveServers) - 1) {
			commandCAdvisorBuffer.WriteString(",")
		}
	}

	//make cAdvisor
	//cAdvisorList := commandCAdvisorBuffer.String()

	//Copy the Prometheus Configurations to Monitor Servers
	for _, monitorServer := range monitorServers {
		commandStr := fmt.Sprintf("sudo mkdir -p /linker/alertmanager && sudo mkdir -p /linker/prometheus/generated && sudo chown -R %s:%s /linker", monitorServer.SshUser, monitorServer.SshUser)
		_, _, err = command.ExecCommandOnMachine(monitorServer.Hostname, commandStr, storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("mkdir /linker/alertmanager and /linker/prometheus failed when copy monitor config file: %v", err)
			return
		}

		_, _, err = command.ScpToMachine(monitorServer.Hostname, "/linker/alertmanager/config.yml", "/linker/alertmanager/config.yml", storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("copy alermanager config file to target server %s fail, err is %v", monitorServer.Hostname, err)
			return
		}

		_, _, err = command.ScpToMachine(monitorServer.Hostname, "/linker/prometheus/prometheus.rules", "/linker/prometheus/prometheus.rules", storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("copy rule file to target server %s fail, err is %v", monitorServer.Hostname, err)
			return
		}

		_, _, err = command.ScpToMachine(monitorServer.Hostname, "/linker/prometheus/prometheus.yml", "/linker/prometheus/prometheus.yml", storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("copy prometheus config file to target server %s fail, err is %v", monitorServer.Hostname, err)
			return
		}

		// No need to config prometheus.yml, just point to master.mesos:10005, change the yml file only
		//		commandStr = fmt.Sprintf("sudo sed -i -e \\\"s/\\(targets:\\).*/\\1 [%s] /\\\" /linker/prometheus/prometheus.yml", cAdvisorList)
		//		_, _, err = command.ExecCommandOnMachine(monitorServer.Hostname, commandStr, storagePath)
		//		if err != nil {
		//			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
		//			dockerCLog.Errorf("change /linker/prometheus yml for cAdvisor failed when copy monitor config file: %v", err)
		//			return
		//		}

		//change alert config.yml url to dcos client address
		alertConfigConst := "LINKER_ALERT_URL"
		clientServerIP := getClientServer(swarmServers)
		alertURL := "http:\\/\\/" + clientServerIP + ":10004\\/v1\\/alerts"
		commandStr = fmt.Sprintf("sudo sed -i -e \\\"s/%s/%s/\\\" /linker/alertmanager/config.yml", alertConfigConst, alertURL)
		_, _, err = command.ExecCommandOnMachine(monitorServer.Hostname, commandStr, storagePath)
		if err != nil {
			errorCode = DEPLOY_ERROR_COPY_CONFIG_FILE
			dockerCLog.Errorf("config /linker/alertmanager/config.yml for alertmanager failed: %v", err)
			return
		}
	}

	//Copy mesos-agent tar file to all slave node for user cluster
	if !request.IsLinkerMgmt {
		err := copyMesosAgentToSlaves(slaveServers, storagePath)
		if err != nil {
			dockerCLog.Errorf("error occurred during copy mesos-agent tar file to agent node %v", err)
		}
	}

	if request.IsLinkerMgmt {
		err := CopyMgmtIpFile(servers, storagePath)
		if err != nil {
			dockerCLog.Errorf("copy mgmtips file to  node err is %v", err)
		}
	}

	if !request.IsLinkerMgmt {
		err := CopyClusterinfoFile(servers, request, storagePath, monitorServers)
		if err != nil {
			dockerCLog.Errorf("copy cluster file to  node err is %v", err)
		}
	}

	return
}

func CopyClusterinfoFile(servers []entity.Server, request entity.Request, storagePath string, monitorServers []entity.Server) (err error) {
	logrus.Infof("start to create cluster basic info")
	var clusterInfo entity.BasicInfo
	clusterInfo.ClusterName = request.ClusterName
	clusterInfo.ClusterId = request.ClusterId
	clusterInfo.UserName = request.UserName
	clusterInfo.UserId = request.UserId
	clusterInfo.TenantId = request.TenantId
	logrus.Infof("start to get mgmt ip info")
	mgmtIps, errG := Log.GetMgmtIps()
	if errG != nil {
		logrus.Errorf("get mgmtip err is %v", errG)
		return errG
	}
	clusterInfo.MgmtIp = mgmtIps

	if len(monitorServers) != 1 {
		logrus.Errorf("monitorserver ip is not only one")
	} else {
		clusterInfo.MonitorIp = monitorServers[0].IpAddress
	}
	logrus.Infof("start to create cluster info")
	errE := Log.CreateFileInfo(utils.BasicInfoPath, clusterInfo)
	if errE != nil {
		logrus.Errorf("create clusterinfo err is %v", errE)
		return errE
	}
	for _, se := range servers {
		commandStr := fmt.Sprintf("sudo mkdir -p /linker/docker && sudo chown -R %s:%s /linker", se.SshUser, se.SshUser)
		_, _, errC := command.ExecCommandOnMachine(se.Hostname, commandStr, storagePath)
		if errC != nil {
			logrus.Errorf("mkdir /linker/docker failed err is: %v", errC)
			return errC
		}
		_, _, errS := command.ScpToMachine(se.Hostname, utils.BasicInfoPath, utils.BasicInfoPath, storagePath)
		if errS != nil {
			logrus.Errorf("scp clusterinfo file err is %v", errS)
			return errS
		}
	}
	return
}

func CopyMgmtIpFile(servers []entity.Server, storagePath string) (err error) {
	logrus.Infof("start to create mgmt ip info")
	var mgmtIp entity.MgmtIps
	if len(servers) != 0 {
		for i, ser := range servers {
			str := make([]string, len(servers))
			mgmtIp.MgmtIps = str
			mgmtIp.MgmtIps[i] = ser.IpAddress
		}
		errC := Log.CreateFileInfo(utils.MgmtFilePath, mgmtIp)
		if errC != nil {
			logrus.Errorf("create mgmtip file err is %v", errC)
			return
		}
		for _, se := range servers {
			commandStr := fmt.Sprintf("sudo mkdir -p /linker/docker && sudo chown -R %s:%s /linker", se.SshUser, se.SshUser)
			_, _, errC := command.ExecCommandOnMachine(se.Hostname, commandStr, storagePath)
			if errC != nil {
				logrus.Errorf("mkdir /linker/docker failed err is: %v", errC)
				return errC
			}
			_, _, errS := command.ScpToMachine(se.Hostname, utils.MgmtFilePath, utils.MgmtFilePath, storagePath)
			if errS != nil {
				logrus.Errorf("scp mgmtip file err is %v", errS)
				return
			}
		}
	} else {
		err = errors.New("server is null")
	}

	return
}

func getClientServer(swarmServers []entity.Server) string {
	for _, server := range swarmServers {
		if server.IsClientServer {
			return server.PrivateIpAddress
		}
	}

	logrus.Warnf("does not found dcos client server!")
	return ""
}

func copyMesosAgentToSlaves(slaveServers []entity.Server, storagePath string) error {
	logrus.Infof("copy mesos-agent tar file to all slave node")

	copyChan := make(chan OperationResp)
	for i := 0; i < len(slaveServers); i++ {
		j := i
		go func() {
			hostname := slaveServers[j].Hostname
			_, _, err := command.ScpToMachine(hostname, "/linker/mesos/customized-slave.tar", "/tmp/customized-slave.tar", storagePath)
			if err != nil {
				logrus.Errorf("copy mesos-agent tar  file to target server %s fail, err is %v", hostname, err)
				copyChan <- OperationResp{ServerName: hostname, ErrMsg: err}
				return
			}
			copyChan <- OperationResp{ServerName: hostname}
		}()
	}

	for i := 0; i < len(slaveServers); i++ {
		copyresp := <-copyChan

		if copyresp.ErrMsg != nil {
			return copyresp.ErrMsg
		}
	}

	return nil
}

type OperationResp struct {
	ServerName string `json:"servername"`
	ErrMsg     error  `json:"errmsg"`
}

type CreateResp struct {
	Server entity.Server `json:"server"`
	ErrMsg error         `json:"errmsg"`
}

func dockerMachineGenericCreateCluster(request entity.Request) (servers, swarmServers, mgmtServers, slaveServers, monitorServers []entity.Server, errorCode string, err error) {
	genericLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
	userAttribute := request.NodeAttribute //form as : rack:abc;zone:bj;level:10
	isLinkerMgmt := request.IsLinkerMgmt
	category := request.CreateCategory //compact or ha

	//	addNodes := request.CreateNodes.Nodes
	masterNodes := request.MasterNodes
	sharedNodes := request.SharedNodes
	pureslaveNodes := request.PureSlaveNodes

	if request.MasterCount != len(masterNodes) && request.SharedCount != len(sharedNodes) && request.PureSlaveCount != len(pureslaveNodes) {
		genericLog.Errorf("request node number is inconsistence with add nodes number!")
		return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errors.New("request node number is inconsistence with add nodes number!")
	}

	masterLabel := entity.Label{Key: "master", Value: "true"}
	slaveLabel := entity.Label{Key: "slave", Value: "true"}

	adminrouterLabel := entity.Label{Key: "adminrouter", Value: "true"}
	dcosclientLabel := entity.Label{Key: "dcosclient", Value: "true"}
	monitorLabel := entity.Label{Key: "monitor", Value: "true"}

	fullAttValue := ""
	if len(userAttribute) > 0 {
		fullAttValue = MESOS_ATTRIBUTE_LB + ";" + userAttribute
	} else {
		fullAttValue = MESOS_ATTRIBUTE_LB
	}

	fullAttributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: fullAttValue}

	//get system component registry if exists
	imageRegistry := ""
	exist, registry := GetSystemRegistry(request.DockerRegistries)
	if exist {
		imageRegistry = registry.Registry
	}

	pubKeyPath := generatePubKey(storagePath, request.ClusterName, request.PubKey)
	genericLog.Infof("pubkey path is %s", pubKeyPath)

	if isLinkerMgmt {
		labels := []entity.Label{}
		labels = append(labels, masterLabel)
		labels = append(labels, slaveLabel)
		labels = append(labels, adminrouterLabel)
		labels = append(labels, fullAttributeLabel)

		if request.MasterCount == 1 {
			if len(masterNodes) != 1 {
				genericLog.Errorf("the node len is not equle to mastercount")
				return
			}
			var privateKeyPath string
			if masterNodes[0].PrivateKey == "" {
				genericLog.Errorf("node privatekey is empty")
				return
			} else {
				privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, masterNodes[0].PrivateKey)
			}
			if masterNodes[0].PrivateNicName == "" {
				masterNodes[0].PrivateNicName = "eth0"
			}

			server, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "reuse",
				masterNodes[0], labels, "", request.DockerRegistries, request.EngineOpts, privateKeyPath)
			server.IsMaster = true
			server.IsSlave = true
			server.IsSharedServer = true

			if errs != nil {
				genericLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
				servers = append(servers, server)
				return
			}
			swarmServers = append(swarmServers, server)
			mgmtServers = append(mgmtServers, server)
			servers = append(servers, server)
		} else if request.MasterCount == 3 {
			if len(masterNodes) != 3 {
				genericLog.Errorf("the node len is not equle to mastercount")
				return
			}
			var serverONE entity.Server

			var privateKeyPathONE string
			if masterNodes[0].PrivateKey == "" {
				genericLog.Errorf("node privatekey is empty")
				return
			} else {
				privateKeyPathONE = generatePrivateKey(request.UserName, request.ClusterName, masterNodes[0].PrivateKey)
			}
			if masterNodes[0].PrivateNicName == "" {
				masterNodes[0].PrivateNicName = "eth0"
			}

			serverONE, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "reuse",
				masterNodes[0], labels, "", request.DockerRegistries, request.EngineOpts, privateKeyPathONE)
			serverONE.IsMaster = true
			serverONE.IsSlave = true
			serverONE.IsSharedServer = true

			if errs != nil {
				genericLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
				servers = append(servers, serverONE)
				return
			}
			swarmServers = append(swarmServers, serverONE)
			mgmtServers = append(mgmtServers, serverONE)
			servers = append(servers, serverONE)

			masterNodeLeft := make([]entity.Node, 2)
			masterNodeLeft[0] = masterNodes[1]
			masterNodeLeft[1] = masterNodes[2]

			//create all Server
			mgmtChan := make(chan CreateResp)
			for i := 0; i < 2; i++ {
				j := i
				go func() {
					var privateKeyPath string
					if masterNodeLeft[j].PrivateKey == "" {
						genericLog.Errorf("node privatekey is empty")
						return
					} else {
						privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, masterNodeLeft[j].PrivateKey)
					}
					if masterNodeLeft[j].PrivateNicName == "" {
						masterNodeLeft[j].PrivateNicName = "eth0"
					}

					server, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
						request.ProviderInfo.Properties, pubKeyPath, "reuse",
						masterNodeLeft[j], labels, "", request.DockerRegistries, request.EngineOpts, privateKeyPath)
					server.IsMaster = true
					server.IsSlave = true
					server.IsSharedServer = true
					if errs != nil {
						genericLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
						servers = append(servers, server)
						mgmtChan <- CreateResp{Server: server, ErrMsg: errs}
						return
					}
					swarmServers = append(swarmServers, server)
					mgmtServers = append(mgmtServers, server)
					servers = append(servers, server)

					mgmtChan <- CreateResp{Server: server}
				}()
			}

			for i := 0; i < 2; i++ {
				mgmtresp := <-mgmtChan

				if mgmtresp.ErrMsg != nil {
					genericLog.Errorf("create a managed node error")
					return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, mgmtresp.ErrMsg
				}
			}

		}

		//create zk cluster and swarm cluster on mgmt nodes
		genericLog.Infof("begin to create zk cluster and swarm cluster on cluster %s", request.ClusterName)

		storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
		//		privateNicName := CreateZKNicNameList(mgmtServers, request)
		errZSM := createZKAndSwarmMaster(mgmtServers, storagePath, imageRegistry, request)
		if errZSM != nil {
			genericLog.Errorf("createZKAndSwarmMaster err is %s", errZSM)
			errorCode = DEPLOY_ERROR_CREATEZKSWARM
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errZSM
		}

	} else {
		//begin to create mgmt node
		if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
			mgmtlabel := []entity.Label{}
			mgmtlabel = append(mgmtlabel, masterLabel)
			mgmtlabel = append(mgmtlabel, adminrouterLabel)

			var nodeONEprivatekeyPath string
			masterNodeONE := masterNodes[0]
			if masterNodeONE.PrivateKey == "" {
				genericLog.Errorf("node privatekey is empty")
				return
			} else {
				nodeONEprivatekeyPath = generatePrivateKey(request.UserName, request.ClusterName, masterNodeONE.PrivateKey)
			}
			if masterNodeONE.PrivateNicName == "" {
				masterNodeONE.PrivateNicName = "eth0"
			}
			masterONELabel := []entity.Label{}
			masterONELabel = mgmtlabel
			masterONELabel = append(masterONELabel, dcosclientLabel)

			serverONE, _, errC := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "reuse",
				masterNodeONE, masterONELabel, "", request.DockerRegistries, request.EngineOpts, nodeONEprivatekeyPath)
			serverONE.IsClientServer = true
			serverONE.IsMaster = true
			if errC != nil {
				genericLog.Errorf("create swarm master server error %v", errC)
				servers = append(servers, serverONE)
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errC
			}

			swarmServers = append(swarmServers, serverONE)
			mgmtServers = append(mgmtServers, serverONE)
			servers = append(servers, serverONE)

			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", servers, request.XAuthToken)
			if errback != nil {
				genericLog.Errorf("callback host is err")
			}

			masterNodeLeft := make([]entity.Node, 2)
			masterNodeLeft[0] = masterNodes[1]
			masterNodeLeft[1] = masterNodes[2]

			//create another 2 management Node
			mgmtChan := make(chan CreateResp)
			for i := 0; i < 2; i++ {
				j := i
				go func() {
					var server entity.Server
					var errs error

					var privateKeyPath string
					if masterNodeLeft[j].PrivateKey == "" {
						genericLog.Errorf("node privatekey is empty")
						return
					} else {
						privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, masterNodeLeft[j].PrivateKey)
					}
					if masterNodeLeft[j].PrivateNicName == "" {
						masterNodeLeft[j].PrivateNicName = "eth0"
					}

					if j == 1 {
						//make the last mgmt server as the monitor server
						mgmtlabel_2 := []entity.Label{}
						mgmtlabel_2 = mgmtlabel
						mgmtlabel_2 = append(mgmtlabel_2, monitorLabel)
						server, _, errs = GetDockerMachineService().Create(request.UserName, request.ClusterName,
							request.ProviderInfo.Properties, pubKeyPath, "reuse",
							masterNodeLeft[j], mgmtlabel_2, "", request.DockerRegistries, request.EngineOpts, privateKeyPath)
					} else {
						server, _, errs = GetDockerMachineService().Create(request.UserName, request.ClusterName,
							request.ProviderInfo.Properties, pubKeyPath, "reuse",
							masterNodeLeft[j], mgmtlabel, "", request.DockerRegistries, request.EngineOpts, privateKeyPath)
					}

					server.IsMaster = true
					if errs != nil {
						genericLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
						servers = append(servers, server)
						mgmtChan <- CreateResp{Server: server, ErrMsg: errs}
						return
					}

					swarmServers = append(swarmServers, server)
					mgmtServers = append(mgmtServers, server)
					if j == 1 {
						//make the last mgmt server as the monitor server
						server.IsMonitorServer = true
						monitorServers = append(monitorServers, server)
					}
					servers = append(servers, server)

					mgmtChan <- CreateResp{Server: server}
				}()
			}

			for i := 0; i < 2; i++ {
				createresp := <-mgmtChan

				if createresp.ErrMsg != nil {
					genericLog.Errorf("create a managed node error")
					return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, createresp.ErrMsg
				}

				_, err = CallbackHost(request.ClusterName, request.UserName, true, "temporary", servers, request.XAuthToken)
				if err != nil {
					genericLog.Errorf("callback host is err")
				}
			}

		} else if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
			labels := []entity.Label{}
			labels = append(labels, masterLabel)
			labels = append(labels, dcosclientLabel)
			labels = append(labels, adminrouterLabel)
			labels = append(labels, monitorLabel)

			var privateKeyPath string
			if masterNodes[0].PrivateKey == "" {
				genericLog.Errorf("node privatekey is empty")
				return
			} else {
				privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, masterNodes[0].PrivateKey)
			}
			if masterNodes[0].PrivateNicName == "" {
				masterNodes[0].PrivateNicName = "eth0"
			}

			swarmServer, _, erro := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "reuse",
				masterNodes[0], labels, "", request.DockerRegistries, request.EngineOpts, privateKeyPath)
			swarmServer.IsMaster = true
			swarmServer.IsMonitorServer = true
			swarmServer.IsClientServer = true
			monitorServers = append(monitorServers, swarmServer)

			if erro != nil {
				genericLog.Errorf("create swarm master server error %v", erro)
				servers = append(servers, swarmServer)
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, erro
			}

			swarmServers = append(swarmServers, swarmServer)
			mgmtServers = append(mgmtServers, swarmServer)
			servers = append(servers, swarmServer)

			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", servers, request.XAuthToken)
			if errback != nil {
				genericLog.Errorf("callback host is err")
			}

		} else {
			genericLog.Errorf("not supported create mode %s", category)
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errors.New("not supported craete mode")
		}

		//create zk cluster and swarm cluster on mgmt nodes
		logrus.Infof("begin to create zk cluster and swarm cluster on cluster %s", request.ClusterName)
		storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName

		//		privateNicName := CreateZKNicNameList(mgmtServers, request)
		errZSM := createZKAndSwarmMaster(mgmtServers, storagePath, imageRegistry, request)
		if errZSM != nil {
			genericLog.Errorf("createZKAndSwarmMaster err is %s", errZSM)
			errorCode = DEPLOY_ERROR_CREATEZKSWARM
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errZSM
		}

		//begin to create slave node

		//prepare the zk for swarm master cluster and overlay network, for example: zk://xxx:2181,xx:2181
		ZK := BuildDiscoveryZKList(mgmtServers)

		//create shared server (first 2 slave nodes)  (2 shared servers for user cluster)
		sharedServerlabels := []entity.Label{}
		sharedServerlabels = append(sharedServerlabels, slaveLabel)
		sharedServerlabels = append(sharedServerlabels, fullAttributeLabel)

		sharedNum := request.SharedCount
		sharedChan := make(chan CreateResp)
		for i := 0; i < sharedNum; i++ {
			j := i
			go func() {
				var privateKeyPath string
				if sharedNodes[j].PrivateKey == "" {
					genericLog.Errorf("node privatekey is empty")
					return
				} else {
					privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, sharedNodes[j].PrivateKey)
				}
				if sharedNodes[j].PrivateNicName == "" {
					sharedNodes[j].PrivateNicName = "eth0"
				}

				server, _, errshare := GetDockerMachineService().Create(request.UserName, request.ClusterName,
					request.ProviderInfo.Properties, pubKeyPath, "reuse",
					sharedNodes[j], sharedServerlabels, ZK, request.DockerRegistries, request.EngineOpts, privateKeyPath)
				server.IsSlave = true
				server.IsSharedServer = true
				if errshare != nil {
					genericLog.Errorf("create one shared server error %v for cluster %s", errshare, request.ClusterName)
					servers = append(servers, server)
					sharedChan <- CreateResp{Server: server, ErrMsg: errshare}
					return
				}

				swarmServers = append(swarmServers, server)
				servers = append(servers, server)
				slaveServers = append(slaveServers, server)

				sharedChan <- CreateResp{Server: server}
			}()
		}

		for i := 0; i < sharedNum; i++ {
			sharedresp := <-sharedChan
			if sharedresp.ErrMsg != nil {
				genericLog.Errorf("create one shared server error")
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, sharedresp.ErrMsg
			}
			callserver := []entity.Server{}
			callserver = append(callserver, sharedresp.Server)
			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", callserver, request.XAuthToken)
			if errback != nil {
				genericLog.Errorf("callback host is err")
			}

		}

		// create pure salve node
		slavelabels := []entity.Label{}
		slavelabels = append(slavelabels, slaveLabel)
		if len(userAttribute) > 0 {
			attributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: userAttribute}
			slavelabels = append(slavelabels, attributeLabel)
		}

		//		offset = 5
		//		if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
		//			offset = 2
		//		}
		slaveChan := make(chan CreateResp)
		//		minmumNode := getMinmumNode(category)
		for i := 0; i < request.PureSlaveCount; i++ {
			j := i
			go func() {
				var privateKeyPath string
				if pureslaveNodes[j].PrivateKey == "" {
					genericLog.Errorf("node privatekey is empty")
					return
				} else {
					privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, pureslaveNodes[j].PrivateKey)
				}
				if pureslaveNodes[j].PrivateNicName == "" {
					pureslaveNodes[j].PrivateNicName = "eth0"
				}

				server, _, errslave := GetDockerMachineService().Create(request.UserName, request.ClusterName,
					request.ProviderInfo.Properties, pubKeyPath, "reuse",
					pureslaveNodes[j], slavelabels, ZK, request.DockerRegistries, request.EngineOpts, privateKeyPath)
				server.IsSlave = true
				if errslave != nil {
					genericLog.Errorf("create one salve server error %v for cluster %s", errslave, request.ClusterName)
					//					servers = append(servers, server)
					slaveChan <- CreateResp{Server: server, ErrMsg: errslave}
					return
				}
				swarmServers = append(swarmServers, server)
				servers = append(servers, server)
				slaveServers = append(slaveServers, server)

				slaveChan <- CreateResp{Server: server}
			}()
		}

		for i := 0; i < request.PureSlaveCount; i++ {
			salveresp := <-slaveChan
			callbackserver := []entity.Server{}
			callbackserver = append(callbackserver, salveresp.Server)
			succe := true
			if salveresp.ErrMsg != nil {
				genericLog.Errorln("create slave node error")
				succe = false
				//TODO don't return error after supporting partial success
				//				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, salveresp.ErrMsg
			}
			_, errback := CallbackHost(request.ClusterName, request.UserName, succe, "temporary", callbackserver, request.XAuthToken)
			if errback != nil {
				genericLog.Errorf("callback host is err")
			}
		}

		//Copy all the docker-machine's servers cluster files to management nodes
		userPath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName
		errscp := copyDockerMachineFiles(mgmtServers, storagePath, userPath)
		if errscp != nil {
			genericLog.Errorf("Create cluster %s to userpath %s failed when copy docker machine config, err is %v\n", request.ClusterName, userPath, errscp)

		}
	}

	return
}

func dockerMachineIaasCreateCluster(request entity.Request) (servers, swarmServers, mgmtServers, slaveServers, monitorServers []entity.Server, errorCode string, err error) {
	lassLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	sshUser := request.ProviderInfo.Provider.SshUser
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
	userAttribute := request.NodeAttribute //form as : rack:abc;zone:bj;level:10
	isLinkerMgmt := request.IsLinkerMgmt
	category := request.CreateCategory //compact or ha

	masterLabel := entity.Label{Key: "master", Value: "true"}
	slaveLabel := entity.Label{Key: "slave", Value: "true"}

	adminrouterLabel := entity.Label{Key: "adminrouter", Value: "true"}
	dcosclientLabel := entity.Label{Key: "dcosclient", Value: "true"}
	monitorLabel := entity.Label{Key: "monitor", Value: "true"}

	fullAttValue := ""
	if len(userAttribute) > 0 {
		fullAttValue = MESOS_ATTRIBUTE_LB + ";" + userAttribute
	} else {
		fullAttValue = MESOS_ATTRIBUTE_LB
	}

	fullAttributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: fullAttValue}

	//get system component registry if exists
	imageRegistry := ""
	exist, registry := GetSystemRegistry(request.DockerRegistries)
	if exist {
		imageRegistry = registry.Registry
	}

	pubKeyPath := generatePubKey(storagePath, request.ClusterName, request.PubKey)
	lassLog.Infof("pubkey path is %s", pubKeyPath)

	exportCredentialForGoogle(storagePath, request.ProviderInfo)
	if isLinkerMgmt {
		labels := []entity.Label{}
		labels = append(labels, masterLabel)
		labels = append(labels, slaveLabel)
		labels = append(labels, adminrouterLabel)
		labels = append(labels, fullAttributeLabel)

		//create all Server
		mgmtChan := make(chan CreateResp)

		if request.MasterCount == 1 {
			server, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "new",
				entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, labels, "", request.DockerRegistries, request.EngineOpts, "")
			server.IsMaster = true
			server.IsSlave = true
			server.IsSharedServer = true
			if errs != nil {
				servers = append(servers, server)
				return
			}

			swarmServers = append(swarmServers, server)
			mgmtServers = append(mgmtServers, server)
			servers = append(servers, server)
		} else if request.MasterCount == 3 {
			server, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "new",
				entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, labels, "", request.DockerRegistries, request.EngineOpts, "")
			server.IsMaster = true
			server.IsSlave = true
			server.IsSharedServer = true
			if errs != nil {
				servers = append(servers, server)
				return
			}

			swarmServers = append(swarmServers, server)
			mgmtServers = append(mgmtServers, server)
			servers = append(servers, server)

			for i := 0; i < 2; i++ {
				go func() {
					server, _, errs := GetDockerMachineService().Create(request.UserName, request.ClusterName,
						request.ProviderInfo.Properties, pubKeyPath, "new",
						entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, labels, "", request.DockerRegistries, request.EngineOpts, "")
					server.IsMaster = true
					server.IsSlave = true
					server.IsSharedServer = true
					if errs != nil {
						lassLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
						servers = append(servers, server)
						mgmtChan <- CreateResp{Server: server, ErrMsg: errs}
						return
					}
					swarmServers = append(swarmServers, server)
					mgmtServers = append(mgmtServers, server)
					servers = append(servers, server)

					mgmtChan <- CreateResp{Server: server}
				}()
			}

			for i := 0; i < 2; i++ {
				mgmtresp := <-mgmtChan

				if mgmtresp.ErrMsg != nil {
					lassLog.Errorf("create a managed node error")
					return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, mgmtresp.ErrMsg
				}
			}

		} else {
			logrus.Errorf("the instance of mgmtcluster is error")
			return
		}

		//create zk cluster and swarm cluster on mgmt nodes
		lassLog.Infof("begin to create zk cluster and swarm cluster on cluster %s", request.ClusterName)
		storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
		errZSM := createZKAndSwarmMaster(mgmtServers, storagePath, imageRegistry, request)
		if errZSM != nil {
			lassLog.Errorf("createZKAndSwarmMaster err is %s", errZSM)
			errorCode = DEPLOY_ERROR_CREATEZKSWARM
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errZSM
		}
	} else {

		if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
			mgmtlabel := []entity.Label{}
			mgmtlabel = append(mgmtlabel, masterLabel)
			mgmtlabel = append(mgmtlabel, adminrouterLabel)
			//create Server, install Swarm Slave and Label as other management Node

			var serverONE entity.Server
			mgmtlabelONE := []entity.Label{}
			mgmtlabelONE = mgmtlabel
			mgmtlabelONE = append(mgmtlabelONE, dcosclientLabel)

			serverONE, _, errC := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "new",
				entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, mgmtlabelONE, "", request.DockerRegistries, request.EngineOpts, "")
			serverONE.IsClientServer = true
			serverONE.IsMaster = true
			if errC != nil {
				lassLog.Errorf("create swarm master server error %v", errC)
				servers = append(servers, serverONE)
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errC
			}

			swarmServers = append(swarmServers, serverONE)
			mgmtServers = append(mgmtServers, serverONE)
			servers = append(servers, serverONE)

			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", servers, request.XAuthToken)
			if errback != nil {
				lassLog.Errorf("callback host is err")
			}

			mgmtChan := make(chan CreateResp)
			for i := 0; i < 2; i++ {
				j := i
				go func() {
					var server entity.Server
					var errs error
					if j == 1 {
						//make the last mgmt server as the monitor server
						mgmtlabel_2 := []entity.Label{}
						mgmtlabel_2 = mgmtlabel
						mgmtlabel_2 = append(mgmtlabel_2, monitorLabel)
						server, _, errs = GetDockerMachineService().Create(request.UserName, request.ClusterName,
							request.ProviderInfo.Properties, pubKeyPath, "new",
							entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, mgmtlabel_2, "", request.DockerRegistries, request.EngineOpts, "")
					} else {
						server, _, errs = GetDockerMachineService().Create(request.UserName, request.ClusterName,
							request.ProviderInfo.Properties, pubKeyPath, "new",
							entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, mgmtlabel, "", request.DockerRegistries, request.EngineOpts, "")
					}

					server.IsMaster = true
					if errs != nil {
						lassLog.Errorf("create one master server error %v for cluster %s", errs, request.ClusterName)
						servers = append(servers, server)
						mgmtChan <- CreateResp{Server: server, ErrMsg: errs}
						return

					}
					swarmServers = append(swarmServers, server)
					mgmtServers = append(mgmtServers, server)
					if j == 1 {
						//make the last mgmt server as the monitor server
						server.IsMonitorServer = true
						monitorServers = append(monitorServers, server)
					}
					servers = append(servers, server)
					mgmtChan <- CreateResp{Server: server}
				}()
			}
			for i := 0; i < 2; i++ {
				mgmtresp := <-mgmtChan
				if mgmtresp.ErrMsg != nil {
					lassLog.Errorf("create mgmt server error")
					return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, mgmtresp.ErrMsg
				}
				callbackserver := []entity.Server{}
				callbackserver = append(callbackserver, mgmtresp.Server)
				_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", callbackserver, request.XAuthToken)
				if errback != nil {
					lassLog.Errorf("callback host is err")
				}

			}
		} else if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
			labels := []entity.Label{}
			labels = append(labels, masterLabel)
			labels = append(labels, adminrouterLabel)
			labels = append(labels, dcosclientLabel)
			labels = append(labels, monitorLabel)

			swarmServer, _, erro := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, "new",
				entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, labels, "", request.DockerRegistries, request.EngineOpts, "")
			swarmServer.IsMaster = true
			swarmServer.IsMonitorServer = true
			swarmServer.IsClientServer = true
			monitorServers = append(monitorServers, swarmServer)

			if erro != nil {
				lassLog.Errorf("create swarm master server error %v", erro)
				servers = append(servers, swarmServer)
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, erro
			}

			swarmServers = append(swarmServers, swarmServer)
			mgmtServers = append(mgmtServers, swarmServer)
			servers = append(servers, swarmServer)

			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", servers, request.XAuthToken)
			if errback != nil {
				lassLog.Errorf("callback host is err")
			}
		} else {
			lassLog.Errorf("not supported create mode %s", category)
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errors.New("not supported create mode")
		}

		//create zk cluster and swarm cluster on mgmt nodes
		lassLog.Infof("begin to create zk cluster and swarm cluster on cluster %s", request.ClusterName)
		storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
		errZSM := createZKAndSwarmMaster(mgmtServers, storagePath, imageRegistry, request)
		if errZSM != nil {
			lassLog.Errorf("createZKAndSwarmMaster err is %s", errZSM)
			errorCode = DEPLOY_ERROR_CREATEZKSWARM
			return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, errZSM
		}

		//begin to create slave node

		//prepare the zk for swarm master cluster and overlay network, for example: zk://xxx:2181,xx:2181
		ZK := BuildDiscoveryZKList(mgmtServers)

		//create shared server (first 2 slave nodes)  (2 shared servers for user cluster)
		sharedServerlabels := []entity.Label{}
		sharedServerlabels = append(sharedServerlabels, slaveLabel)
		sharedServerlabels = append(sharedServerlabels, fullAttributeLabel)

		//		index := 1
		//		if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
		//			index = 1
		//		} else if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
		//			index = 2
		//		}
		sharedNum := request.SharedCount
		sharedChan := make(chan CreateResp)
		for i := 0; i < sharedNum; i++ {
			go func() {
				server, _, errshare := GetDockerMachineService().Create(request.UserName, request.ClusterName,
					request.ProviderInfo.Properties, pubKeyPath, "new",
					entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, sharedServerlabels, ZK, request.DockerRegistries, request.EngineOpts, "")
				server.IsSlave = true
				server.IsSharedServer = true
				if errshare != nil {
					lassLog.Errorf("create one shared server error %v for cluster %s", errshare, request.ClusterName)
					servers = append(servers, server)
					sharedChan <- CreateResp{Server: server, ErrMsg: errshare}
					return
				}

				swarmServers = append(swarmServers, server)
				servers = append(servers, server)
				slaveServers = append(slaveServers, server)

				sharedChan <- CreateResp{Server: server}
			}()
		}

		for i := 0; i < sharedNum; i++ {
			shareresp := <-sharedChan
			if shareresp.ErrMsg != nil {
				lassLog.Errorln("create shared server error")
				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, shareresp.ErrMsg
			}
			backserver := []entity.Server{}
			backserver = append(backserver, shareresp.Server)
			_, errback := CallbackHost(request.ClusterName, request.UserName, true, "temporary", backserver, request.XAuthToken)
			if errback != nil {
				lassLog.Errorf("callback host is err")
			}
		}

		// create pure salve node
		slavelabels := []entity.Label{}
		slavelabels = append(slavelabels, slaveLabel)
		if len(userAttribute) > 0 {
			attributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: userAttribute}
			slavelabels = append(slavelabels, attributeLabel)
		}

		//		minmumNode := getMinmumNode(category)
		pureslaveNum := request.PureSlaveCount
		slaveChan := make(chan CreateResp)
		for i := 0; i < pureslaveNum; i++ {
			go func() {
				server, _, errslave := GetDockerMachineService().Create(request.UserName, request.ClusterName,
					request.ProviderInfo.Properties, pubKeyPath, "new",
					entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}, slavelabels, ZK, request.DockerRegistries, request.EngineOpts, "")
				server.IsSlave = true
				if errslave != nil {
					lassLog.Errorf("create one salve server error %v for cluster %s", errslave, request.ClusterName)
					//					servers = append(servers, server)
					slaveChan <- CreateResp{Server: server, ErrMsg: errslave}
					return
				}

				swarmServers = append(swarmServers, server)
				servers = append(servers, server)
				slaveServers = append(slaveServers, server)

				slaveChan <- CreateResp{Server: server}
			}()
		}

		for i := 0; i < pureslaveNum; i++ {
			slaveresp := <-slaveChan
			callhostserver := []entity.Server{}
			callhostserver = append(callhostserver, slaveresp.Server)
			succ := true
			if slaveresp.ErrMsg != nil {
				lassLog.Errorln("create a slave server error")
				succ = false
				//TODO don't return error after supporting partial success
				//				return servers, swarmServers, mgmtServers, slaveServers, monitorServers, errorCode, slaveresp.ErrMsg
			}
			_, errback := CallbackHost(request.ClusterName, request.UserName, succ, "temporary", callhostserver, request.XAuthToken)
			if errback != nil {
				lassLog.Errorf("callback host is err")
			}
		}

		//Copy all the docker-machine's servers cluster files to management nodes
		userPath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName
		errscp := copyDockerMachineFiles(mgmtServers, storagePath, userPath)
		if errscp != nil {
			lassLog.Errorf("Create cluster %s to userpath %s failed when copy docker machine config, err is %v\n", request.ClusterName, userPath, errscp)

		}
	}

	return
}

func (p *DeployService) DeleteCluster(username, clustername, clusterId, logobjid string, x_auth_token string, mgmtIp string) (
	errorCode string, err error) {
	deleteLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	deleteLog.Infof("start to Delete Cluster Networks...")
	err = CleanNetwork(clusterId, x_auth_token, mgmtIp)
	if err != nil {
		deleteLog.Errorf("Delete Network %s failed, when delete networks err is %v\n", clustername, err)
	}

	//used for google cloud
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clustername
	googleCredentialPath := storagePath + GOOGLE_CREDENTIAL_FILE
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleCredentialPath)

	deleteLog.Infof("start to Delete Docker Cluster...")
	//get the cluster name and user info to call docker machine to delete all the vms
	err = GetDockerMachineService().DeleteAllMachines(username, clustername)
	if err != nil {
		//delete cluster is err callback to change cluster and host status failed
		_, errC := CallbackCluster(clustername, username, false, "delete", logobjid, "", x_auth_token)
		if errC != nil {
			deleteLog.Errorf("callback to cluster is err")
		}
		return
	}
	_, errC := CallbackCluster(clustername, username, true, "delete", logobjid, "", x_auth_token)
	if errC != nil {
		deleteLog.Errorf("callback to cluster is err")
	}

	return
}

func (p *DeployService) DeleteNode(request entity.DeleteRequest) (err error) {
	deletenodeLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	deletenodeLog.Infof("start to Delete Docker Machine Nodes...")
	//get the cluster name and user info to call docker machine to delete all the vms
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName

	//used for google cloud
	googleCredentialPath := storagePath + GOOGLE_CREDENTIAL_FILE
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleCredentialPath)

	var existErr error
	for _, server := range request.Servers {
		deletenodeLog.Infof("Update env file in docker compose service.")
		err = GetDockerComposeService().Remove(request.UserName, request.ClusterName, server.Hostname)
		if err != nil {
			existErr = err
			deletenodeLog.Errorf("Remove node %s failed in docker compose, when delete node: %s  err is %v\n", request.ClusterName, server.Hostname, err)
		}

		err = command.CleanUp(server.Hostname, storagePath)
		if err != nil {
			deletenodeLog.Warnf("clean delete node %s slave and swarm agent error %v", server.Hostname, err)
		}

		deletenodeLog.Infof("Removing docker machine node, username: %s, clustername %s, hostname %s\n", request.UserName, request.ClusterName, server.Hostname)
		err = GetDockerMachineService().DeleteMachine(request.UserName, request.ClusterName, server.Hostname)
		if err != nil {
			existErr = err
			deletenodeLog.Errorf("Delete node %s failed, when delete node: %s  err is %v\n", request.ClusterName, server.Hostname, err)
		}

	}

	success := true
	if existErr != nil {
		success = false
	}
	var hostname string
	for _, slave := range request.Servers {
		deletenodeLog.Infof("start to get delete slave hostname")
		hostname = hostname + slave.Hostname + ":" + slave.IpAddress + "\n"
	}
	//notify
	_, errC := CallbackCluster(request.ClusterName, request.UserName, success, "move", request.LogId, hostname, request.XAuthToken)
	if errC != nil {
		deletenodeLog.Errorf("callback to cluster is error %v", errC)
	}

	_, errH := CallbackHost(request.ClusterName, request.UserName, success, "move", request.Servers, request.XAuthToken)
	if errH != nil {
		deletenodeLog.Errorf("callback host is error %v", errH)
	}

	ip := request.DnsServers[0].IpAddress

	appid := "/linkerdns/lb/marathonlb"
	existInstance, _ := getAppInstance(ip, appid)
	logrus.Infof("the exist app instance is %v", existInstance)
	if existInstance == 0 {
		logrus.Errorf("lb existinstance is 0")
	}
	//	deleteshared := getsharedInstanceInServer(servers)
	logrus.Infof("shared instance is %v", request.NowShared)
	if request.NowShared != existInstance {
		scalelb(ip, request.NowShared)
	} else {
		logrus.Infof("don not need to scale lb instance")
	}

	return
}

func (p *DeployService) GetNodesCheck(username, clustername string) (ret []entity.NodesCheck, errorCode string, err error) {
	checkLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	if len(username) <= 0 || len(clustername) <= 0 {
		checkLog.Warnf("parameter is invalid for node check! username is %s, clustername is %s", username, clustername)
		err = errors.New("parameter is invalid for node check! username and clustername should not be null")
		return
	}

	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clustername + ""
	output, _, errc := command.LsNodesCheck(storagePath)
	if errc != nil {
		checkLog.Errorf("ls node check error %v", errc)
		return
	}

	ret = ParseNodecheck(output)
	return
}

func (p *DeployService) CreateNode(request entity.AddNodeRequest) (err error) {
	nodeLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	nodeLog.Infof("start to Scale Docker Machine...")
	slaves, errorCodeo, erro := p.execCreateNode(request)

	//set universe registry for user cluster
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName
	configUniverseRegistry(slaves, request.ClusterName, storagePath)

	success := true
	if erro != nil {
		success = false
	}

	var hostname string
	for _, slave := range slaves {
		nodeLog.Infof("start to get add slave hostname")
		hostname = hostname + slave.Hostname + ":" + slave.IpAddress + "\n"
	}

	//CALLBACK :
	if success {
		_, errC := CallbackCluster(request.ClusterName, request.UserName, success, "add", request.LogId, hostname, request.XAuthToken)
		if errC != nil {
			nodeLog.Errorf("callback to cluster is error %v", errC)
		}
	} else {
		_, errC := CallbackCluster(request.ClusterName, request.UserName, success, "add", request.LogId, errorCodeo, request.XAuthToken)
		if errC != nil {
			nodeLog.Errorf("callback to cluster is error %v", errC)
		}
	}

	_, errH := CallbackHost(request.ClusterName, request.UserName, success, "add", slaves, request.XAuthToken)
	if errH != nil {
		nodeLog.Errorf("callback host is error %v", errH)
	}

	return
}

func (p *DeployService) execCreateNode(request entity.AddNodeRequest) (slaves []entity.Server, errorCode string, err error) {
	execLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	userAttribute := request.NodeAttribute
	addmode := request.AddMode
	//	addnodes := request.AddNodes
	sharedNodes := request.SharedNodes
	pureshaveNodes := request.PureSlaveNodes

	slaveLabel := entity.Label{Key: "slave", Value: "true"}

	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName + "/" + request.ClusterName + ""

	pubKeyPath := generatePubKey(storagePath, request.ClusterName, request.PubKey)
	execLog.Infof("pubkey path is %s", pubKeyPath)

	//used for google cloud
	googleCredentialPath := storagePath + GOOGLE_CREDENTIAL_FILE
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", googleCredentialPath)

	sshUser := ""
	if addmode == "reuse" {
		if request.SharedCount != len(sharedNodes) && len(pureshaveNodes) != request.PureSlaveCount {
			execLog.Errorf("invalid node number for add node operations")
			return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, errors.New("inconsistent add node number!")
		}

	} else if addmode == "new" {
		sshUser = request.ProviderInfo.Provider.SshUser
		execLog.Debugf("mode new, ssh user is %s", sshUser)
	} else {
		execLog.Errorf("not supported mode %s", addmode)
		return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, errors.New("not supported add mode!")
	}

	//make zookeeper string
	zkurl := BuildDiscoveryZKList(request.DnsServers)

	slaves = []entity.Server{}
	slaveChan := make(chan CreateResp)
	sharedNum := request.SharedCount
	for i := 0; i < sharedNum; i++ {
		//	originNode := sharedNodes[i]
		j := i
		go func() {
			var privateKeyPath string
			fullAttValue := ""
			if len(userAttribute) > 0 {
				fullAttValue = MESOS_ATTRIBUTE_LB + ";" + userAttribute
			} else {
				fullAttValue = MESOS_ATTRIBUTE_LB
			}

			sharedslavelabels := []entity.Label{}
			sharedslavelabels = append(sharedslavelabels, slaveLabel)

			fullAttributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: fullAttValue}
			sharedslavelabels = append(sharedslavelabels, fullAttributeLabel)
			node := entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}
			if addmode == "reuse" {
				node = sharedNodes[j]
				if node.PrivateKey == "" {
					execLog.Errorf("node privatekey is empty")
					return
				} else {
					privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, node.PrivateKey)
				}
				if node.PrivateNicName == "" {
					node.PrivateNicName = "eth0"
				}
			}

			server, _, err := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, addmode,
				node, sharedslavelabels, zkurl, request.DockerRegistries, request.EngineOpts, privateKeyPath)
			server.IsSlave = true
			server.IsSharedServer = true
			if err != nil {
				execLog.Errorf("Create node failed in docker machine for cluster %s:  err is %v", request.ClusterName, err)
				slaveChan <- CreateResp{Server: server, ErrMsg: err}
				return
			}

			slaves = append(slaves, server)
			slaveChan <- CreateResp{Server: server}
		}()
	}
	var number int
	for i := 0; i < sharedNum; i++ {
		slaveresp := <-slaveChan
		callbackserver := []entity.Server{}
		callbackserver = append(callbackserver, slaveresp.Server)
		success := true
		if slaveresp.ErrMsg != nil {
			execLog.Errorln("create server error when add node to cluster")
			success = false
			number = number + 1
			//			return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, slaveresp.ErrMsg
		}
		_, errback := CallbackHost(request.ClusterName, request.UserName, success, "temporary", callbackserver, request.XAuthToken)
		if errback != nil {
			execLog.Errorf("callback host is err")
		}
	}

	if request.SharedCount != 0 {
		if number == request.SharedCount {
			execLog.Error("all add node is err")
			return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, errors.New("all add node is err")
		}
	}

	pureslaveNum := request.PureSlaveCount
	for i := 0; i < pureslaveNum; i++ {
		//		originNode := pureshaveNodes[i]
		j := i
		go func() {
			var privateKeyPath string
			pureslavelabels := []entity.Label{}
			pureslavelabels = append(pureslavelabels, slaveLabel)
			if len(userAttribute) > 0 {
				attributeLabel := entity.Label{Key: MESOS_ATTRIBUTE_LABEL_PREFIX, Value: userAttribute}
				pureslavelabels = append(pureslavelabels, attributeLabel)
			}
			node := entity.Node{SshUser: sshUser, PrivateNicName: "eth0"}

			if addmode == "reuse" {
				node = pureshaveNodes[j]
				if node.PrivateKey == "" {
					execLog.Errorf("node privatekey is empty")
					return
				} else {
					privateKeyPath = generatePrivateKey(request.UserName, request.ClusterName, node.PrivateKey)
				}
				if node.PrivateNicName == "" {
					node.PrivateNicName = "eth0"
				}
			}

			server, _, err := GetDockerMachineService().Create(request.UserName, request.ClusterName,
				request.ProviderInfo.Properties, pubKeyPath, addmode,
				node, pureslavelabels, zkurl, request.DockerRegistries, request.EngineOpts, privateKeyPath)
			server.IsSlave = true
			if err != nil {
				execLog.Errorf("Create node failed in docker machine for cluster %s:  err is %v", request.ClusterName, err)
				slaveChan <- CreateResp{Server: server, ErrMsg: err}
				return
			}

			slaves = append(slaves, server)
			slaveChan <- CreateResp{Server: server}

		}()
	}

	var num int
	for i := 0; i < pureslaveNum; i++ {
		slaveresp := <-slaveChan
		callbackserver := []entity.Server{}
		callbackserver = append(callbackserver, slaveresp.Server)
		success := true
		if slaveresp.ErrMsg != nil {
			execLog.Errorln("create server error when add node to cluster")
			success = false
			num = num + 1
			//			return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, slaveresp.ErrMsg
		}
		_, errback := CallbackHost(request.ClusterName, request.UserName, success, "temporary", callbackserver, request.XAuthToken)
		if errback != nil {
			execLog.Errorf("callback host is err")
		}
	}

	if pureslaveNum != 0 {
		if num == pureslaveNum {
			execLog.Error("all add node is err")
			return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, errors.New("all add node is err")
		}
	}

	//Change the "/etc/resolve.conf" of server
	err = changeNameserver(slaves, request.DnsServers, storagePath, false)
	if err != nil {
		execLog.Warnf("change name server error during add node %v", err)
		return slaves, DEPLOY_ERROR_DOCKERMACHINECREATE, err
	}

	masterSize := len(request.DnsServers)
	clusterSlaveSize := pureslaveNum + sharedNum + request.ExistedNumber - masterSize
	//Copy all the docker-machine's servers cluster files to management nodes
	userPath := DOCKERMACHINE_STORAGEPATH_PREFIX + request.UserName
	err = copyDockerMachineFiles(request.DnsServers, storagePath, userPath)
	if err != nil {
		execLog.Errorf("Create node %s failed in copy docker machine config: err is %v\n", request.ClusterName, err)
		return slaves, DEPLOY_ERROR_DOCKERCOMPOSE, err
	}

	isHA := masterSize > 1
	//prepare ".env" file for new Node
	allNodes := entity.GetAddrequestNodes(request)
	var pricName []string
	if len(allNodes) > 0 {
		pricName = getAllPrivateNicName(allNodes, slaves)
	}

	execLog.Infof("waiting 60s for adding node")
	time.Sleep(60 * time.Second)
	err = GetDockerComposeService().Add(request.UserName, request.ClusterName, pricName, slaves, clusterSlaveSize, request.SwarmMaster, isHA, request.DockerRegistries)
	if err != nil {
		execLog.Warnf("docker compose error during add node %v", err)
		return slaves, DEPLOY_ERROR_DOCKERCOMPOSE, err
	}

	//change prometheus configuration and restart prometheus
	// 1. execute the replacement of the cadvisor server
	var commandCAdvisorBuffer bytes.Buffer
	for i, server := range slaves {
		commandCAdvisorBuffer.WriteString("'")
		commandCAdvisorBuffer.WriteString(server.PrivateIpAddress)
		commandCAdvisorBuffer.WriteString(":10000")
		commandCAdvisorBuffer.WriteString("'")
		if i < (len(slaves) - 1) {
			commandCAdvisorBuffer.WriteString(",")
		}
	}

	//make cAdvisor monitor list
	//	cAdvisorList := commandCAdvisorBuffer.String()
	//	commandStr := fmt.Sprintf("sudo sed -i -e \\\"s/targets:\\s*\\[/targets: \\[%s,/\\\" /linker/prometheus/prometheus.yml", cAdvisorList)
	//	_, _, err = command.ExecCommandOnMachine(request.MonitorServerHostName, commandStr, storagePath)
	//	if err != nil {
	//		execLog.Errorf("change /linker/prometheus yml for cAdvisor failed when replace monitor config file: %v", err)
	//	}

	//	// 2. restart the prometheus containers
	//	err = GetDockerComposeService().Restart(request.UserName, request.ClusterName, request.SwarmMaster, "prometheus", "docker-compose-user.yml")
	//	if err != nil {
	//		execLog.Errorf("restart prometheusr failed when replace monitor config file for adding nodes with error %s", err)
	//	}

	errs := startMesosAgent(slaves, request.UserName, request.ClusterName)
	if errs != nil {
		execLog.Errorf("error occurred during copy mesos-agent tar file to agent node %v", errs)
	}

	ip := request.DnsServers[0].IpAddress
	appid := "/linkerdns/lb/marathonlb"
	existInstance, _ := getAppInstance(ip, appid)
	logrus.Infof("the exist app instance is %v", existInstance)
	if existInstance == 0 {
		logrus.Errorf("lb existinstance is 0")
	}
	addshared := getsharedInstanceInServer(slaves)
	logrus.Infof("add shared instance is %v", addshared)
	scaleto := existInstance + addshared
	scalelb(ip, scaleto)

	return
}

func startMesosAgent(slaves []entity.Server, username, clustername string) error {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clustername + ""
	err := copyMesosAgentToSlaves(slaves, storagePath)
	if err != nil {
		logrus.Errorf("error occurred during copy mesos-agent tar file to agent node %v", err)
		return err
	}

	bootUpMesosAgent(slaves, username, clustername)

	return nil
}

func scalelb(ip string, instance int) {
	appid := "/linkerdns/lb/marathonlb"
	scaleinstance := ScaleInstance{}
	scaleinstance.Instances = instance
	scaleAppInstance(scaleinstance, ip, appid)

}

type ScaleInstance struct {
	Instances int `json:"instances"`
}

func scaleAppInstance(scaleapp ScaleInstance, ip string, appid string) {
	url := strings.Join([]string{ip, "/marathon", "/v2/apps/", appid, "?force=true"}, "")
	logrus.Infof("scale appinstance url is %v", url)
	body, err := json.Marshal(scaleapp)
	if err != nil {
		logrus.Warnf("marshal scale app object error %v", err)
		return
	}

	var resp *http.Response
	resp, err = httpclient.Http_put(url, string(body),
		httpclient.Header{"Content-Type", "application/json"})

	if err != nil {
		logrus.Errorf("http put error %v", err)
		return
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http put failed %v", string(data))
		return
	}
}

func getAppInstance(ip string, appid string) (int, error) {
	logrus.Infof("start to get lb instance")
	url := strings.Join([]string{ip, "/marathon", "/v2/apps/", appid}, "")
	logrus.Infof("url is %v", url)
	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})

	if err != nil {
		logrus.Errorf("http get app by appid error %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	// if response status is greater than 400, means marathon returns error
	// else parse body, findout deploymentId, and return
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("marathon returned error code is %v", resp.StatusCode)
		logrus.Errorf("detail is %v", string(data))
		err = errors.New(string(data))
		return 0, err
	}

	// Parse data: marathon json data
	jsondata := map[string]interface{}{}
	result := json.NewDecoder(strings.NewReader(string(data)))
	result.Decode(&jsondata)
	jq := jsonq.NewQuery(jsondata)
	instance, errq := jq.Int("app", "instances")
	if errq != nil {
		logrus.Warnf("parse instances error %v", errq)
		return 0, errq
	}

	return instance, nil

}

// save and export google credential for google cloud
func exportCredentialForGoogle(storagePath string, providerinfo entity.ProviderInfo) error {
	if providerinfo.Provider.ProviderType == command.PROVIDER_TYPE_GOOGLE {
		value := providerinfo.Properties["google-application-credentials"]
		keyBytes, err := common.Base64Decode([]byte(value))
		if err != nil {
			logrus.Errorf("fail to decode google credential: %v", err)
			return err
		}
		credentials := string(keyBytes)

		//create folder
		if !common.CheckFileIsExist(storagePath + "/googlecredentials") {
			err := os.MkdirAll(storagePath+"/googlecredentials", 0755)
			if err != nil {
				logrus.Errorf("create google credentials folder error: %v", err)
				return err
			}
		}

		//create credential json file and write content
		var f *os.File
		var errn error
		filePath := storagePath + GOOGLE_CREDENTIAL_FILE //"/googlecredentials/service-account.json"
		if !common.CheckFileIsExist(filePath) {
			logrus.Infof("google credential file does exist, create it %s", filePath)
			f, errn = os.Create(filePath)
			if errn != nil {
				logrus.Errorf("create google credentials file error %v", errn)
				return errn
			}
		} else {
			f, errn = os.OpenFile(filePath, os.O_RDWR, 0755)
			if errn != nil {
				logrus.Errorf("open google credentials file error %v", errn)
				return errn
			}
		}

		_, errw := io.WriteString(f, credentials)
		if errw != nil {
			logrus.Errorf("write google credentials to file error %v", errw)
			return errw
		}

		//write google env to envfile and set google env
		envpath := storagePath + "/googlecredentials" + "/.googlecredentialenv"
		envcmd := "GOOGLE_APPLICATION_CREDENTIALS=" + filePath
		commandexport := fmt.Sprintf(`echo "export %s" > %s`, envcmd, envpath)

		cmd.ExecCommand(commandexport)

		//used for inside(above env can not be seen in code)
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filePath)

		return nil
	}

	return nil
}

func createZKAndSwarmMaster(servers []entity.Server, storagePath, imageRegistry string, request entity.Request) (err error) {
	zkList := BuildZookeeperList(servers)
	zkDiscoveryList := BuildDiscoveryZKList(servers)
	for i, server := range servers {
		hostname := server.Hostname
		// var NicNa string
		// if privateNicName == "" {
		getEth0 := make([]entity.Server, 1)
		getEth0[0] = server
		logrus.Infof("geteth0 is %v", getEth0)
		NicNa := CreateZKNicNameList(getEth0, request)
		// }
		_, _, err = command.BootUpZookeeper(hostname, storagePath, NicNa, zkList, imageRegistry, strconv.Itoa(i+1))
		if err != nil {
			logrus.Warnf("create zookeeper on mgmt node %s error %v", hostname, err)
			return
		}

		_, _, err = command.BootUpSwarmMaster(hostname, server.IpAddress, storagePath, zkDiscoveryList, imageRegistry)
		if err != nil {
			logrus.Warnf("create swarm master on mgmt node %s error %v", hostname, err)
			return
		}

		_, _, err = command.BootUpSwarmAgent(hostname, server.IpAddress, storagePath, zkDiscoveryList, imageRegistry)
		if err != nil {
			logrus.Warnf("create swarm agent on mgmt node %s error %v", hostname, err)
			return
		}

		command.ChangeConfigFile(hostname, zkDiscoveryList, storagePath, true, imageRegistry)
	}
	return
}

func getMinmumNode(category string) int {
	if strings.EqualFold(category, CLUSTER_CATEGORY_COMPACT) {
		return MINMUM_NODE_NUMBER_COMPACT
	} else if strings.EqualFold(category, CLUSTER_CATEGORY_HA) {
		return MINMUM_NODE_NUMBER_HA
	} else {
		logrus.Errorf("invalid cluster category value %s", category)
		return MINMUM_NODE_NUMBER_COMPACT
	}
}

func generatePrivateKey(user, cluster, keyValue string) string {
	geLog := logrus.WithFields(logrus.Fields{"clustername": cluster})
	if len(keyValue) <= 0 {
		geLog.Infof("no specified private key, will not generate it!")
		return ""
	}
	geLog.Infof("generate private key for user %s, cluster %s", user, cluster)

	decodeValue, errd := common.Base64Decode([]byte(keyValue))
	if errd != nil {
		geLog.Errorf("decode private key error %v", errd)
		return ""
	}

	value := string(decodeValue)
	geLog.Infof("the decoded private key value is %s", value)

	timeSec := strconv.FormatInt(time.Now().Unix(), 10)
	keyPath := "/tmp/" + user + "/" + cluster + "/" + timeSec
	keyabsolutPath := keyPath + "/id_rsa"

	os.MkdirAll(keyPath, os.ModePerm)
	file, err := os.Create(keyabsolutPath)
	if err != nil {
		geLog.Errorf("create private key error %v", err)
		return ""
	}

	if _, err := io.Copy(file, strings.NewReader(value)); err != nil {
		geLog.Errorf("write privatekey error %v", err)
		return ""
	}

	err = os.Chmod(keyabsolutPath, 0400)
	if err != nil {
		geLog.Errorf("set privatekey mode to 0400 error %v", err)
		return ""
	}

	return keyabsolutPath
}

func generatePubKey(storagePath, clusterName string, pubkey []entity.PubkeyInfo) (keyPath []string) {
	pubLog := logrus.WithFields(logrus.Fields{"clustername": clusterName})
	if len(pubkey) <= 0 {
		pubLog.Infof("no public key or invalid public key!")
		return
	}
	pubLog.Info("generate pubkey")

	for _, pub := range pubkey {
		subpath := storagePath + "/" + clusterName + "_" + pub.Name + ".pub"
		err := os.MkdirAll(storagePath, os.ModePerm)
		if err != nil {
			pubLog.Warnf("create storagePath error %v", err)
		}

		keyfile, errf := os.Create(subpath)
		if errf != nil {
			pubLog.Warnf("write key to file[%s] error [%v]", keyPath, errf)
			continue
		}
		defer keyfile.Close()

		keyfile.WriteString("\n")
		keyfile.WriteString(pub.Value)

		keyPath = append(keyPath, subpath)
	}

	return
}

func GetSystemRegistry(registries []entity.DockerRegistry) (bool, entity.DockerRegistry) {

	if registries == nil || len(registries) == 0 {
		return false, entity.DockerRegistry{}
	}
	for _, registry := range registries {

		if registry.IsSystemRegistry {
			return true, registry
		}
	}
	return false, entity.DockerRegistry{}

}

func CreateZKNicNameList(servers []entity.Server, request entity.Request) string {
	if len(servers) <= 0 {
		logrus.Warnln("server list is 0, can not return privatenicname list!")
		return ""
	}
	var ret bytes.Buffer
	for i, server := range servers {
		nicname := getNicAccIp(server.IpAddress, request)
		strTmp := strings.Replace(server.Hostname, ".", "_", -1)
		strTmp = strings.Replace(strTmp, "-", "_", -1)
		envname := "ENNAME_" + strTmp
		str := envname + "=" + nicname
		ret.WriteString(str)
		if i < len(servers)-1 {
			ret.WriteString(",")
		}

	}
	return ret.String()

}

func (p *DeployService) AddPubkeys(request entity.AddPubkeysRequest) (err error) {
	clusterName := request.ClusterName
	userName := request.UserName
	hosts := request.Hosts
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + userName + "/" + clusterName + ""
	//	var ids []string
	var arry []string
	for _, key := range request.Pubkey {
		//	var errMess error
		pubKeyPath := generateOtherPubKey(storagePath, request.ClusterName, key.Value, key.Name)
		arry = append(arry, pubKeyPath)
		logrus.Infof("arry is %v", arry)
	}
	for _, host := range hosts {
		GetDockerMachineService().ReplaceKey(host.HostName, host.SshUser, storagePath, arry, host.IP)
		//TODO callback
	}

	return
}

func generateOtherPubKey(storagePath, clusterName, pubkey string, pubkeyname string) (keyPath string) {
	pubLog := logrus.WithFields(logrus.Fields{"clustername": clusterName})
	if len(pubkey) <= 0 {
		pubLog.Infof("no public key or invalid public key!")
		return ""
	}
	pubLog.Info("generate pubkey")
	keyPath = storagePath + "/" + clusterName + "_" + pubkeyname + ".pub"
	err := os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		pubLog.Warnf("create storagePath error %v", err)
	}

	keyfile, errf := os.Create(keyPath)
	if errf != nil {
		pubLog.Warnf("write key to file[%s] error [%v]", keyPath, errf)
		return ""
	}
	defer keyfile.Close()

	keyfile.WriteString("\n")
	keyfile.WriteString(pubkey)

	return
}

func (p *DeployService) DeletePubkeys(request entity.DeletePubkeysRequest) (err error) {
	clusterName := request.ClusterName
	userName := request.UserName
	hosts := request.Hosts

	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + userName + "/" + clusterName + ""
	for _, key := range request.Pubkey {
		pubKeyPath := storagePath + "/" + clusterName + "_" + key.Name + ".pub"
		for _, host := range hosts {
			GetDockerMachineService().DeleteKey(host.HostName, host.SshUser, storagePath, pubKeyPath, host.IP)
			//TODO callback
		}
		commandDe := "rm -rf " + pubKeyPath
		_, _, err = cmd.ExecCommand(commandDe)
		if err != nil {
			logrus.Errorf("Call ssh-add failed , err is %v", err)
			continue
		}

	}

	return
}

func (p *DeployService) AddRegistry(request entity.AddRegistryRequest) (err error) {
	logrus.Infof("start to add registry to cluster")
	clusterName := request.ClusterName
	userName := request.UserName
	hosts := request.Hosts
	storagePath := GetDockerMachineService().ComposeStoragePath(userName, clusterName)

	for _, host := range hosts {
		for _, registry := range request.Registrys {
			if registry.Secure {
				err := GetDockerMachineService().WriteDockerRegistryCaFile(host.HostName, storagePath, registry)
				if err != nil {
					logrus.Errorf("write docker registry ca file to %s error is: %v", host.HostName, err)
					continue
				}
				logrus.Infof("write docker registry ca file to %s:%s success", host.HostName, storagePath)
			}

			if len(registry.Username) != 0 {
				err = GetDockerMachineService().InstallPackageExpect(host.HostName, storagePath)
				if err != nil {
					logrus.Errorf("execute install-expect.sh error: %v", err)
					continue
				}

				err = GetDockerMachineService().RegistryLogin(host.HostName, storagePath, registry)
				if err != nil {
					logrus.Errorf("login to docker registry error is %v", err)
					continue
				}
				logrus.Infoln("login to registry success")

				err := GetDockerMachineService().CompressDotDocker(host.HostName, storagePath)
				if err != nil {
					logrus.Errorf("compress docker auth config error: %v", err)
					continue
				}
				logrus.Infoln("compress docker auth config success")

			}

		}

	}

	return

}

func (p *DeployService) DeleteRegistry(request entity.DeleteRegistryRequest) (err error) {
	logrus.Infof("start to delete registry to cluster")
	clusterName := request.ClusterName
	userName := request.UserName
	hosts := request.Hosts
	storagePath := GetDockerMachineService().ComposeStoragePath(userName, clusterName)

	for _, host := range hosts {
		for _, registry := range request.Registrys {
			if registry.Secure {
				if strings.Contains(registry.Registry, "//") {
					logrus.Errorln("docker registry url contains protocol string //")
					return errors.New("invalid registry url")
				}
				path := "/etc/docker/certs.d/" + registry.Registry
				//file := path + "/ca.crt"
				deleteCommand := "sudo rm -rf " + path
				_, _, err = command.ExecCommandOnMachine(host.HostName, deleteCommand, storagePath)
				if err != nil {
					logrus.Errorf("Can't mkdir for host: [%s] for private docker registry, error is %v", host.HostName, err)
					continue
				}
			}

			if len(registry.Username) != 0 {
				errIn := GetDockerMachineService().InstallPackageExpect(host.HostName, storagePath)
				if errIn != nil {
					logrus.Errorf("execute install-expect.sh error: %v", errIn)
					continue
				}

				copyErr := GetDockerMachineService().CopyDockerLogoutScript(host.HostName, storagePath)
				if copyErr != nil {
					logrus.Errorf("copy docker logout script to %s:%s error is %v", host.HostName, storagePath, copyErr)
					continue
				}

				errRe := GetDockerMachineService().RegistryLogout(host.HostName, storagePath, registry)
				if errRe != nil {
					logrus.Errorf("logout registry err is %v", errRe)
				}
				errCo := GetDockerMachineService().CompressDotDocker(host.HostName, storagePath)
				if errCo != nil {
					logrus.Errorf("compress docker auth config error: %v", errCo)
					continue
				}
				logrus.Infoln("compress docker auth config success")

			}

		}
	}
	return

}

func (p *DeployService) ComponentsCheck(request entity.Components) (componentInfo entity.ComponentsInfo, errCode string, err error) {
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	creLog.Infof("start to check cluster component")
	storagePath := GetDockerMachineService().ComposeStoragePath(request.UserName, request.ClusterName)
	filePath, errF := command.CreateComponentFile(request, storagePath)
	if errF != nil {
		creLog.Errorf("create component container.txt file err is %v", errF)
		errCode = COMPONENT_ERROR_CREATE_FILE
		return componentInfo, errCode, errF
	}

	masterComponents := request.MasterComponents
	masterContainerInfo, errCodeM, errM := GetComponentInfoByType(request, storagePath, filePath, masterComponents, false)
	if errM != nil {
		creLog.Errorf("get master componentinfo err is %v", errM)
		errCode = errCodeM
		err = errM
		return
	}
	slaveComponents := request.SlaveComponents
	slaveContainerInfo, errCodeS, errS := GetComponentInfoByType(request, storagePath, filePath, slaveComponents, false)
	if errS != nil {
		creLog.Errorf("get master componentinfo err is %v", errM)
		errCode = errCodeS
		err = errS
		return
	}
	OneComponents := request.OnlyOneComponents
	OneContainerInfo, errCodeO, errO := GetComponentInfoByType(request, storagePath, filePath, OneComponents, true)
	if errO != nil {
		creLog.Errorf("get master componentinfo err is %v", errM)
		errCode = errCodeO
		err = errO
		return
	}
	allComponentStatus := []entity.ComponentsStatus{}
	allComponents := request.AllComponents
	allContainerInfo, errCodeA, errA := GetComponentInfoByType(request, storagePath, filePath, allComponents, false)
	if errA != nil {
		creLog.Errorf("get master componentinfo err is %v", errM)
		errCode = errCodeA
		err = errA
		return
	}
	if len(masterContainerInfo) != 0 {
		for _, masterComponent := range masterContainerInfo {
			allComponentStatus = append(allComponentStatus, masterComponent)
		}
	}

	if len(slaveContainerInfo) != 0 {
		for _, slaveComponent := range slaveContainerInfo {
			allComponentStatus = append(allComponentStatus, slaveComponent)
		}
	}
	if len(OneContainerInfo) != 0 {
		for _, oneComponent := range OneContainerInfo {
			allComponentStatus = append(allComponentStatus, oneComponent)
		}
	}
	if len(allContainerInfo) != 0 {
		for _, allComponent := range allContainerInfo {
			allComponentStatus = append(allComponentStatus, allComponent)
		}
	}
	componentInfo.UserName = request.UserName
	componentInfo.ClusterName = request.ClusterName
	componentInfo.ClusterId = request.ClusterId
	componentInfo.ComponentsStatus = allComponentStatus
	return
}

func GetComponentInfoByType(request entity.Components, storagePath, filePath string, image entity.Image, onlyOne bool) (containerInfo []entity.ComponentsStatus, errCode string, err error) {
	creLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	if len(image.ImageName) == 0 {
		creLog.Errorf("NodeInfo cannot be 0")
		err = errors.New("NodeInfo cannot be 0")
		return
	}
	if !onlyOne {
		for _, imagename := range image.ImageName {
			if imagename != "mesosslave" {
				creLog.Infof("start to get componentsid, imagename is %v", imagename)
				componentid, errC := command.GetCompanentId(imagename, filePath, request.ClusterName)
				if errC != nil {
					creLog.Errorf("get component id err is %v", errC)
					errCode = COMPONENT_ERROR_GET_COMPONENTID
					return containerInfo, errCode, errC
				}
				creLog.Infof("componentid is %v", componentid)
				creLog.Infof("componentid lenth is %v", len(componentid))
				ComStatus := make([]entity.ComponentsStatus, len(image.NodeInfo))
				if len(componentid) != 0 {
					creLog.Infof("componentid lenth is not 0")
					if len(componentid) <= len(image.NodeInfo) {
						var compareIp []string
						var notexistIp []string
						for i, id := range componentid {
							creLog.Infof("start to get container status and ip, id is %v", id)
							ip, status, errI := command.GetcomponentsInfo(storagePath, request.SwarmName, id)
							if errI != nil {
								creLog.Errorf("get component status err is %v", errI)
								return
							}
							var Stat string
							creLog.Infof("ip is %v", ip)
							creLog.Infof("status is %v", status)

							flag := false
							for _, nodeip := range image.NodeInfo {
								if nodeip.IP == ip {
									flag = true
									compareIp = append(compareIp, nodeip.IP)
								}
							}
							if flag {
								if status == "running" {
									Stat = "Healthy"
								} else {
									Stat = "UnHealthy"
								}
								creLog.Infof("Stat is %v", Stat)
								ComStatus[i].ComponentName = imagename
								ComStatus[i].Ip = ip
								ComStatus[i].Status = Stat
							}

						}
						for _, node := range image.NodeInfo {
							flagtwo := false
							for _, compare := range compareIp {
								if node.IP == compare {
									flagtwo = true
								}
							}
							if !flagtwo {
								notexistIp = append(notexistIp, node.IP)
							}
						}
						creLog.Infof("notexistIp is %v", notexistIp)
						if len(notexistIp) != 0 {
							for j, notip := range notexistIp {
								lenth := len(componentid)
								ComStatus[lenth+j].ComponentName = imagename
								ComStatus[lenth+j].Ip = notip
								ComStatus[lenth+j].Status = "UnHealthy"
							}
						}

					}
				} else if len(componentid) == 0 {
					creLog.Infof("componentis is 0, so this component is unhealthy")
					for l, nod := range image.NodeInfo {
						ComStatus[l].ComponentName = imagename
						ComStatus[l].Ip = nod.IP
						ComStatus[l].Status = "UnHealthy"
					}

				}
				creLog.Infof("ComStatus is %v", ComStatus)

				for _, comstatu := range ComStatus {
					containerInfo = append(containerInfo, comstatu)
				}
				creLog.Infof("containerInfo is %v", containerInfo)
			} else {
				creLog.Infof("image is mesosslave")
				container := make([]entity.ComponentsStatus, len(image.NodeInfo))
				for k, nodeIn := range image.NodeInfo {
					cmd := "systemctl is-active dcos-mesos-slave"
					output, errput, errE := command.ExecCommandOnMachine(nodeIn.HostName, cmd, storagePath)
					if errE != nil {
						creLog.Errorf("get mesos slave status err is %v", errE)
						creLog.Errorf("get mesos slave status errput is %v", errput)
						creLog.Errorf("get mesos slave status err, but output has some description, if the output is not active, we think the status is unhealthy")
					}
					var status string
					if output == "active" {
						status = "Healthy"
					} else {
						status = "UnHealthy"
					}
					container[k].ComponentName = "mesosslave"
					container[k].Ip = nodeIn.IP
					container[k].Status = status

				}
				for _, contain := range container {
					containerInfo = append(containerInfo, contain)
				}
				creLog.Infof("ComStatus is %v", container)
				creLog.Infof("containerInfo is %v", containerInfo)
			}
		}
	} else {
		creLog.Infof("these image container must be only one in master node")
		for _, imagena := range image.ImageName {
			componentid, errC := command.GetCompanentId(imagena, filePath, request.ClusterName)
			if errC != nil {
				creLog.Errorf("get component id err is %v", errC)
				errCode = COMPONENT_ERROR_GET_COMPONENTID
				return containerInfo, errCode, errC
			}
			if len(componentid) == 1 {
				ip, status, errI := command.GetcomponentsInfo(storagePath, request.SwarmName, componentid[0])
				if errI != nil {
					creLog.Errorf("get component status err is %v", errI)
					return
				}
				var stats string
				if status == "running" {
					stats = "Healthy"
				} else {
					stats = "UnHealthy"
				}
				containeronly := make([]entity.ComponentsStatus, 1)
				containeronly[0].ComponentName = imagena
				containeronly[0].Ip = ip
				containeronly[0].Status = stats
				containerInfo = append(containerInfo, containeronly[0])
			} else if len(componentid) == 0 {
				var needip string
				if imagena == "dcosclient" {
					needip = request.ClientIp
				} else if imagena == "alertmanager" || imagena == "prometheus" {
					needip = request.MonitorIp
				}
				containeronlyT := make([]entity.ComponentsStatus, 1)
				containeronlyT[0].ComponentName = imagena
				containeronlyT[0].Status = "UnHealthy"
				containeronlyT[0].Ip = needip
				containerInfo = append(containerInfo, containeronlyT[0])
			}
		}
	}

	return
}
