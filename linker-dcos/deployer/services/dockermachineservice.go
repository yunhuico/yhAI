package services

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	// "github.com/pborman/uuid"
	//	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/common/persistence/entity"

	commonCommand "linkernetworks.com/dcos-backend/common/common"
	command "linkernetworks.com/dcos-backend/deployer/command"
	"linkernetworks.com/dcos-backend/deployer/common"
)

const (
	DOCKERMACHINE_ERROR_STORAGEPATH_CREATE string = "E61001"

	DOCKERMACHINE_STORAGEPATH_PREFIX string = "/linker/docker/"
	// DOCKERMACHINE_STORAGEPATH_PREFIX string = "/linker/docker/"
	DOCKER_LOGIN_SCRIPT_DIR   string = "/linker"
	DOCKER_LOGIN_SCRIPT_PATH  string = "/linker/dockerlogin.sh"
	DOCKER_LOGOUT_SCRIPT_PATH string = "/linker/dockerlogout.sh"

	INSTALL_EXPECT_SCRIPT_DIR  string = "/linker"
	INSTALL_EXPECT_SCRIPT_PATH string = "/linker/install-expect.sh"
)

var (
	dockerMachineService *DockerMachineService = nil
	onceDockerMachine    sync.Once
)

type DockerMachineService struct {
	serviceName string
}

func GetDockerMachineService() *DockerMachineService {
	onceDockerMachine.Do(func() {
		logrus.Debugf("Once called from DockerMachineService ......................................")
		dockerMachineService = &DockerMachineService{"DockerMachineService"}
	})
	return dockerMachineService

}

func (p *DockerMachineService) Create(username, clusername string, properties map[string]string, pubkeyPath []string, mode string, node entity.Node, labels []entity.Label, zkurl string, registries []entity.DockerRegistry, engineOpts []entity.EngineOpt, privateKeyPath string) (
	server entity.Server, errorCode string, err error) {
	crLog := logrus.WithFields(logrus.Fields{"clustername": clusername})
	crLog.Infof("start to create Docker Machine...")

	hostname := node.HostName
	if len(hostname) <= 0 {
		// hostname = uuid.New() + "-" + clusername + "-" + username
		hostname = GenerateHostName(clusername, username)
	}

	storagePath := p.ComposeStoragePath(username, clusername)

	err = os.MkdirAll(storagePath, os.ModePerm)
	if err != nil {
		errorCode = DOCKERMACHINE_ERROR_STORAGEPATH_CREATE
		crLog.Errorf("DOCKERMACHINE_ERROR_STORAGEPATH_CREATE: %v", err)
		return
	}

	pro, erro := p.buildProperties(properties, engineOpts, mode, node, privateKeyPath)
	if erro != nil {
		crLog.Errorf("build create command error %v", erro)
		return server, errorCode, erro
	}

	server = entity.Server{Hostname: hostname, SshUser: node.SshUser}
	_, errput, errc := command.CreateMachine(hostname, storagePath, pro, engineOpts, node.PrivateNicName, labels, zkurl, registries)
	if errc != nil {
		crLog.Errorf("Create docker machine failed: %v", errc)
		crLog.Warnf("error message is : %s", errput)
		return server, errorCode, errc

	}

	ipAddress, err := command.GetMachinePublicIPAddress(hostname, storagePath)
	if err != nil {
		crLog.Errorf("GetMachinePublicIPAddress failed , err is %v", err)
		return
	}
	server.IpAddress = ipAddress

	privateIPAddress := ""
	nicname := "eth0"
	if mode == "new" {
		nicname = "eth0"
	} else if mode == "reuse" {
		nicname = node.PrivateNicName
	} else {
		crLog.Errorf("not supported create mode %s", mode)
		err = errors.New("not supported create mode for getting private ip!")
		return
	}
	privateIPAddress, err = command.GetMachinePrivateIPAddress(hostname, nicname, storagePath)
	if err != nil {
		crLog.Errorf("GetMachinePrivateIPAddress failed , err is %v", err)
		return
	}
	server.PrivateIpAddress = privateIPAddress

	// here, change host first to let docker-machine connect machine first.
	errorCode, err = p.ChangeHost(hostname, privateIPAddress, storagePath)
	if err != nil {
		crLog.Errorf("replace ssh key failed , err is %v", err)
		return
	}

	p.ReplaceKey(hostname, node.SshUser, storagePath, pubkeyPath, server.IpAddress)

	if err == nil {
		server.IsFullfilled = true
	}

	//start swarm agent in node
	if len(strings.TrimSpace(zkurl)) > 0 {

		imageRegistry := ""
		exist, registry := GetSystemRegistry(registries)
		if exist {
			imageRegistry = registry.Registry
		}

		err = p.bootUpSwarmAgentServer(server.Hostname, server.IpAddress, storagePath, zkurl, imageRegistry)
		if err != nil {
			crLog.Errorf("boot up swarm agent server error %v", err)
			return
		}
	}

	//config docker registries
	regErr := p.configDockerRegistry(server, storagePath, registries)
	if regErr != nil {
		crLog.Errorf("config docker registry error: %v", regErr)
	}

	crLog.Infof("docker-machine service is %v", server)

	return
}

func (p *DockerMachineService) buildProperties(properties map[string]string, engineOpts []entity.EngineOpt, mode string, node entity.Node, privateKeyPath string) (newPro map[string]string, err error) {
	newPro = make(map[string]string)
	for key, value := range properties {
		if key == "google-application-credentials" { //this att has been export as env, not a command para
			continue
		}
		if key == "google-use-internal-ip" && value == "false" {
			continue
		}
		newPro[key] = value
	}

	genericPort := node.Port
	if len(genericPort) <= 0 {
		genericPort = "22"
	}

	if mode == "reuse" {
		newPro["generic-ip-address"] = node.IP
		newPro["generic-ssh-user"] = node.SshUser
		newPro["generic-ssh-key"] = privateKeyPath
		newPro["generic-ssh-port"] = genericPort
		newPro["driver"] = "generic"
	} else if mode == "new" {
		for key, value := range newPro {
			if key == "amazonec2-access-key" || key == "amazonec2-secret-key" {
				keyBytes, errk := common.Base64Decode([]byte(value))
				if errk != nil {
					logrus.Errorf("fail to decode secretKey or accessKey: %v", value)
					return newPro, errk
				}
				decodeValue := string(keyBytes)
				newPro[key] = decodeValue
			}
		}
	} else {
		logrus.Errorf("not supported create mode %s", mode)
		return newPro, errors.New("not supported create mode for build properties!")
	}

	logrus.Debugf("properties content is %v", newPro)
	return
}

func (p *DockerMachineService) bootUpSwarmAgentServer(hostname, ipAddr, storagePath, zkurl string, imageRegistry string) (err error) {
	_, _, err = command.BootUpSwarmAgent(hostname, ipAddr, storagePath, zkurl, imageRegistry)
	if err != nil {
		logrus.Errorf("bootup swarm agent error %v", err)
		return
	}
	err = command.ChangeConfigFile(hostname, zkurl, storagePath, false, imageRegistry)
	if err != nil {
		logrus.Errorf("change swarm master config file error %v", err)
		return
	}

	return
}

func (p *DockerMachineService) configDockerRegistry(server entity.Server, storagePath string, registries []entity.DockerRegistry) (err error) {
	logrus.Infof("config host %s docker registries ", server.Hostname)

	hostname := server.Hostname
	err = p.MkdirAll(hostname, storagePath, DOCKER_LOGIN_SCRIPT_DIR, server.SshUser, server.SshUser, true)
	if err != nil {
		logrus.Errorf("cannot mkdir [%s] on [%s], error is [%v]", DOCKER_LOGIN_SCRIPT_DIR, hostname, err)
		return
	}

	// set private docker registry for both master and slave nodes
	err = p.CopyInstallExpectScript(hostname, storagePath)
	if err != nil {
		logrus.Errorf("copy install-expect.sh to %s:%s error is %v", hostname, storagePath, err)
		return
	}

	copyErr := p.CopyDockerLoginScript(hostname, storagePath)
	if copyErr != nil {
		logrus.Errorf("copy docker login script to %s:%s error is %v", hostname, storagePath, copyErr)
		return
	}

	logrus.Infoln("docker registries:%v", registries)
	for _, registry := range registries {
		if registry.Secure {
			//copy ca file
			err := p.WriteDockerRegistryCaFile(hostname, storagePath, registry)
			if err != nil {
				logrus.Errorf("write docker registry ca file to %s error is: %v", hostname, err)
				continue
			}
			logrus.Infof("write docker registry ca file to %s:%s success", hostname, storagePath)
		}
		//login to registry
		if len(registry.Username) != 0 && copyErr == nil {
			// dockerlogin.sh need `expect` installed
			// install-expect.sh will install `expect` if not exist (support 'yum' or 'apt-get' now)
			err = p.InstallPackageExpect(hostname, storagePath)
			if err != nil {
				logrus.Errorf("execute install-expect.sh error: %v", err)
				continue
			}

			err = p.RegistryLogin(hostname, storagePath, registry)
			if err != nil {
				logrus.Errorf("login to docker registry error is %v", err)
				continue
			}
			logrus.Infoln("login to registry success")

			// mesos need to add this config to pull images
			err := p.CompressDotDocker(hostname, storagePath)
			if err != nil {
				logrus.Errorf("compress docker auth config error: %v", err)
				continue
			}
		}
	}

	return
}

func (p *DockerMachineService) DeleteMachine(username, clustername, hostname string) (err error) {
	storagePath := p.ComposeStoragePath(username, clustername)
	_, _, err = command.DeleteMachine(hostname, storagePath)
	if err != nil {
		return err
	}
	return
}

func (p *DockerMachineService) DeleteAllMachines(username, clustername string) (err error) {
	storagePath := p.ComposeStoragePath(username, clustername)
	_, _, err = command.DeleteAllMachines(storagePath)
	if err != nil {
		return err
	}
	machineFolder := storagePath
	swarmFolder := command.DOCKERSWARM_STORAGEPATH_PREFIX + username + "/" + clustername + ""
	_, _, err = command.DeleteClusterFolder(machineFolder, swarmFolder)
	if err != nil {
		logrus.WithFields(logrus.Fields{"clustername": clustername}).Warnf("delete cluster machine folder and swarm fodler error %v", err)
	}

	return
}

func (p *DockerMachineService) ReplaceKey(hostname, sshUser, storagePath string, pubKeyPath []string, publicip string) (err error) {
	clustername := GetClusterName(storagePath)
	repLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	if len(pubKeyPath) <= 0 {
		repLog.Warnf("pubkey is not specified, will not inject customized pub key")
		return
	}

	privateKey := storagePath + "/machines" + "/" + hostname + "/id_rsa"

	for _, subpub := range pubKeyPath {
		commandStr := "eval `ssh-agent` && ssh-add " + privateKey + " && " + "/linker/copy-ssh-id.sh " + subpub + " " + privateKey + " " + sshUser + " " + publicip

		repLog.Infof("Executing add key and copy id command: %s", commandStr)
		_, _, err = commonCommand.ExecCommand(commandStr)
		if err != nil {
			repLog.Errorf("Call ssh-add failed , err is %v", err)
			continue
		}
	}

	return
}

func (p *DockerMachineService) ChangeHost(hostname, ipAddress, storagePath string) (errCode string, err error) {
	//prepare command
	clustername := GetClusterName(storagePath)

	commandStr := fmt.Sprintf(`
	if grep -xq .*%s /etc/hosts; then
		if grep -xq 127.0.1.1.* /etc/hosts; then
			sudo sed -i 's/^127.0.1.1.*/%s %s/g' /etc/hosts;
		else
			echo '%s %s' | sudo tee -a /etc/hosts;
		fi
	else
		echo '%s %s' | sudo tee -a /etc/hosts;
	fi`,
		hostname, ipAddress, hostname,
		ipAddress, hostname, ipAddress, hostname)

	_, _, err = command.ExecCommandOnMachine(hostname, commandStr, storagePath)
	if err != nil {
		errCode = DEPLOY_ERROR_CHANGE_HOST
		logrus.WithFields(logrus.Fields{"clustername": clustername}).Errorf("change hosts failed for server [%v], error is %v", ipAddress, err)
		return
	}
	return
}

func (p *DockerMachineService) ComposeStoragePath(username, clusername string) string {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clusername + ""
	return storagePath
}

func (p *DockerMachineService) WriteDockerRegistryCaFile(hostname, storagePath string, registry entity.DockerRegistry) (err error) {
	//prepare command
	// create folder for registry ca file
	// assume registry.Registry like "repo.linker.io:5000", no "https://" nor "http://"
	clustername := GetClusterName(storagePath)
	wirteLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	if strings.Contains(registry.Registry, "//") {
		wirteLog.Errorln("docker registry url contains protocol string //")
		return errors.New("invalid registry url")
	}
	path := "/etc/docker/certs.d/" + registry.Registry
	mkdirCommand := "sudo mkdir -p " + path
	_, _, err = command.ExecCommandOnMachine(hostname, mkdirCommand, storagePath)
	if err != nil {
		wirteLog.Errorf("Can't mkdir for host: [%s] for private docker registry, error is %v", hostname, err)
		return
	}

	// create CA file
	file := path + "/ca.crt"
	createfileCommand := "sudo touch " + file
	_, _, err = command.ExecCommandOnMachine(hostname, createfileCommand, storagePath)
	if err != nil {
		wirteLog.Errorf("Can't create ca file for host: [%s] for private docker registry, error is %v", hostname, err)
		return
	}

	// write ca text to file
	commandStr := fmt.Sprintf(`sudo echo '%s' | sudo tee -a %s`, registry.CAText, file)
	wirteLog.Debugf("Write ca text to file for host [%v], command is %v", hostname, commandStr)
	_, _, err = command.ExecCommandOnMachine(hostname, commandStr, storagePath)
	if err != nil {
		wirteLog.Errorf("Can't write ca text to file for host [%v], error is %v", hostname, err)
		return
	}
	return
}

func (p *DockerMachineService) RegistryLogin(hostname, storagePath string, registry entity.DockerRegistry) (err error) {
	// login to  authenticated private registry
	loginCommand := fmt.Sprintf("sudo expect %s %s %s %s", DOCKER_LOGIN_SCRIPT_PATH, registry.Registry, registry.Username, registry.Password)
	_, _, err = command.ExecCommandOnMachine(hostname, loginCommand, storagePath)
	if err != nil {
		logrus.Errorf("Login to registry[%s] failed on host: [%s] for private docker registry, error is %v", registry.Registry, hostname, err)
		return
	}
	return
}

func (p *DockerMachineService) RegistryLogout(hostname, storagePath string, registry entity.DockerRegistry) (err error) {
	// login to  authenticated private registry
	loginCommand := fmt.Sprintf("sudo expect %s %s %s %s", DOCKER_LOGOUT_SCRIPT_PATH, registry.Registry, registry.Username, registry.Password)
	_, _, err = command.ExecCommandOnMachine(hostname, loginCommand, storagePath)
	if err != nil {
		logrus.Errorf("Logout to registry[%s] failed on host: [%s] for private docker registry, error is %v", registry.Registry, hostname, err)
		return
	}
	return
}

func (p *DockerMachineService) CopyDockerLoginScript(hostname, storagePath string) (err error) {
	// scp shell script to slave.
	localFilePath := DOCKER_LOGIN_SCRIPT_PATH
	remotePath := DOCKER_LOGIN_SCRIPT_PATH
	_, _, err = command.ScpToMachine(hostname, localFilePath, remotePath, storagePath)
	if err != nil {
		logrus.Errorf("Can't copy dockerlogin.sh to host [%v], error is %v", hostname, err)
		return
	}
	return
}

func (p *DockerMachineService) CopyDockerLogoutScript(hostname, storagePath string) (err error) {
	// scp shell script to slave.
	localFilePath := DOCKER_LOGOUT_SCRIPT_PATH
	remotePath := DOCKER_LOGOUT_SCRIPT_PATH
	_, _, err = command.ScpToMachine(hostname, localFilePath, remotePath, storagePath)
	if err != nil {
		logrus.Errorf("Can't copy dockerlogin.sh to host [%v], error is %v", hostname, err)
		return
	}
	return
}

//mkdir if not exist
func (p *DockerMachineService) MkdirAll(hostname, storagePath, dir, owner, group string, sudo bool) (err error) {
	var mkdirCommand string
	if sudo {
		mkdirCommand = fmt.Sprintf("sudo mkdir -p %s && sudo chown -R %s:%s %s", dir, owner, group, dir)
	} else {
		mkdirCommand = fmt.Sprintf("mkdir -p %s && chown -R %s:%s", dir, owner, group)
	}
	_, _, err = command.ExecCommandOnMachine(hostname, mkdirCommand, storagePath)
	return
}

// Copy install-expect.sh
func (p *DockerMachineService) CopyInstallExpectScript(hostname, storagePath string) (err error) {
	// scp script to host
	localFilePath := INSTALL_EXPECT_SCRIPT_PATH
	remotePath := INSTALL_EXPECT_SCRIPT_PATH
	_, _, err = command.ScpToMachine(hostname, localFilePath, remotePath, storagePath)
	return
}

// Install package 'expect'
func (p *DockerMachineService) InstallPackageExpect(hostname, storagePath string) (err error) {
	installCommand := fmt.Sprintf("sudo bash %s", INSTALL_EXPECT_SCRIPT_PATH)
	_, _, err = command.ExecCommandOnMachine(hostname, installCommand, storagePath)
	return
}

// sudo tar -czf /var/lib/mesos/docker.tar.gz /root/.docker
func (p *DockerMachineService) CompressDotDocker(hostname, storagePath string) (err error) {
	// do not use `sudo cd /root` or `sudo su -` here
	var tarCommand bytes.Buffer
	tarCommand.WriteString("sudo mkdir -p /var/lib/mesos /tmp/.docker && ")
	tarCommand.WriteString("sudo cp /root/.docker/config.json /tmp/.docker/config.json && ")
	tarCommand.WriteString("cd /tmp && ")
	tarCommand.WriteString("sudo tar -czf /var/lib/mesos/docker.tar.gz .docker && ")
	tarCommand.WriteString("sudo rm -rf /tmp/.docker")
	_, _, err = command.ExecCommandOnMachine(hostname, tarCommand.String(), storagePath)
	return
}

func (p *DockerMachineService) ConfigAndBootMesosAgent(hostname, username, clustername string) (err error) {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + username + "/" + clustername
	// 1. untar mesos-agent tar file to /opt/
	var tarCommand bytes.Buffer
	tarCommand.WriteString("cd /tmp  && sudo rm -rf /tmp/mesosphere && ")
	tarCommand.WriteString("sudo tar -xf /tmp/customized-slave.tar  && ")
	tarCommand.WriteString("sudo mkdir -p /opt &&  ")
	tarCommand.WriteString("sudo rm -rf /opt/mesosphere && ")
	tarCommand.WriteString("sudo mv -f /tmp/mesosphere /opt/ && ")
	tarCommand.WriteString("sudo rm -f /tmp/customized-slave.tar && ")

        //remove the /tmp/clusterenv if exist
	tarCommand.WriteString("sudo rm -f /tmp/clusterenv ")
	_, _, err = command.ExecCommandOnMachine(hostname, tarCommand.String(), storagePath)
	if err != nil {
		logrus.Errorf("failed to untar mesos-agent file for host: %s, error: %v", hostname, err)
		return
	}

	// 2. copy .env file to /tmp/clusterenv
	localenvfile := command.DOCKERSWARM_STORAGEPATH_PREFIX + username + "/" + clustername + "/.env"
	_, _, err = command.ScpToMachine(hostname, localenvfile, "/tmp/clusterenv", storagePath)
	if err != nil {
		logrus.Errorf("failed to copy env file to host: %s, error: %v", hostname, err)
		return
	}

	// 3. config and start mesos-agent by systemd
	var configCommand bytes.Buffer
	configCommand.WriteString("more /tmp/clusterenv | sudo tee /opt/mesosphere/etc/mesos-slave-dynamic && ")
	configCommand.WriteString("sudo rm -f /etc/systemd/system/dcos-mesos-slave.service && ")
	configCommand.WriteString("sudo cp /opt/mesosphere/etc/dcos-mesos-slave.service /etc/systemd/system/ && ")
        //no need set this since mesos 1.2.1
	//configCommand.WriteString("sudo rm -f /tmp/dcos.conf && ")
	//configCommand.WriteString("echo \"/opt/mesosphere/lib\" >> /tmp/dcos.conf && ")
	//configCommand.WriteString("sudo chown root:root /tmp/dcos.conf && ")
	//configCommand.WriteString("sudo mv -f /tmp/dcos.conf /etc/ld.so.conf.d/  && ")
	//configCommand.WriteString("sudo ldconfig && ")
	configCommand.WriteString("sudo systemctl daemon-reload && sudo systemctl start dcos-mesos-slave && sudo systemctl enable dcos-mesos-slave ")

	_, _, err = command.ExecCommandOnMachine(hostname, configCommand.String(), storagePath)
	if err != nil {
		logrus.Errorf("failed to config and start mesos-agent on host: %s, error: %v", hostname, err)
		return
	}

	logrus.Infof("config and bootup mesos-agent process on host %s complete", hostname)

	return nil
}

func GetClusterName(storagePath string) (clustername string) {
	str := strings.Split(storagePath, "/")
	if len(str) >= 4 {
		clustername = str[4]
	}
	return
}

func (p *DockerMachineService) DeleteKey(hostname, sshUser, storagePath, pubKeyPath, publicip string) (err error) {
	clustername := GetClusterName(storagePath)
	repLog := logrus.WithFields(logrus.Fields{"clustername": clustername})

	if len(pubKeyPath) <= 0 {
		logrus.Errorf("keypath is not set")
		return
	}
	privateKey := storagePath + "/machines" + "/" + hostname + "/id_rsa"
	commandStr := "eval `ssh-agent` && ssh-add " + privateKey + " && " + "/linker/delete-ssh-id.sh " + pubKeyPath + " " + privateKey + " " + sshUser + " " + publicip

	repLog.Infof("Executing add key and copy id command: %s", commandStr)
	_, _, err = commonCommand.ExecCommand(commandStr)
	if err != nil {
		repLog.Errorf("Call ssh-add failed , err is %v", err)
		return
	}

	return
}
