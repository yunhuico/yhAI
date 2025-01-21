package common

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
)

func ExecCommand(input string) (output string, errput string, err error) {

	return ExecCommandinDir("", input)
}

func ExecCommandinDir(workDir string, input string) (output string, errput string, err error) {
	var retoutput string
	var reterrput string
	cmd := exec.Command("/bin/bash", "-c", input)

	//workDir is not ""
	if !strings.EqualFold(workDir, "") {
		cmd.Dir = workDir
	}

	logrus.Debugf("execute local command [%v]", cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Errorf("init stdout failed, error is %v", err)
		return "", "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.Errorf("init stderr failed, error is %v", err)
		return "", "", err
	}

	if err := cmd.Start(); err != nil {
		logrus.Errorf("start command failed, error is %v", err)
		return "", "", err
	}

	bytesErr, err := ioutil.ReadAll(stderr)
	if err != nil {
		logrus.Errorf("read stderr failed, error is %v", err)
		return "", "", err
	}

	if len(bytesErr) != 0 {
		reterrput = strings.Trim(string(bytesErr), "\n")
	}

	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		logrus.Errorf("read stdout failed, error is %v", err)
		return "", reterrput, err
	}

	if len(bytes) != 0 {
		retoutput = strings.Trim(string(bytes), "\n")
	}

	if err := cmd.Wait(); err != nil {
		logrus.Errorf("wait command failed, error is %v", err)
		logrus.Errorf("reterrput is %s", reterrput)
		return retoutput, reterrput, err
	}

	logrus.Debugf("retouput is %s", retoutput)
	logrus.Debugf("reterrput is %s", reterrput)
	return retoutput, reterrput, err
}

func SshExecCmdWithKey(ip string, port string, sshUser string, privateKeyPath string, command string) (output string, errput string, err error) {

	logrus.Debugf("start SshExecCmdWithKey... ip: %s, port: %s, sshUser: %s, privateKeyPath: %s, command: %s", ip, port, sshUser, privateKeyPath, command)
	//command: ssh -oConnectTimeout=10 -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -oBatchMode=yes -oPasswordAuthentication=no -p22 -i $privateKeyPath $(sshuser)@$(ip) $command
	input := "ssh -oConnectTimeout=10 -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -oBatchMode=yes -oPasswordAuthentication=no -p" + port +
		" -i " + privateKeyPath + " " + sshUser + "@" + ip + " " + command
	logrus.Infof("SshExecCmdWithKey, command: %s", input)

	return ExecCommand(input)
}
