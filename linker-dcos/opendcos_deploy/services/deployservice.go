package services

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"linkernetworks.com/opendcos_deploy/common"
	"linkernetworks.com/opendcos_deploy/common/entity"

	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	BASE_PATH = "/opendcos/clusters/"

	DEPLOY_ERROR_INVALIDATE_CLUSTERNAME string = "INVALIDATE_CLUSTERNAME"
	DEPLOY_ERROR_NODE_NOT_EXISTED       string = "NODE_NOT_EXISTED"
	DEPLOY_ERROR_CLUSTER_NOT_EXISTED    string = "CLUSTER_NOT_EXISTED"
)

func CreateCluster(request entity.CreateRequest) (err error) {

	logrus.Infof("start createCluster...")
	logrus.Infof("createRequset: %v", request)

	config := request.Config
	clusterName := config.Cluster_name
	username := request.UserName

	//check if username&clusterName validate
	if !checkName(username, clusterName) {
		logrus.Errorf("CreateCluster, invalidate clusterName.")
		err = errors.New(DEPLOY_ERROR_INVALIDATE_CLUSTERNAME)
		return
	}

	//generate clusterDir
	clusterDir := genClusterDir(username, clusterName)

	//preparation
	err = preparation(clusterDir, config, request.PrivateKey, request.PrivateNicName)
	if err != nil {
		logrus.Errorf("createCluster, preparation failed. err is %v", err)
		return
	}

	//preCheck
	err = preCheck(clusterName, clusterDir, request.SkipInstall)
	if err != nil {
		logrus.Errorf("createCluster, preCheck failed. err is %v", err)
		return
	}

	//provision
	err = provision(clusterName, clusterDir)
	if err != nil {
		logrus.Errorf("createCluster, provision failed. err is %v", err)
		return
	}

	//postAction
	err = postAction(clusterName, clusterDir)
	if err != nil {
		logrus.Errorf("createCluster, postAction failed. err is %v", err)
		return
	}

	//backup
	err = backup(clusterDir)
	if err != nil {
		logrus.Errorf("createCluster, backup failed. err is %v", err)
		return
	}

	//remove tar package
	commandStr := "sudo rm -f " + clusterDir + "*.tar"
	logrus.Infof("CreateCluster, execute command: %s", commandStr)
	output, errput, err := common.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("CreateCluster, ExecCommand err: %v", err)
		logrus.Infof("CreateCluster %s, errput: %s", clusterName, errput)
		return
	}
	logrus.Infof("CreateCluster %s, output: %s", clusterName, output)

	return
}

func DeleteCluster(username string, clusterName string) (err error) {

	logrus.Infof("start DeleteCluster... username: %s, clusterName: %s", username, clusterName)

	//generate clusterDir
	clusterDir := genClusterDir(username, clusterName)
	privateKeyPath := clusterDir + "genconf/ssh_key"

	//check if cluster exists
	exist, _ := common.PathExist(clusterDir)
	if !exist {
		logrus.Errorf("DeleteCluster, cluster not existed. clusterDir: %s", clusterDir)
		return errors.New(DEPLOY_ERROR_CLUSTER_NOT_EXISTED)
	}

	//find added slaves from cluster dir
	nodes, _ := getNodes(clusterDir)
	if nodes != nil {
		sshUser, _, _ := getSshUserAndPort(clusterDir)
		for _, nodeip := range nodes {
			go deleteSingleNode(clusterDir, nodeip, sshUser, privateKeyPath)
		}
	}

	//execute --uninstall
	logrus.Infof("DeleteCluster, execute command: yes | sudo bash /opendcos/dcos_generate_config.sh --uninstall --verbose")
	output, errput, err := common.ExecCommandinDir(clusterDir, "yes | sudo bash /opendcos/dcos_generate_config.sh --uninstall --verbose"+" >> opendcos_deployer.log 2>&1")
	if err != nil {
		logrus.Errorf("DeleteCluster, ExecCommand err: %v", err)
		logrus.Infof("DeleteCluster %s, errput: %s", clusterName, errput)
		return
	}
	logrus.Infof("DeleteCluster %s, output: %s", clusterName, output)

	cleanup(clusterDir)

	return

}

func AddNodes(request entity.AddNodeRequest) (err error) {

	logrus.Infof("start AddNodes...")
	logrus.Infof("AddNodeRequest: %v", request)

	clusterName := request.ClusterName
	username := request.UserName
	nodes := request.Nodes

	//generate clusterDir
	clusterDir := genClusterDir(username, clusterName)
	privateKeyPath := clusterDir + "genconf/ssh_key"

	//check if cluster exists
	exist, _ := common.PathExist(clusterDir)
	if !exist {
		logrus.Errorf("AddNodes, cluster not existed. clusterDir: %s", clusterDir)
		return errors.New(DEPLOY_ERROR_CLUSTER_NOT_EXISTED)
	}

	//check if dcos-installer.tar exists
	exist, _ = common.PathExist(clusterDir + "genconf/serve/dcos-install.tar")
	if !exist {
		logrus.Infof("AddNodes, backup file not exists. try backup...")
		errb := backup(clusterDir)
		if errb != nil {
			logrus.Errorf("AddNodes, backup failed, err is %v", err)
			return
		}
	}

	//start to add nodes
	for _, node := range nodes {

		go addSingleNode(node.Ip, request.SshUser, privateKeyPath, clusterDir, node.SlaveType, node.SkipInstall)

	}

	return
}

func DeleteNode(username string, clusterName string, ip string) (err error) {

	logrus.Infof("start DeleteNode...")

	//generate clusterDir
	clusterDir := genClusterDir(username, clusterName)

	//check if node exists
	exist, _ := nodeExists(clusterDir, ip)
	if !exist {
		logrus.Errorf("DeleteNode, node %s not exists", ip)
		return errors.New(DEPLOY_ERROR_NODE_NOT_EXISTED)
	}

	sshUser, _, _ := getSshUserAndPort(clusterDir)
	privateKeyPath := clusterDir + "genconf/ssh_key"

	go deleteSingleNode(clusterDir, ip, sshUser, privateKeyPath)

	return
}

//prepare for deploy, create cluster directory, and execute --genconf
func preparation(clusterDir string, config entity.DCOSConfig, privateKey string, privateNicName string) (err error) {

	clusterName := config.Cluster_name

	logrus.Infof("start preparation... clusterDir: %s, config:%v, privateKey: %s, privateNicName: %s",
		clusterDir, config, privateKey, privateNicName)

	_, _, err = common.ExecCommand("sudo mkdir -p " + clusterDir)
	if err != nil {
		logrus.Errorf("Preparation, ExecCommand err: %v", err)
		return
	}
	_, _, err = common.ExecCommand("sudo mkdir -p " + clusterDir + "genconf")
	if err != nil {
		logrus.Errorf("Preparation, ExecCommand err: %v", err)
		return
	}

	genConf(clusterDir+"genconf/", config)
	genIPDetect(clusterDir+"genconf/", privateNicName)
	genSshKey(clusterDir+"genconf/", privateKey)

	//execute --genconf
	logrus.Infof("Preparation, execute command: sudo bash /opendcos/dcos_generate_config.sh --genconf --verbose")
	output, errput, err := common.ExecCommandinDir(clusterDir, "sudo bash /opendcos/dcos_generate_config.sh --genconf --verbose"+" >> opendcos_deployer.log 2>&1")
	if err != nil {
		logrus.Errorf("Preparation, ExecCommand err: %v", err)
		logrus.Infof("Preparation for cluster %s, errput: %s", clusterName, errput)
		//cleanup()
		return
	}
	logrus.Infof("Preparation for cluster %s, output: %s", clusterName, output)
	logrus.Infof("Preparation finished clusterName: %s, clusterDir: %s", clusterName, clusterDir)

	return
}

//check config to ensure the cluster can be deployed, execute --install-prereqs & --preflight
//not execute --install-prereqs if skipInstall is true
func preCheck(clusterName string, clusterDir string, skipInstall bool) (err error) {

	logrus.Infof("start preCheck... clusterName: %s, clusterDir: %s, skipInstall: %t", clusterName, clusterDir, skipInstall)

	//skipInstall is false, execute --install-prereqs
	if !skipInstall {
		logrus.Infof("preCheck, execute command: sudo bash /opendcos/dcos_generate_config.sh --install-prereqs --verbose")
		output, errput, err1 := common.ExecCommandinDir(clusterDir, "sudo bash /opendcos/dcos_generate_config.sh --install-prereqs --verbose"+" >> opendcos_deployer.log 2>&1")
		if err1 != nil {
			logrus.Errorf("preCheck --install-prereqs, ExecCommand err: %v", err)
			logrus.Infof("preCheck --install-prereqs for cluster %s, errput: %s", clusterName, errput)
			//cleanup()
			return err1
		}
		logrus.Infof("preCheck --install-prereqs for cluster %s, output: %s", clusterName, output)
	}

	//execute --preflight
	logrus.Infof("preCheck, execute command: sudo bash /opendcos/dcos_generate_config.sh --preflight --verbose")
	output, errput, err := common.ExecCommandinDir(clusterDir, "sudo bash /opendcos/dcos_generate_config.sh --preflight --verbose"+" >> opendcos_deployer.log 2>&1")
	if err != nil {
		logrus.Errorf("preCheck --preflight, ExecCommand err: %v", err)
		logrus.Infof("preCheck --preflight for cluster %s, errput: %s", clusterName, errput)
		//cleanup()
		return
	}
	logrus.Infof("preCheck --preflight for cluster %s, output: %s", clusterName, output)
	logrus.Infof("preCheck finished. clusterName: %s, clusterDir: %s", clusterName, clusterDir)

	return
}

//deploy cluster, execute --deploy
func provision(clusterName string, clusterDir string) (err error) {

	logrus.Infof("start provision... clusterName: %s, clusterDir: %s", clusterName, clusterDir)

	//execute --deploy
	logrus.Infof("provision, execute command: sudo bash /opendcos/dcos_generate_config.sh --deploy --verbose")
	output, errput, err := common.ExecCommandinDir(clusterDir, "sudo bash /opendcos/dcos_generate_config.sh --deploy --verbose"+" >> opendcos_deployer.log 2>&1")
	if err != nil {
		logrus.Errorf("provision --deploy, ExecCommand err: %v", err)
		logrus.Infof("provision --deploy for cluster %s, errput: %s", clusterName, errput)
		//cleanup()
		return
	}
	logrus.Infof("provision --deploy for cluster %s, output: %s", clusterName, output)
	logrus.Infof(" provision finished. clusterName: %s, clusterDir: %s", clusterName, clusterDir)
	return
}

//postaction of deployment, exec --postflight
func postAction(clusterName string, clusterDir string) (err error) {

	logrus.Infof("start postAction... clusterName: %s, clusterDir: %s", clusterName, clusterDir)

	//execute --postflight
	logrus.Infof("postAction, execute command: sudo bash /opendcos/dcos_generate_config.sh --postflight --verbose")
	output, errput, err := common.ExecCommandinDir(clusterDir, "sudo bash /opendcos/dcos_generate_config.sh --postflight --verbose"+" >> opendcos_deployer.log 2>&1")
	if err != nil {
		logrus.Errorf("postAction --postflight, ExecCommand err: %v", err)
		logrus.Infof("postAction --postflight for cluster %s, errput: %s", clusterName, errput)
		//cleanup()
		return
	}
	logrus.Infof("postAction --postflight for cluster %s, output: %s", clusterName, output)
	logrus.Infof("postAction finished. clusterName: %s, clusterDir: %s", clusterName, clusterDir)
	return
}

//backup dcos-install.tar
func backup(clusterDir string) (err error) {

	logrus.Infof("start backup...  clusterDir: %s", clusterDir)

	//command: sudo tar cf $clusterDir/genconf/serve/dcos-install.tar -C $clusterDir/genconf/serve .
	commandStr := "sudo tar cf " + clusterDir + "genconf/serve/dcos-install.tar -C " + clusterDir + "genconf/serve ."
	logrus.Infof("backup, execute command: %s", commandStr)
	_, errput, err := common.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("backup tar cf failed, ExecCommand err: %v", err)
		logrus.Infof("backup tar cf, errput: %s", errput)
		return
	}

	logrus.Infof("backup finished. clusterDir: %s", clusterDir)

	return
}

//add single node, for loop in AddNodes
func addSingleNode(nodeip string, sshUser string, privateKeyPath string, clusterDir string, slaveType string, skipInstall bool) {

	logrus.Infof("add node %s ...", nodeip)

	if !skipInstall {

		logrus.Infof("do install-prereqs on node %s", nodeip)

		//scp -i $ssh_key script/install_prereqs.sh $(sshuser)@$(nodeip):/tmp/install_prereqs.sh
		commandStr := "scp -oConnectTimeout=10 -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -oBatchMode=yes -oPasswordAuthentication=no -i " + privateKeyPath + " " +
			"script/install_prereqs.sh " + sshUser + "@" + nodeip + ":/tmp/install_prereqs.sh"

		logrus.Infof("execute command: %s", commandStr)
		_, errput, err := common.ExecCommand(commandStr)
		if err != nil {
			logrus.Errorf("AddNodes, ExecCommand err: %v", err)
			logrus.Infof("add node %s failed failed, errput: %s", nodeip, errput)
			return
		}

		commandStr = "sudo sh /tmp/install_prereqs.sh"
		logrus.Infof("execute install-prereqs on node: %s", nodeip)
		output, errput, err := common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
		if err != nil {
			logrus.Errorf("AddNodes, ExecCommand err: %v", err)
			logrus.Infof("add node %s failed, errput: %s", nodeip, errput)
			return
		}
		logrus.Infof("install-prereqs on node: %s finished.", nodeip, output)

	}

	//scp -i $ssh_key $clusterDir/genconf/serve/dcos-install.tar $(sshuser)@$(nodeip):/tmp/dcos-install.tar
	commandStr := "scp -oConnectTimeout=10 -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -oBatchMode=yes -oPasswordAuthentication=no -i " + privateKeyPath + " " +
		clusterDir + "genconf/serve/dcos-install.tar " + sshUser + "@" + nodeip + ":/tmp/dcos-install.tar"

	logrus.Infof("execute command: %s", commandStr)
	_, errput, err := common.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("AddNodes, ExecCommand err: %v", err)
		logrus.Infof("add node %s failed failed, errput: %s", nodeip, errput)
		return
	}

	commandStr = "sudo mkdir -p /opt/dcos_install_tmp"
	_, errput, err = common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("AddNodes, ExecCommand err: %v", err)
		logrus.Infof("add node %s failed, errput: %s", nodeip, errput)
		return
	}

	commandStr = "sudo tar xf /tmp/dcos-install.tar -C /opt/dcos_install_tmp"
	_, errput, err = common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("AddNodes, ExecCommand err: %v", err)
		logrus.Infof("add node %s failed, errput: %s", nodeip, errput)
		return
	}

	commandStr = "echo --------add node to cluster: " + clusterDir + ", nodeip: " + nodeip + " >> " + clusterDir + "opendcos_addnode.log"
	_, errput, err = common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("AddNodes, ExecCommand err: %v", err)
		logrus.Infof("add node %s failed, errput: %s", nodeip, errput)
		return
	}

	commandStr = "sudo bash /opt/dcos_install_tmp/dcos_install.sh " + slaveType + " >> " + clusterDir + "opendcos_addnode.log"
	_, errput, err = common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("AddNodes, ExecCommand err: %v", err)
		logrus.Infof("add node %s failed, errput: %s", nodeip, errput)
		return
	}
	logrus.Infof("Addnodes, record nodeip %s", nodeip)
	recordNodeip(clusterDir, nodeip)
	logrus.Infof("add node %s succeeded", nodeip)
	return
}

//delete single node
func deleteSingleNode(clusterDir string, nodeip string, sshUser string, privateKeyPath string) {

	logrus.Infof("delete node %s ...", nodeip)

	commandStr := "sudo -i /opt/mesosphere/bin/pkgpanda uninstall"
	output, errput, err := common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("DeleteNode, ExecCommand err: %v", err)
		logrus.Infof("DeleteNode failed, errput: %s", errput)
		return
	}
	logrus.Infof("command output: %s", output)

	commandStr = "sudo rm -rf /opt/mesosphere /etc/mesosphere"
	_, errput, err = common.SshExecCmdWithKey(nodeip, "22", sshUser, privateKeyPath, commandStr)
	if err != nil {
		logrus.Errorf("DeleteNode, ExecCommand err: %v", err)
		logrus.Infof("DeleteNode failed, errput: %s", errput)
		return
	}

	logrus.Infof("deleteSingleNode, rmnoderecord nodeip %s", nodeip)
	rmNodeRecord(clusterDir, nodeip)
	logrus.Infof("delete node %s succeeded", nodeip)
}

//check if username & clusterName validate
func checkName(username string, clusterName string) (validate bool) {

	//check if existed
	clusterDir := genClusterDir(username, clusterName)
	exist, _ := common.PathExist(clusterDir)
	if !exist {
		validate = true
		return
	}
	validate = false
	return
}

//generate config.yaml
//path should be absolute and end with "/"
func genConf(path string, config entity.DCOSConfig) (err error) {

	logrus.Infof("start create file config.yaml... path: %s, config: %v", path, config)

	logrus.Infof("genConf, slaves:\n%v", config.Agent_list)
	logrus.Infof("genConf, masters:\n%v", config.Master_list)

	fileBytes, err := yaml.Marshal(&config)
	if err != nil {
		logrus.Error("genConf, generate config.yaml failed. error is %v", err)
		return
	}
	logrus.Debugf("genConf, config.yaml:\n", string(fileBytes))

	err = ioutil.WriteFile(path+"config.yaml", fileBytes, 0644)
	if err != nil {
		logrus.Errorf("genConf failed, err is %v", err)
		return
	}
	logrus.Infof("file config.yaml created.")
	return
}

//generate ip-detect
//path should be absolute and end with "/"
func genIPDetect(path string, privateNicName string) (err error) {

	logrus.Infof("start create file ip-detect... path: %s", path)

	fileStr := `#!/usr/bin/env bash
set -o nounset -o errexit
export PATH=/usr/sbin:/usr/bin:$PATH
echo $(ip addr show ` + privateNicName + ` | grep -Eo '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}' | head -1)
`

	logrus.Debugf("genIPDetect, ip-detect:\n%s", fileStr)
	err = ioutil.WriteFile(path+"ip-detect", []byte(fileStr), 0644)
	if err != nil {
		logrus.Errorf("genIPDetect failed, err is %v", err)
		return
	}
	logrus.Infof("file ip-detect created.")
	return
}

//generate ssh_key
//path should be absolute and end with "/"
func genSshKey(path string, privateKey string) (err error) {

	logrus.Infof("start create file ssh_key... path: %s, privateKey:\n%s", path, privateKey)

	decodeKey, err := common.Base64Decode([]byte(privateKey))
	if err != nil {
		logrus.Errorf("decode private key error %v", err)
		return
	}

	logrus.Infof("the decoded privateKey:\n%s", decodeKey)

	err = ioutil.WriteFile(path+"ssh_key", []byte(decodeKey), 0600)
	if err != nil {
		logrus.Errorf("genSshKey failed, err is %v", err)
	}
	logrus.Infof("file ssh_key created.")
	return
}

//generate cluster directory with username & clusterName
func genClusterDir(username string, clusterName string) (clusterDir string) {

	clusterDir = BASE_PATH + username + "/" + clusterName + "/"
	return
}

//download dcos_generate_config.sh
func DownloadInstaller() (err error) {

	logrus.Infof("DownloadInstaller, downloading dcos_generate_config.sh. this may take some time...")

	//check if file exists
	exist, _ := common.PathExist("/opendcos/dcos_generate_config.sh")
	if exist {
		logrus.Infof("DownloadInstaller, dcos_generate_config.sh existed.")
		return
	}

	commandStr := "mkdir -p /opendcos/"
	logrus.Infof("execute command: %s", commandStr)
	_, errput, err := common.ExecCommand(commandStr)
	if err != nil {
		logrus.Errorf("DownloadInstaller, ExecCommand err: %v", err)
		logrus.Infof("DownloadInstaller failed, errput: %s", errput)
		return
	}

	commandStr = "curl -o /opendcos/dcos_generate_config.sh https://downloads.dcos.io/dcos/stable/dcos_generate_config.sh"
	logrus.Infof("execute command: %s", commandStr)
	output, errput, err := common.ExecCommand(commandStr)
	logrus.Infof(output)
	if err != nil {
		logrus.Errorf("DownloadInstaller, ExecCommand err: %v", err)
		logrus.Infof("DownloadInstaller failed, errput: %s", errput)
		return
	}
	return

}

//remove the cluster directory
func cleanup(clusterDir string) (err error) {

	logrus.Infof("start cleanup... clusterDir: %s", clusterDir)

	_, errput, err := common.ExecCommand("sudo rm -rf " + clusterDir)
	if err != nil {
		logrus.Errorf("cleanup error, err is %s, errput: %s", err, errput)
	}

	logrus.Infof("cleanup clusterDir: %s, finished", clusterDir)
	return
}

//get ssh_user and ssh_port in config.yaml
func getSshUserAndPort(clusterDir string) (sshUser string, sshPort string, err error) {

	config := clusterDir + "genconf/" + "config.yaml"
	configByte, err := ioutil.ReadFile(config)
	if err != nil {
		logrus.Errorf("getSshUserAndPort, read file %s failed, err is %v", config, err)
		return "root", "22", nil
	}

	m := make(map[interface{}]interface{})

	err = yaml.Unmarshal(configByte, &m)

	if err != nil {
		logrus.Errorf("getSshUserAndPort, parse config.yaml failed, err is %v", err)
		return "root", "22", nil
	}

	sshUser, ok := m["ssh_user"].(string)
	if !ok {
		sshUser = "root"
	}
	sshPort, _ = m["ssh_port"].(string)
	if !ok {
		sshPort = "22"
	}

	return
}

//get all node ip of cluster
func getNodes(clusterDir string) (nodeip []string, err error) {

	fileName := clusterDir + "addSlaves"
	exist, _ := common.PathExist(fileName)
	if !exist {
		//logrus.Infof("getNodeIp, file %s not exists", fileName)
		return nil, nil

	}

	nodeByte, err := ioutil.ReadFile(fileName)
	if err != nil {
		logrus.Errorf("getNodeIp, read file %s failed, err is %v", fileName, err)
		return nil, err
	}
	nodeip = strings.Split(strings.TrimSuffix(string(nodeByte), ","), ",")
	return
}

//check if nodeip exists in cluster
func nodeExists(clusterDir string, nodeip string) (exist bool, err error) {

	fileName := clusterDir + "addSlaves"
	exist, _ = common.PathExist(fileName)
	if !exist {
		return false, nil
	}

	record, err := ioutil.ReadFile(fileName)
	if err != nil {
		logrus.Errorf("nodeExists, read file %s failed, err is %v", fileName, err)
		return false, err
	}
	return strings.Contains(string(record), nodeip), err
}

//record node ip
func recordNodeip(clusterDir string, nodeip string) (err error) {

	fileName := clusterDir + "addSlaves"

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("recordNodeip, open file %s failed, err is %v", fileName, err)
		return
	}
	defer file.Close()
	_, err = file.WriteString(nodeip + ",")
	if err != nil {
		logrus.Errorf("recordNodeip, WriteString to file %s failed, err is %v", fileName, err)
		return
	}

	return
}

//remove ip record
func rmNodeRecord(clusterDir string, nodeip string) (err error) {

	exist, _ := nodeExists(clusterDir, nodeip)
	if !exist {
		return
	}

	fileName := clusterDir + "addSlaves"
	record, err := ioutil.ReadFile(fileName)
	if err != nil {
		logrus.Errorf("rmNodeRecord, read file %s failed, err is %v", fileName, err)
		return
	}

	recordStr := string(record)
	oldstr := nodeip + ","
	if !strings.Contains(recordStr, ",") {
		logrus.Errorf("node %s not recorded in cluster %s", nodeip, clusterDir)
		return
	} else {
		recordStr = strings.Replace(recordStr, oldstr, "", 1)
	}

	err = ioutil.WriteFile(fileName, []byte(recordStr), 0644)
	if err != nil {
		logrus.Errorf("rmNodeRecord, error when writeFile, err is %v", err)
	}
	return

}
