package command

import (
	"bytes"
	"errors"

	"github.com/Sirupsen/logrus"
	command "linkernetworks.com/dcos-backend/common/common"
	common "linkernetworks.com/dcos-backend/deployer/common"

	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	PROVIDER_TYPE_OPENSTACK = "openstack"
	PROVIDER_TYPE_AWSEC2    = "amazonec2"
	PROVIDER_TYPE_GOOGLE    = "google"
	PROVIDER_TYPE_GENERIC   = "generic"
	// BASE_SWARM_IMAGE        = "linkerrepository/swarm:2.0.0-1.2.4"
	// BASE_EXHIBITOR_IMAGE    = "linkerrepository/linker_exhibitor:2.0.0-1.5.6"
)

func CreateComponentFile(components entity.Components, storagePath string) (filePath string, err error) {
	machineLog := logrus.WithFields(logrus.Fields{"clustername": components.ClusterName})
	machineLog.Infof("start to get cluster components healthy")
	swarmName := components.SwarmName
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash --swarm ")
	commandTextBuffer.WriteString(swarmName + "` && ")
	commandTextBuffer.WriteString("docker ps | grep linkerrepository > ")
	commandTextBuffer.WriteString(storagePath + "/container.txt")
	machineLog.Infof(commandTextBuffer.String())

	_, tmpErrPut, tmpErr := command.ExecCommand(commandTextBuffer.String())
	if tmpErr != nil {
		err = tmpErr
		machineLog.Infof("tmpErrPut is %v", tmpErrPut)
		return
	}

	filePath = storagePath + "/container.txt"

	return
}

func GetCompanentId(imagename string, filepath string, clustername string) (componenttids []string, err error) {
	getLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	name := strings.Replace(clustername, "-", "", -1)
	getLog.Infof("start to get cluster components healthy")
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("more ")
	commandTextBuffer.WriteString(filepath + " | grep ")
	commandTextBuffer.WriteString("\"" + "[a-z0-9A-Z_-]\\+/" + name + "_" + imagename + "_[0-9]\\+" + "\"")
	commandTextBuffer.WriteString(" | awk " + "'{print $1}'")

	getLog.Infof(commandTextBuffer.String())
	tmpOutPut, tmpErrPut, tmpErr := command.ExecCommand(commandTextBuffer.String())
	if tmpErr != nil {
		err = tmpErr
		getLog.Infof("tmpErrPut is %v", tmpErrPut)
		return
	}
	getLog.Infof("len output is ", len(tmpOutPut))
	if len(tmpOutPut) == 0 {
		return componenttids, nil
	}
	strs := strings.Split(tmpOutPut, "\n")
	for _, str := range strs {
		componenttids = append(componenttids, str)
	}
	return
}

func GetcomponentsInfo(storagePath, swarmName, containerId string) (ip string, status string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash --swarm ")
	commandTextBuffer.WriteString(swarmName + "` && ")
	commandTextBuffer.WriteString("docker inspect --format " + "\"" + "{{.Node.IP}}:{{.State.Status}}" + "\"" + " " + containerId)
	logrus.Infof(commandTextBuffer.String())

	tmpOutPut, tmpErrPut, tmpErr := command.ExecCommand(commandTextBuffer.String())
	if tmpErr != nil {
		err = tmpErr
		logrus.Infof("tmpErrPut is %v", tmpErrPut)
		return
	}

	strs := strings.Split(tmpOutPut, ":")
	ip = strs[0]
	status = strs[1]
	return
}

func CreateMachine(hostname, storagePath string, properties map[string]string, engineOpts []entity.EngineOpt, privateNicName string, labels []entity.Label, zkurl string, registries []entity.DockerRegistry) (output string, errput string, err error) {
	clustername := getName(storagePath)
	machineLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	machineLog.Infof("Prepare command to create docker machine: \n")

	engine_url := common.UTIL.Props.GetString("docker.engine.url", "")

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("create ")
	if len(zkurl) > 0 {
		commandTextBuffer.WriteString("--engine-opt=" + "\"cluster-store=" + zkurl + "\" ")
		commandTextBuffer.WriteString("--engine-opt=" + "\"cluster-advertise=" + privateNicName + ":2376\" ")
	}

	if registries != nil && len(registries) != 0 {
		for _, registry := range registries {
			if strings.Contains(registry.Registry, "//") {
				machineLog.Errorln("docker registry url contains protocol string //")
				err = errors.New("docker registry url contains protocol string //")
				return
			}
			if registry.Secure {
				commandTextBuffer.WriteString("--engine-registry-mirror=https://" + registry.Registry + " ")
			} else {
				commandTextBuffer.WriteString("--engine-registry-mirror=http://" + registry.Registry + " ")
				commandTextBuffer.WriteString("--engine-insecure-registry=" + registry.Registry + " ")
			}
		}
	}

	//build customized engine opts
	engineOptValues := []string{}
	for _, opt := range engineOpts {
		if opt.OptKey != "" {
			if opt.OptValue != "" {
				optValue := opt.OptKey + "=" + opt.OptValue
				logrus.Debugf("one customized engine opt is %v", opt)
				engineOptValues = append(engineOptValues, optValue)
			} else {
				engineOptValues = append(engineOptValues, opt.OptKey)
			}
		}
	}

	for _, label := range labels {
		commandTextBuffer.WriteString("--engine-label=\"" + label.Key + "=" + label.Value + "\" ")
	}

	for _, enginopt := range engineOptValues {
		commandTextBuffer.WriteString("--engine-opt=" + enginopt + " ")
	}

	if len(engine_url) > 0 {
		commandTextBuffer.WriteString("--engine-install-url  \"" + engine_url + "\" ")
	}

	for key, value := range properties {
		if key == "google-use-internal-ip" && value == "true" {
			commandTextBuffer.WriteString("--" + key + " ")
			continue
		}
		commandTextBuffer.WriteString("--" + key + " " + value + " ")
	}

	commandTextBuffer.WriteString(hostname)

	machineLog.Infof("Executing create machine command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func DeleteAllMachines(storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	deleLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	deleLog.Infof("Prepare command to delete all docker machines: %s \n", storagePath)
	if needExecRm(storagePath) {
		var commandTextBuffer bytes.Buffer
		commandTextBuffer.WriteString("docker-machine ")
		commandTextBuffer.WriteString("--storage-path " + storagePath + "/ ")
		commandTextBuffer.WriteString("rm -f `")
		commandTextBuffer.WriteString("docker-machine ")
		commandTextBuffer.WriteString("--storage-path " + storagePath + "/ ls -q` ")

		deleLog.Infof("Executing delete all machines command: %s", commandTextBuffer.String())
		output, errput, err = command.ExecCommand(commandTextBuffer.String())
		return output, errput, err
	} else {
		deleLog.Infof("cluster %s doesn't exist or no node in cluster, no need to remove it", storagePath)
		return
	}

}

func DeleteClusterFolder(machinefolder, swarmfolder string) (output string, errput string, err error) {
	logrus.Infof("prepare command to delete cluster folder %s, %s", machinefolder, swarmfolder)
	commandStr := fmt.Sprintf("rm -rf  %s %s ", machinefolder, swarmfolder)
	output, errput, err = command.ExecCommand(commandStr)
	return
}

func DeleteMachine(hostname, storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	delemcLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	delemcLog.Infof("Prepare command to delete docker machines: %s \n", hostname)
	if hostExist(hostname, storagePath) {
		var commandTextBuffer bytes.Buffer
		commandTextBuffer.WriteString("docker-machine ")
		commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
		commandTextBuffer.WriteString("rm -f ")
		commandTextBuffer.WriteString(hostname)

		delemcLog.Infof("Executing delete machine command: %s", commandTextBuffer.String())
		output, errput, err = command.ExecCommand(commandTextBuffer.String())
		return output, errput, err
	} else {
		delemcLog.Infof("host %s doesn't exist under %s, no need to remove it", hostname, storagePath)
		return
	}
}

func hostExist(hostname, storagePath string) bool {
	clustername := getName(storagePath)
	if len(hostname) <= 0 {
		return false
	}
	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("check host %s existence under %s", hostname, storagePath)
	hostFolder := storagePath + "/machines/" + hostname

	_, errInfile := os.Stat(hostFolder)
	isExisted := errInfile == nil || os.IsExist(errInfile)
	return isExisted
}

//if no node under "xxx/machines", "docker-machine rm" command will failed
func needExecRm(storagePath string) bool {
	clustername := getName(storagePath)
	needLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	if !folderExist(storagePath) {
		needLog.Debugf("cluster does not exist! folder is %s", storagePath)
		return false
	}

	clusterMachines := storagePath + "/machines"
	if !folderExist(clusterMachines) {
		needLog.Debugf("cluster machines folder does not exist %s", clusterMachines)
		return false
	}

	empty, err := isEmptyFolder(clusterMachines)
	if err != nil {
		needLog.Warnf("check folder %s, error %v", clusterMachines, err)
		return true
	}

	return !empty

}

func isEmptyFolder(name string) (bool, error) {
	logrus.Infof("check folder empty or not: %s", name)
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}

func folderExist(storagePath string) bool {
	logrus.Infof("check folder existence  %s", storagePath)

	_, errInfile := os.Stat(storagePath)
	isExisted := errInfile == nil || os.IsExist(errInfile)
	return isExisted
}

func ExecCommandOnMachine(hostname, commandstr, storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("ssh ")
	commandTextBuffer.WriteString(hostname)
	commandTextBuffer.WriteString(" -t \"")
	commandTextBuffer.WriteString(commandstr)
	commandTextBuffer.WriteString(" \"")

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing ssh command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func ScpToMachine(hostname, localpath, remotepath, storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("scp ")
	commandTextBuffer.WriteString(localpath + " ")
	commandTextBuffer.WriteString(hostname + ":")
	commandTextBuffer.WriteString(remotepath)
	commandTextBuffer.WriteString("")

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing scp command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func LsNodesCheck(storagePath string) (output string, errput string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("ls --format ")
	commandTextBuffer.WriteString(" \"{{.Name}}:{{.State}}:{{.DockerVersion}}\"")

	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func ScpFolderToMachine(hostname, localpath, remotepath, storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("scp -r ")
	commandTextBuffer.WriteString(localpath + " ")
	commandTextBuffer.WriteString(hostname + ":")
	commandTextBuffer.WriteString(remotepath)
	commandTextBuffer.WriteString("")

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing scp command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func InspectMachineByKey(hostname, key, storagePath string) (output string, errput string, err error) {
	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("inspect ")
	commandTextBuffer.WriteString(hostname + " ")
	commandTextBuffer.WriteString("-f ")
	commandTextBuffer.WriteString("{{" + key + "}}")

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing inspect command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func GetMachinePublicIPAddress(hostname, storagePath string) (ipaddress string, err error) {
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("ip ")
	commandTextBuffer.WriteString(hostname)

	logrus.Infof("Executing ssh command: %s", commandTextBuffer.String())
	output, _, err := command.ExecCommand(commandTextBuffer.String())
	if err != nil {
		logrus.Warnf("getting public ip error %v", err)
		return "", err
	}

	return strings.TrimSpace(output), nil
}

func GetMachinePrivateIPAddress(hostname, privateNicName, storagePath string) (ipaddress string, err error) {
	clustername := getName(storagePath)
	getgenLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	commandstr := fmt.Sprintf("sudo ip addr show %s|grep 'inet.*brd.*%s'|head -1 |awk '{print $2}'|awk -F/ '{print $1}'", privateNicName, privateNicName)

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("ssh ")
	commandTextBuffer.WriteString(hostname)
	commandTextBuffer.WriteString(" -t ")
	commandTextBuffer.WriteString(commandstr)
	commandTextBuffer.WriteString(" ")

	getgenLog.Infof("Executing ssh command: %s", commandTextBuffer.String())
	output, errput, err := command.ExecCommand(commandTextBuffer.String())
	if err != nil {
		getgenLog.Warnf("getting private ip error %v", err)
		return "", err
	}

	getgenLog.Debugf("PrivateIP for  machine output: %s", output)
	getgenLog.Debugf("PrivateIP for  machine errput: %s", errput)

	return strings.TrimSpace(output), nil
}

func BootUpSwarmMaster(hostname, ipAddr, storagePath string, zks string, imageRegistry string) (output, errput string, err error) {

	BASE_SWARM_IMAGE := common.UTIL.Props.MustGetString("base_swarm_image")
	swarmImage := BASE_SWARM_IMAGE
	if !strings.EqualFold(imageRegistry, "") {

		imageRegistry = strings.TrimSuffix(imageRegistry, "/")
		swarmImage = imageRegistry + "/" + BASE_SWARM_IMAGE
	}

	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(hostname + ") ")

	commandTextBuffer.WriteString("run -d --name swarm-agent-master --restart=always --net=bridge -p 3376:3376  -v /etc/docker:/etc/docker " + swarmImage + " manage --tlsverify --tlscacert=/etc/docker/ca.pem  --tlscert=/etc/docker/server.pem  --tlskey=/etc/docker/server-key.pem -H tcp://0.0.0.0:3376  --replication --strategy spread --advertise ")
	commandTextBuffer.WriteString(ipAddr + ":3376  ")
	commandTextBuffer.WriteString(zks)

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing BootUpSwarmMaster command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func BootUpSwarmAgent(hostname, ipAddr, storagePath, zks string, imageRegistry string) (output, errput string, err error) {

	BASE_SWARM_IMAGE := common.UTIL.Props.MustGetString("base_swarm_image")
	swarmImage := BASE_SWARM_IMAGE
	if !strings.EqualFold(imageRegistry, "") {

		imageRegistry = strings.TrimSuffix(imageRegistry, "/")
		swarmImage = imageRegistry + "/" + BASE_SWARM_IMAGE
	}

	clustername := getName(storagePath)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(hostname + ") ")

	commandTextBuffer.WriteString("run -d --name swarm-agent --restart=always --net=bridge " + swarmImage + " join --advertise ")
	commandTextBuffer.WriteString(ipAddr + ":2376  ")
	commandTextBuffer.WriteString(zks)

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing BootUpSwarmAgent command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

//bootup exhibitor(zookeeper) ethName: private nic name(eth0), zkList: all zookeeper node(172.31.3.27:2888:3888,172.31.3.4:2888:3888,172.31.12.213:2888:3888)
func BootUpZookeeper(hostname, storagePath string, ethName, zkList string, imageRegistry string, i string) (output, errput string, err error) {

	BASE_EXHIBITOR_IMAGE := common.UTIL.Props.MustGetString("base_exhibitor_image")
	exhibitorImage := BASE_EXHIBITOR_IMAGE
	if !strings.EqualFold(imageRegistry, "") {
		imageRegistry = strings.TrimSuffix(imageRegistry, "/")
		exhibitorImage = imageRegistry + "/" + BASE_EXHIBITOR_IMAGE
	}

	clustername := getName(storagePath)
	name := strings.Replace(clustername, "-", "", -1)
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(hostname + ") ")

	commandTextBuffer.WriteString("run -d --restart=always --net=host -v /opt/zookeeper/snapshot:/opt/zookeeper/snapshot -v /opt/zookeeper/transactions:/opt/zookeeper/transactions ")
	commandTextBuffer.WriteString(" -e " + ethName)
	commandTextBuffer.WriteString(" -e ZOOKEEPERLIST=" + zkList)
	commandTextBuffer.WriteString(" --name=" + name + "_exhibitor_" + i)
	commandTextBuffer.WriteString(" " + exhibitorImage + " ")

	logrus.WithFields(logrus.Fields{"clustername": clustername}).Infof("Executing BootUpZookeeper command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	return
}

func CleanUp(hostname, storagePath string) (err error) {

	//clean containers
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("eval ")
	commandTextBuffer.WriteString("`docker-machine ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("env --shell bash  ")
	commandTextBuffer.WriteString(hostname + "` && ")
	commandTextBuffer.WriteString(" docker stop `docker ps|grep swarm | awk '{print $1}'` ")

	_, _, err = command.ExecCommand(commandTextBuffer.String())
	if err != nil {
		return
	}

	//clean mesos and stop mesos-agent
	cleanCommand := "sudo systemctl stop dcos-mesos-slave && sudo systemctl disable dcos-mesos-slave  &&  sudo rm -rf /var/lib/mesos "
	_, _, err = ExecCommandOnMachine(hostname, cleanCommand, storagePath)

	return
}

func ChangeConfigFile(hostname, zkList, storagePath string, isSwarm bool, imageRegistry string) (err error) {

	clustername := getName(storagePath)
	configFile, err := getConfigFile(hostname, storagePath)

	BASE_SWARM_IMAGE := common.UTIL.Props.MustGetString("base_swarm_image")
	swarmImage := BASE_SWARM_IMAGE
	if !strings.EqualFold(imageRegistry, "") {

		imageRegistry = strings.TrimSuffix(imageRegistry, "/")
		swarmImage = imageRegistry + "/" + BASE_SWARM_IMAGE
	}

	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clustername}).Error(err)
		return err
	}
	defer configFile.Close()

	content, err := ioutil.ReadFile(configFile.Name())
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	discoverValue := zkList
	configFile.Truncate(0)
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if isSwarm {
			if strings.HasPrefix(value, "\"IsSwarm\"") {
				configFile.WriteString("\"IsSwarm\": true,")
			} else if strings.HasPrefix(value, "\"Discovery\"") {
				configFile.WriteString("\"Discovery\": " + "\"" + discoverValue + "\",")
			} else if strings.HasPrefix(value, "\"Agent\"") {
				configFile.WriteString("\"Agent\": true,")
			} else if strings.HasPrefix(value, "\"Master\"") {
				configFile.WriteString("\"Master\": true,")
			} else if strings.HasPrefix(value, "\"Image\"") {
				configFile.WriteString("\"Image\": " + "\"" + swarmImage + "\",")
			} else {
				configFile.WriteString(line)
			}
		} else {
			if strings.HasPrefix(value, "\"IsSwarm\"") {
				configFile.WriteString("\"IsSwarm\": true,")
			} else if strings.HasPrefix(value, "\"Discovery\"") {
				configFile.WriteString("\"Discovery\": " + "\"" + discoverValue + "\",")
			} else if strings.HasPrefix(value, "\"Agent\"") {
				configFile.WriteString("\"Agent\": true,")
			} else if strings.HasPrefix(value, "\"Image\"") {
				configFile.WriteString("\"Image\": " + "\"" + swarmImage + "\",")
			} else {
				configFile.WriteString(line)
			}
		}

	}

	return nil
}

func getConfigFile(hostname, storagePath string) (configFile *os.File, err error) {
	clustername := getName(storagePath)
	conLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	absoluteFilePath := storagePath + "/machines" + "/" + hostname + "/config.json"
	bakFilePath := storagePath + "/machines" + "/" + hostname + "/config.jsonbak"

	_, errInfile := os.Stat(absoluteFilePath)
	isExisted := errInfile == nil || os.IsExist(errInfile)
	if isExisted {
		configFile, err = os.OpenFile(absoluteFilePath, os.O_RDWR, 0)
	} else {
		conLog.Errorf("host %s config file does exist!", hostname)
		err = errors.New("specific host's config file exsits!")
		return
	}

	conLog.Debugf("cp config.json to config.jsonbak")
	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("cp ")
	commandTextBuffer.WriteString(absoluteFilePath)
	commandTextBuffer.WriteString(" ")
	commandTextBuffer.WriteString(bakFilePath)
	conLog.Infof("backup config file command %s", commandTextBuffer.String())
	_, _, _ = command.ExecCommand(commandTextBuffer.String())

	return

}

func getName(storagePath string) (clustername string) {
	str := strings.Split(storagePath, "/")
	if len(str) >= 4 {
		clustername = str[4]
	}
	return
}
