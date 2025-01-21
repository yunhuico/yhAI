package common

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
)

func ExecCommand(input string) (output string, errput string, err error) {
	var retoutput string
	var reterrput string
	cmd := exec.Command("/bin/bash", "-c", input)
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
		// stdoutbytes, _ := ioutil.ReadAll(stdout)
		// stderrbytes, _ := ioutil.ReadAll(stderr)
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
