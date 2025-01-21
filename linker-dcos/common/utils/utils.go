package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

var MgmtFilePath = "/linker/docker/MgmtIp.json"
var BasicInfoPath = "/linker/docker/BasicInfo.json"

type LoggerSender interface {
	CreateLogUrl(ips []string) string
	CreateEmailURL(ips []string) string
	PostCreateLog(string, entity.LogMessage) (bool, error)
	CreateFileInfo(string, interface{}) error
	CreateLogMess(string, string, string, string, entity.BasicInfo) entity.LogMessage
	// SendEmail calls clustermgmt API(arg1: API URL) to send email to user(arg2: userID)
	// with subject(arg3) and content(arg4)
	SendEmail(string, string, string, string) error
	GetBasicInfo() (entity.BasicInfo, error)
	GetMgmtIps() ([]string, error)
}

type Log struct {
	IsHttpsEnabled bool
	CaCertPath     string
}

func GetInstance(isHttpsEnabled bool, caCertPath string) (Logger LoggerSender) {
	log := &Log{IsHttpsEnabled: isHttpsEnabled, CaCertPath: caCertPath}
	return log
}

func (p *Log) CreateLogUrl(ips []string) (url string) {
	return p.getAvailIP(ips) + ":10002/v1/logs/create"
}

// CreateEmailURL checks over 'ips' and return first avaiable clustermgmt address
// with Email API path appended
func (p *Log) CreateEmailURL(ips []string) (url string) {
	if p.getAvailIP(ips) != "" {
		return p.getAvailIP(ips) + ":10002/v1/cluster/alertEmail"
	}
	return
}

func (p *Log) getAvailIP(ips []string) (availIP string) {
	// return ip in ips that is health
	if len(ips) == 0 {
		logrus.Infof("ips is null start to get ips again")
		info, _ := p.GetBasicInfo()
		ips = info.MgmtIp
		if len(ips) == 0 {
			return ""
		}
	}
	var resp *http.Response
	var err error
	var testUrl string
	for _, ip := range ips {
		testUrl = strings.Join([]string{ip, ":10002/v1/cluster/check"}, "")
		if p.IsHttpsEnabled {
			resp, err = httpclient.Https_get(testUrl, "", p.CaCertPath,
				httpclient.Header{"Content-Type", "application/json"})
		} else {
			resp, err = httpclient.Http_get(testUrl, "",
				httpclient.Header{"Content-Type", "application/json"})
		}
		if err != nil {
			logrus.Errorf("send http check error %v", err)
			return ""
		}
		defer resp.Body.Close()
		data, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			logrus.Errorf("http status code from dcos deployment failed %v", string(data))
			continue
		}
		success := isResponseSuccess(data)
		if !success {
			continue
		}
		return ip
	}
	return
}

func (p *Log) PostCreateLog(url string, logs entity.LogMessage) (isPost bool, err error) {
	if url == "" {
		logrus.Errorf("url cannot be null")
		Info, _ := p.GetBasicInfo()
		url = p.CreateLogUrl(Info.MgmtIp)
		if url == "" {
			errS := errors.New("has no mgmtip cannot post logs")
			return false, errS
		}
	}
	body, errM := json.Marshal(logs)
	if errM != nil {
		return false, errM
	}
	var resp *http.Response
	if p.IsHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), p.CaCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from token create failed")
	}

	success := isResponseSuccess(data)
	if !success {
		logrus.Errorf("post create log not success")
		return false, errors.New("post create log not success")
	}

	return true, nil
}

func (p *Log) CreateLogMess(status, operation, querytype, components string, basicInfo entity.BasicInfo) (logs entity.LogMessage) {
	if len(basicInfo.MgmtIp) == 0 {
		logrus.Infof("start to get basicinfo again")
		basicInfo, _ = p.GetBasicInfo()
	}
	logs.ClusterName = basicInfo.ClusterName
	logs.ClusterId = basicInfo.ClusterId
	logs.Username = basicInfo.UserName
	logs.TenantId = basicInfo.TenantId
	logs.UserId = basicInfo.UserId
	logs.OperateType = operation
	logs.QueryType = querytype
	logs.Status = status
	logs.Comments = components

	return
}

func (p *Log) CreateFileInfo(FilePath string, createInfo interface{}) (err error) {
	file, errC := os.Create(FilePath)
	if errC != nil {
		logrus.Errorf("create file err is %v", errC)
		return errC
	}
	enc := json.NewEncoder(file)
	errE := enc.Encode(&createInfo)
	if errE != nil {
		logrus.Errorf("encode file err is %v", errE)
		return errE
	}
	return
}

// SendEmail calls clustermgmt API(apiURL) to send email to user(userID)
// with subject and content
func (p *Log) SendEmail(apiURL, userID, subject, content string) (err error) {
	req := entity.SendHostAlertReq{
		UserId:  userID,
		Subject: subject,
		Content: content,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return
	}
	var resp *http.Response
	if p.IsHttpsEnabled {
		resp, err = httpclient.Https_post(apiURL, string(reqBody), p.CaCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(apiURL, string(reqBody),
			httpclient.Header{"Content-Type", "application/json"})
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.New("bad http status code")
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if !isResponseSuccess(respBody) {
		return errors.New("call clustermgmt to send email failed")
	}
	return nil
}

func (p *Log) GetMgmtIps() (mgmtIp []string, err error) {
	fp, _ := os.Open(MgmtFilePath)
	logrus.Infof("mgmtip file is %v", MgmtFilePath)
	dec := json.NewDecoder(fp)
	logrus.Infof("mgmtip fp is %v", fp)
	mgmt := entity.MgmtIps{}
	err = dec.Decode(&mgmt)
	if err != nil {
		logrus.Errorf("get mgmtip err is %v", err)
		return mgmtIp, err
	}
	mgmtIp = mgmt.MgmtIps
	return
}

func (p *Log) GetBasicInfo() (basicInfo entity.BasicInfo, err error) {
	fp, _ := os.Open(BasicInfoPath)
	dec := json.NewDecoder(fp)
	var basic entity.BasicInfo
	err = dec.Decode(&basic)
	if err != nil {
		logrus.Errorf("get basic info err is %v", err)
		return
	}
	basicInfo = basic
	return
}

func isResponseSuccess(data []byte) bool {
	var resp *response.Response
	resp = new(response.Response)
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return false
	}

	return resp.Success
}
