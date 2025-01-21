package services

import (
	"encoding/json"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/jsonq"
	"io/ioutil"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"strings"
	"sync"
)

const (
// CLUSTER_STATUS_TERMINATED = "TERMINATED"
// CLUSTER_STATUS_DEPLOYED   = "RUNNING"
// CLUSTER_STATUS_DEPLOYING  = "DEPLOYING"
// CLUSTER_STATUS_FAILED     = "FAILED"

// CLUSTER_ERROR_CREATE string = "E60000"
// CLUSTER_ERROR_UPDATE string = "E60001"
// CLUSTER_ERROR_DELETE string = "E60002"
// CLUSTER_ERROR_UNIQUE string = "E60003"
// CLUSTER_ERROR_QUERY  string = "E60004"

// CLUSTER_ERROR_PARSE_NUMBER       string = "E60010"
// CLUSTER_ERROR_NO_SUCH_MANY_NODES string = "E60011"
)

var (
	marathonService *MarathonService = nil
	onceMarathon    sync.Once
)

type MarathonService struct {
	serviceName string
}

func GetMarathonService() *MarathonService {
	onceMarathon.Do(func() {
		logrus.Debugf("Once called from MarathonService ......................................")
		marathonService = &MarathonService{"MarathonService"}
	})
	return marathonService
}

func (m *MarathonService) CreateGroup(payload []byte, marathonEndpoint string) (deploymentId string, err error) {
	url := strings.Join([]string{marathonEndpoint, "/v2/groups"}, "")
	logrus.Debugf("start to post group json %b to marathon %v", string(payload), marathonEndpoint)
	resp, err := httpclient.Http_post(url, string(payload),
		httpclient.Header{"Content-Type", "application/json"})
	if err != nil {
		logrus.Errorf("post group to marathon failed, error is %v", err)
		return
	}
	defer resp.Body.Close()

	// if response status is greater than 400, means marathon returns error
	// else parse body, findout deploymentId, and return
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("marathon returned error code is %v", resp.StatusCode)
		logrus.Errorf("detail is %v", string(data))
		err = errors.New(string(data))
		return
	}

	// Parse data: marathon json data
	jsondata := map[string]interface{}{}
	result := json.NewDecoder(strings.NewReader(string(data)))
	result.Decode(&jsondata)
	jq := jsonq.NewQuery(jsondata)
	deploymentId, err = jq.String("deploymentId")
	return
}

func (m *MarathonService) ScaleAppInstance(payload []byte, appid string, marathonEndpoint string) (deploymentId string, err error) {
	url := strings.Join([]string{marathonEndpoint, "/v2/apps/", appid}, "")
	logrus.Debugf("start to put app json %b to url %s", string(payload), url)
	resp, err := httpclient.Http_put(url, string(payload),
		httpclient.Header{"Content-Type", "application/json"})
	if err != nil {
		logrus.Errorf("scale app to marathon failed, error is %v", err)
		return
	}
	defer resp.Body.Close()

	// if response status is greater than 400, means marathon returns error
	// else parse body, findout deploymentId, and return
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("marathon returned error code is %v", resp.StatusCode)
		logrus.Errorf("detail is %v", string(data))
		err = errors.New(string(data))
		return
	}

	// Parse data: marathon json data
	jsondata := map[string]interface{}{}
	result := json.NewDecoder(strings.NewReader(string(data)))
	result.Decode(&jsondata)
	jq := jsonq.NewQuery(jsondata)
	deploymentId, err = jq.String("deploymentId")
	return
}

func (m *MarathonService) IsDeploymentDone(deploymentId, marathonEndpoint string) (isDone bool, err error) {
	isDone = true
	url := strings.Join([]string{marathonEndpoint, "/v2/deployments"}, "")
	logrus.Debugf("start to check deployment %v status from  marathon %v", deploymentId, marathonEndpoint)
	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})
	if err != nil {
		logrus.Errorf("get deployments from marathon failed, error is %v", err)
		return
	}
	defer resp.Body.Close()

	// if response status is greater than 400, means marathon returns error
	// else returned body, findout deploymentId, and return
	data, _ := ioutil.ReadAll(resp.Body)
	datastr := string(data)
	if resp.StatusCode >= 400 {
		logrus.Errorf("marathon returned error code is %v", resp.StatusCode)
		logrus.Errorf("detail is %v", datastr)
		err = errors.New(datastr)
		return
	}

	// Parse data: marathon json data
	jsondata := []map[string]interface{}{}
	result := json.NewDecoder(strings.NewReader(datastr))
	result.Decode(&jsondata)
	for i := 0; i < len(jsondata); i++ {
		jq := jsonq.NewQuery(jsondata[i])
		inDeployingId, err := jq.String("id")
		logrus.Debugf("found inDeployingId %v", inDeployingId)
		if strings.EqualFold(inDeployingId, deploymentId) {
			isDone = false
			return isDone, err
		}
	}
	return
}

func (m *MarathonService) IsServiceAvaiable(marathonEndpoint string) (isAvaiable bool) {
	isAvaiable = false
	url := strings.Join([]string{marathonEndpoint, "/v2/deployments"}, "")
	logrus.Debugf("start to check status of  marathon %s", marathonEndpoint)
	resp, err := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})
	if err != nil {
		logrus.Errorf("get deployments from marathon failed, error is %v", err)
		return
	}

	defer resp.Body.Close()

	// if response status is greater than 400, means marathon returns error
	// else returned body, findout deploymentId, and return
	data, _ := ioutil.ReadAll(resp.Body)
	datastr := string(data)
	if resp.StatusCode >= 400 {
		logrus.Errorf("marathon returned error code is %v", resp.StatusCode)
		logrus.Errorf("detail is %v", datastr)
		err = errors.New(datastr)
		return
	}

	isAvaiable = true

	return

}
