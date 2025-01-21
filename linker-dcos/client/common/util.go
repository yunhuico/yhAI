package common

import (
	"errors"
	"strings"

	marathon "github.com/LinkerNetworks/go-marathon"
	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/utils"
)

var (
	UTIL *Util
	// marathonLog *logrus.Entry = logrus.New().WithFields(logrus.Fields{
	// 	"url": UTIL.MarathonClient.GetMarathonURL()})
)

type Util struct {
	Props          *properties.Properties
	MarathonClient marathon.Marathon
	LbClient       *LbClient
}

var Logger utils.LoggerSender
var BasicInfo entity.BasicInfo
var MgmtIps []string
var Url string

type LbClient struct {
	Host string
}

func (p *LbClient) GetUserMgmtEndpoint() (endpoint string, err error) {
	userMgmtPort := UTIL.Props.GetString("lb.usermgmt.port", "")
	if len(strings.TrimSpace(userMgmtPort)) == 0 {
		return "", errors.New("lb.deploy.port not configured!")
	}
	endpoint = p.Host + ":" + userMgmtPort
	return
}

func (p *LbClient) GetDeployEndpoint() (endpoint string, err error) {
	deployPort := UTIL.Props.GetString("lb.deploy.port", "")
	if len(strings.TrimSpace(deployPort)) == 0 {
		return "", errors.New("lb.deploy.port not configured!")
	}
	endpoint = p.Host + ":" + deployPort
	return
}

func (p *LbClient) GetMarathonEndpoint() (endpoint string, err error) {
	endpoint = UTIL.Props.GetString("marathon.endpoint", "")
	if len(strings.TrimSpace(endpoint)) == 0 {
		return "", errors.New("marathon.endpoint not configured!")
	}
	return
}

func (p *LbClient) GetMesosEndpoint() (endpoint string, err error) {
	endpoint = UTIL.Props.GetString("mesos.endpoint", "")
	if len(strings.TrimSpace(endpoint)) == 0 {
		return "", errors.New("mesos.endpoint not configured!")
	}
	return
}

func SendCreateLog(errs error, operation, queryType, comments string) (err error) {
	var status string
	if errs != nil {
		status = "fail"
	} else {
		status = "success"
	}
	logrus.Infof("basicinfo is %v", BasicInfo)
	logrus.Infof("url is %v", Url)
	logrus.Infof("mgmtips is %v", MgmtIps)
	newlog := Logger.CreateLogMess(status, operation, queryType, comments, BasicInfo)
	_, errS := Logger.PostCreateLog(Url, newlog)
	if errS != nil {
		logrus.Errorf("post create log err is %v", errS)
		return
	}
	return
}
