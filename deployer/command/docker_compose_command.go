package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	command "linkernetworks.com/dcos-backend/common/common"
	"linkernetworks.com/dcos-backend/deployer/common"
)

const (
	DOCKERSWARM_STORAGEPATH_PREFIX string = "/linker/swarm/"
	DOCKER_COMPOSE_USER_YML_NAME   string = "docker-compose-user.yml"
)

func clusterCompose(userName, clusterName, swarmName, storagePath string, slaveScale int, isLinkerMgmt bool, isHA bool) (output, errput string, err error) {
	var tmpOutPut string
	var tmpErrPut string
	var tmpErr error
	if isLinkerMgmt {
		tmpOutPut, tmpErrPut, tmpErr = changeSwarmForMgmtCluster(userName, clusterName, swarmName, storagePath, slaveScale)
	} else {
		masterScale := 1
		if isHA {
			masterScale = 3
		}
		tmpOutPut, tmpErrPut, tmpErr = changeSwarmForUserCluster(userName, clusterName, swarmName, storagePath, slaveScale, masterScale)
	}
	output = output + tmpOutPut + "\n"
	errput = errput + tmpErrPut + "\n"
	if tmpErr != nil {
		err = tmpErr
		return
	}
	return
}

//generate command for docker-machine to create cluster machine on Iaas
func changeSwarmForMgmtCluster(userName, clusterName, swarmName, storagePath string, slaveScale int) (output, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	str := strconv.Itoa(slaveScale)
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash --swarm ")
	commandTextBuffer.WriteString(swarmName + "` && ")
	commandTextBuffer.WriteString(" export COMPOSE_HTTP_TIMEOUT=300 && ")
	commandTextBuffer.WriteString("docker-compose -f ")
	commandTextBuffer.WriteString(DOCKERSWARM_STORAGEPATH_PREFIX + userName + "/" + clusterName + "/docker-compose.yml ")
	commandTextBuffer.WriteString("scale ")
	commandTextBuffer.WriteString("mesosmaster")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("dnsserver")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("genresolvconf")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("adminrouter")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("marathon")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("mesosslave")
	commandTextBuffer.WriteString("=" + str + " ")
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof(commandTextBuffer.String())
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof("Change Swarm to: %s", userName)
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func changeSwarmForUserCluster(userName, clusterName, swarmName, storagePath string, slaveScale int, masterScale int) (output, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	str := strconv.Itoa(slaveScale)
	masterStr := strconv.Itoa(masterScale)
	resolvStr := strconv.Itoa(slaveScale + masterScale)
	// replace yml field
	// /linker/swarm/sysadmin/cluster1/docker-compose-user.yml
	ymlPath := fmt.Sprintf("%s%s/%s/%s", DOCKERSWARM_STORAGEPATH_PREFIX, userName, clusterName, DOCKER_COMPOSE_USER_YML_NAME)
	err = replaceFileContent(ymlPath, "${CLUSTER_OWNER}", userName)
	if err != nil {
		logrus.Errorf("replace string in yml error: %v", err)
		return
	}
	err = replaceFileContent(ymlPath, "${CLUSTER_NAME}", clusterName)
	if err != nil {
		logrus.Errorf("replace string in yml error: %v", err)
		return
	}
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash --swarm ")
	commandTextBuffer.WriteString(swarmName + "` && ")
	commandTextBuffer.WriteString(" export COMPOSE_HTTP_TIMEOUT=300 && ")
	commandTextBuffer.WriteString("docker-compose -f ")
	commandTextBuffer.WriteString(DOCKERSWARM_STORAGEPATH_PREFIX + userName + "/" + clusterName + "/docker-compose-user.yml ")
	commandTextBuffer.WriteString(" scale ")
	// commandTextBuffer.WriteString("exhibitor")
	// commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("mesosmaster")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("dnsserver")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("genresolvconf")
	commandTextBuffer.WriteString("=" + resolvStr + " ")
	commandTextBuffer.WriteString("adminrouter")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("marathon")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("cosmos")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	// commandTextBuffer.WriteString("mesosslave")
	// commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("cadvisormonitor")
	commandTextBuffer.WriteString("=" + str + " ")
	commandTextBuffer.WriteString("prometheus")
	commandTextBuffer.WriteString("=1 ")
	commandTextBuffer.WriteString("alertmanager")
	commandTextBuffer.WriteString("=1 ")
	commandTextBuffer.WriteString("mongodb")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("universeregistry")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("universenginx")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	commandTextBuffer.WriteString("dcosclient")
	commandTextBuffer.WriteString("=1 ")
	commandTextBuffer.WriteString("webconsole")
	commandTextBuffer.WriteString("=1 ")
	commandTextBuffer.WriteString("metricscollector")
	commandTextBuffer.WriteString("=" + masterStr + " ")
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof(commandTextBuffer.String())
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof("Change Swarm to: %s", userName)
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

// masterList: ["10.10.10.1", "10.10.10.2", "10.10.10.3"]
// nodeList node=publicIp: ["hostname1=10.10.10.1", "hostname2=10.10.10.2", "hostname2=10.10.10.3"]
// scale mesosslave
func InstallCluster(userName string, clusterName string, privateNicName []string, swarmName string, storagePath string, masterList []string, nodeList []string, scale int, isLinkerMgmt bool, isHA bool, imageRegistry string) error {

	err := fillEnvFile(userName, clusterName, privateNicName, masterList, nodeList, isLinkerMgmt, imageRegistry)
	if err != nil {
		return err
	}

	_, tmpErr, err := clusterCompose(userName, clusterName, swarmName, storagePath, scale, isLinkerMgmt, isHA)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof(tmpErr)
		return err
	}
	return nil
}

// addNode node=publicIp: "hostname1=10.10.10.1"
// scale mesosslave
func AddSlaveToCluster(userName string, clusterName string, privateNicName []string, swarmName string, storagePath string, addNodes []string, scale int, isHA bool, imageRegistry string) error {
	addLog := logrus.WithFields(logrus.Fields{"clustername": clusterName})
	envFilePath, err := createOrGetEnvFile(userName, clusterName, false, imageRegistry)
	if err != nil {
		addLog.Error(err)
		return err
	}

	envFile, erro := os.OpenFile(envFilePath, os.O_RDWR|os.O_APPEND, 0644)
	if erro != nil {
		addLog.Errorf("open env file error %v", erro)
		return erro
	}
	defer envFile.Close()

	for _, tmpStr := range addNodes {
		envFile.WriteString(tmpStr + "\n")
	}
	if len(privateNicName) > 0 {
		for _, priname := range privateNicName {
			envFile.WriteString(priname + "\n")
		}
	}

	_, tmpErr, err := clusterCompose(userName, clusterName, swarmName, storagePath, scale, false, isHA)
	if err != nil {
		addLog.Infof(tmpErr)
		return err
	}

	return nil
}

// deleteNode node=publicIp: "hostname2"
func RemoveSlaveFromCluster(userName, clusterName, deleteNode string) error {
	envFilePath, err := createOrGetEnvFile(userName, clusterName, false, "")
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Error(err)
		return err
	}

	content, err := ioutil.ReadFile(envFilePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	envFile, erro := os.OpenFile(envFilePath, os.O_RDWR, 0644)
	if erro != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Errorf("open env file error %v", erro)
		return erro
	}

	defer envFile.Close()

	envFile.Truncate(0)
	for _, line := range lines {
		if strings.HasPrefix(line, deleteNode+"=") == false {
			envFile.WriteString(line)
			envFile.WriteString("\n")
		}
	}

	return nil
}

func RestartService(userName, clusterName, swarmMaster, serviceName, ymlFileName, storagePath string) error {
	_, tmpErr, err := restartService(userName, clusterName, swarmMaster, serviceName, ymlFileName, storagePath)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof(tmpErr)
		return err
	}

	return nil
}

func restartService(userName, clusterName, swarmName, serviceName, ymlFileName, storagePath string) (output, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash --swarm ")
	commandTextBuffer.WriteString(swarmName + "` && ")
	commandTextBuffer.WriteString("docker-compose -f ")
	commandTextBuffer.WriteString(DOCKERSWARM_STORAGEPATH_PREFIX + userName + "/" + clusterName + "/" + ymlFileName)
	commandTextBuffer.WriteString(" restart ")
	commandTextBuffer.WriteString(serviceName)
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof(commandTextBuffer.String())
	logrus.WithFields(logrus.Fields{"clustername": clusterName}).Infof("restart Service : %s", serviceName)
	tmpOutPut, tmpErrPut, tmpErr := command.ExecCommand(commandTextBuffer.String())
	output = output + tmpOutPut + "\n"
	errput = errput + tmpErrPut + "\n"
	if tmpErr != nil {
		err = tmpErr
		return
	}
	return
}

func fillEnvFile(userName string, clusterName string, privateNicName []string, masterList []string, nodeList []string, isLinkerMgmt bool, imageRegistry string) error {
	envFilePath, err := createOrGetEnvFile(userName, clusterName, isLinkerMgmt, imageRegistry)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Error(err)
		return err
	}

	envFile, erro := os.OpenFile(envFilePath, os.O_RDWR, 0644)
	if erro != nil {
		logrus.WithFields(logrus.Fields{"clustername": clusterName}).Errorf("open env file error %v", erro)
		return erro
	}
	defer envFile.Close()
	ZOOKEEPERLIST := buildZKList(masterList)
	MESOS_ZK := buildMesosZK(masterList)
	MARATHON_MASTER := MESOS_ZK
	MESOS_MASTER := MESOS_ZK
	MARATHON_ZK := buildMarathonZK(masterList)
	MONGODBLIST := buildMongoList(masterList)
	MESOS_QUORUM := generateQuorum(len(masterList))
	SWARM_ENDPOINTS := buildSwarmEndpoints(nodeList, "3376")
	envFile.WriteString("ZOOKEEPERLIST=" + ZOOKEEPERLIST + "\n")
	envFile.WriteString("MESOS_ZK=" + MESOS_ZK + "\n")
	envFile.WriteString("MESOS_MASTER=" + MESOS_MASTER + "\n")
	envFile.WriteString("MESOS_QUORUM=" + MESOS_QUORUM + "\n")
	envFile.WriteString("MARATHON_MASTER=" + MARATHON_MASTER + "\n")
	envFile.WriteString("MARATHON_ZK=" + MARATHON_ZK + "\n")
	envFile.WriteString("MESOS_HOSTNAME_LOOKUP=false\n")
	envFile.WriteString("MONGODB_NODES=" + MONGODBLIST + "\n")
	envFile.WriteString("SWARM_ENDPOINTS=" + SWARM_ENDPOINTS + "\n")
	for _, tmpStr := range nodeList {
		envFile.WriteString(tmpStr + "\n")
	}
	if len(privateNicName) > 0 {
		for _, nicName := range privateNicName {
			envFile.WriteString(nicName + "\n")
		}
	}
	return nil
}

func createOrGetEnvFile(userName string, clusterName string, isLinkerMgmt bool, imageRegistry string) (envFilePath string, err error) {
	absolutePath := DOCKERSWARM_STORAGEPATH_PREFIX + userName + "/" + clusterName
	absoluteFilePath := absolutePath + "/.env"
	_, errInfile := os.Stat(absoluteFilePath)
	isExisted := errInfile == nil || os.IsExist(errInfile)
	if isExisted {
		return absoluteFilePath, nil
	} else {
		composeFile := "docker-compose.yml"
		if !isLinkerMgmt {
			composeFile = "docker-compose-user.yml"
		}

		os.MkdirAll(absolutePath, os.ModePerm)
		//_, err = copyFile(absolutePath+"/"+composeFile, "/linker/config/"+composeFile)
		err = parseRegistry(absolutePath+"/"+composeFile, "/linker/config/"+composeFile, imageRegistry)
		if err != nil {
			return
		}

		_, err = os.Create(absoluteFilePath)
		return absoluteFilePath, err
	}
}

//Copyfile
func copyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func buildZKList(masterList []string) string {
	value := buildValueList(masterList, ":2888:3888")
	logrus.Debugf("zookeeper list value is %s", value)

	return value
}

func buildMesosZK(masterList []string) string {
	value := buildValueList(masterList, ":2181")
	mesoszk := "zk://" + value + "/mesos"
	logrus.Debugf("mesoszk value is %s", mesoszk)

	return mesoszk
}

func buildMarathonZK(masterList []string) string {
	value := buildValueList(masterList, ":2181")
	marathonzk := "zk://" + value + "/marathon"
	logrus.Debugf("marathonzk value is %s", marathonzk)

	return marathonzk
}

func buildMongoList(masterList []string) string {
	value := buildValueList(masterList, "")
	logrus.Debugf("mongo list is %s", value)

	return value
}

func generateQuorum(num int) string {
	value := float64(num) / 2.0
	quorum := math.Ceil(value)

	return strconv.Itoa(int(quorum))
}

func buildValueList(masterList []string, port string) string {
	if len(masterList) <= 0 {
		logrus.Errorf("master list length is zero!")
		return ""
	}

	length := len(masterList)
	var ret bytes.Buffer
	for i := 0; i < length; i++ {
		ret.WriteString(masterList[i])
		if len(port) > 0 {
			ret.WriteString(port)
		}

		if i < length-1 {
			ret.WriteString(",")
		}
	}

	logrus.Debugf("the value list is %s", ret.String())
	return ret.String()
}

// get swarm master from nodes
func buildSwarmEndpoints(nodeList []string, swarmPort string) (swarmEndpoints string) {
	var endpoints []string
	for _, node := range nodeList {
		// node like:
		// linker_hostname_92fc5668_db08_479b_8d0d_a3207a502fca_all_reg_sysadmin=54.238.241.230
		arr := strings.Split(node, "=")
		if len(arr) != 2 {
			logrus.Errorln("parse public ip from env node error")
			break
		}
		ip := arr[1]
		endpoint := ip + ":" + swarmPort
		if isSwarmOK(endpoint) {
			endpoints = append(endpoints, endpoint)
		}
	}
	return strings.Join(endpoints, ",")
}

// ping swarm API to check if swarm status OK
func isSwarmOK(endpoint string) bool {
	url := fmt.Sprintf("http://%s/_ping", endpoint)
	resp, err := http.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "malformed HTTP response") {
			// return true here because swarm refuse to return any formed data over TLS
			// anyway swarm is OK on that port
			return true
		}
		return false
	}
	if resp.StatusCode == http.StatusOK {
		return true
	}
	return false
}

// replace all field in ASCII text file with value
func replaceFileContent(filePath, field, value string) (err error) {
	_, err = os.Stat(filePath)
	if err != nil {
		logrus.Errorf("file[%s] not found: %v", filePath, err)
		return
	}
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		logrus.Errorf("read file error: %v", err)
		return
	}
	data = bytes.Replace(data, []byte(field), []byte(value), -1)
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		logrus.Errorf("write change to file error: %v", err)
		return
	}
	return
}

//copy a yaml file and GenRegistry
func parseRegistry(dstName, srcName string, registry string) (err error) {
	_, err = os.Stat(srcName)
	if err != nil {
		logrus.Errorf("file[%s] not found: %v", srcName, err)
		return
	}

	srcByte, err := ioutil.ReadFile(srcName)
	if err != nil {
		logrus.Errorf("read file error: %v", err)
		return
	}
	dstByte := common.GenRegistry(registry, srcByte)
	err = ioutil.WriteFile(dstName, dstByte, 0644)
	if err != nil {
		logrus.Errorf("write change to file error: %v", err)
		return
	}
	return
}
