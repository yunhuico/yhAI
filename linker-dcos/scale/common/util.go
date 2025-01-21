package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/jsonq"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

const (
	base64Table = "ABCDEFGHIJKLMNOPQRSTpqrstuvwxyz0123456789+/UVWXYZabcdefghijklmno"

	OPERATION_ADD    = "add"
	OPERATION_REMOVE = "remove"

	CLUSTER_STATUS_RUNNING      = "RUNNING"
	CLUSTER_STATUS_MODIFYING    = "MODIFYING"
	CLUSTER_STATUS_UNTERMINATED = "unterminated"

	LINKER_CONNECTOR_APPID = "linker-connector-docker"

	USER_MGMT_PORT    = "10001"
	CLUSTER_MGMT_PORT = "10002"

	RetryTime = 120
)

var coder = base64.NewEncoding(base64Table)

func Base64Encode(src []byte) []byte {
	return []byte(coder.EncodeToString(src))
}

func getUserMgmtEndpoint(clusterlb string) string {
	return clusterlb + ":" + USER_MGMT_PORT
}

func getClusterMgmtEndpoint(clusterlb string) string {
	return clusterlb + ":" + CLUSTER_MGMT_PORT
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Tenantname string `json:"tenantname"`
}

type ScaleApp struct {
	Instances int `json:"instances"`
}

type TokenId struct {
	Id string `json:"id"`
}

type TerminateHostsRequestBody struct {
	HostIds []string `json:"host_ids"`
}

func GenerateToken(username, password, clusterlb string) (tokenId string, err error) {
	userUrl := getUserMgmtEndpoint(clusterlb)

	url := strings.Join([]string{userUrl, "/v1/token"}, "")
	logrus.Debugln("token validation url=" + url)

	login := LoginRequest{}
	login.Username = username
	login.Password = password
	login.Tenantname = username
	body, errm := json.Marshal(login)
	if errm != nil {
		logrus.Errorf("marshal login object error %v", errm)
		return tokenId, errm
	}

	var resp *http.Response
	resp, err = httpclient.Http_post(url, string(body),
		httpclient.Header{"Content-Type", "application/json"})

	if err != nil {
		logrus.Errorf("http generate token error %v", err)
		return tokenId, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("token generate failed %v", string(data))
		return tokenId, errors.New("http generate token error")
	}

	token := TokenId{}
	err = getRetFromResponse(data, &token)
	if err != nil {
		return
	}

	tokenId = token.Id
	return
}

func GetClusterByName(tokenId, clusterlb, clustername, clusterstatus string) (cluster entity.Cluster, err error) {
	clusterUrl := getClusterMgmtEndpoint(clusterlb)

	url := strings.Join([]string{clusterUrl, "/v1/cluster/?name=", clustername, "&status=", clusterstatus}, "")
	logrus.Debugln("get cluster by name url=" + url)

	var resp *http.Response

	resp, err = httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"},
		httpclient.Header{"X-Auth-Token", tokenId})

	if err != nil {
		logrus.Errorf("http get cluster error %v", err)
		return cluster, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get cluster by clustername failed %v", string(data))
		return cluster, errors.New("get cluster by clustername failed")
	}

	clusters := []entity.Cluster{}
	err = getRetFromResponse(data, &clusters)

	if len(clusters) <= 0 {
		logrus.Errorf("the specified  Running cluster does not exist!")
		return cluster, errors.New("the specified cluster does not exist!")
	}
	if len(clusters) > 1 {
		logrus.Errorf("duplicated Running cluster for cluster: %s", clustername)
		return cluster, errors.New("duplicated Running cluster for specified clustername!")
	}

	return clusters[0], nil

}

func GetHostsByClusterId(tokenId, clusterlb, clusterId string) (hostinfos []entity.HostInfo, err error) {
	clusterUrl := getClusterMgmtEndpoint(clusterlb)

	url := strings.Join([]string{clusterUrl, "/v1/cluster/", clusterId, "/hosts/?", "status=unterminated"}, "")
	logrus.Debugln("get hosts by cluster url=" + url)

	var resp *http.Response

	resp, err = httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"},
		httpclient.Header{"X-Auth-Token", tokenId})

	if err != nil {
		logrus.Errorf("http get hosts by cluster error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get hosts by cluster failed %v", string(data))
		return nil, errors.New("get hosts by cluster failed")
	}

	hostinfos = []entity.HostInfo{}
	err = getRetFromResponse(data, &hostinfos)

	return

}

func SendScaleOut(clusterlb, clusterId, tokenId string, addrequest entity.AddRequest) (err error) {
	clusterUrl := getClusterMgmtEndpoint(clusterlb)

	url := strings.Join([]string{clusterUrl, "/v1/cluster/", clusterId, "/hosts"}, "")
	logrus.Debugln("cluster scale out url=" + url)

	body, errm := json.Marshal(addrequest)
	if errm != nil {
		logrus.Errorf("marshal add request error %v", errm)
		return errm
	}

	var resp *http.Response
	resp, err = httpclient.Http_post(url, string(body),
		httpclient.Header{"Content-Type", "application/json"},
		httpclient.Header{"X-Auth-Token", tokenId})

	if err != nil {
		logrus.Errorf("scaleout operation error %v", err)
		return err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("scale out operation failed %v", string(data))
		return errors.New("http scale out error")
	}

	return

}

func SendScaleIn(clusterlb, clusterId, tokenId string, removeRequest TerminateHostsRequestBody) (err error) {
	clusterUrl := getClusterMgmtEndpoint(clusterlb)

	url := strings.Join([]string{clusterUrl, "/v1/cluster/", clusterId, "/hosts"}, "")
	logrus.Debugln("cluster scale in url=" + url)

	body, errm := json.Marshal(removeRequest)
	if errm != nil {
		logrus.Errorf("marshal remove request error %v", errm)
		return errm
	}

	var resp *http.Response
	resp, err = httpclient.Http_delete(url, string(body),
		httpclient.Header{"Content-Type", "application/json"},
		httpclient.Header{"X-Auth-Token", tokenId})

	if err != nil {
		logrus.Errorf("scalein operation error %v", err)
		return err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("scale in operation failed %v", string(data))
		return errors.New("http scale in error")
	}

	return
}

func WaitingAndScaleApp(clustername, tokenId, clusterlb, operation string, number int) (err error) {
	//waiting VM creation complete
	flag := false
	var cluster entity.Cluster
	for i := 0; i < RetryTime; i++ {
		cluster, err = GetClusterByName(tokenId, clusterlb, clustername, CLUSTER_STATUS_UNTERMINATED)
		if err != nil {
			logrus.Warnf("get cluster by name error %v", err)
			continue
		}
		if cluster.Status == CLUSTER_STATUS_RUNNING {
			flag = true
			break
		} else {
			time.Sleep(30000 * time.Millisecond)
		}
	}

	if !flag {
		logrus.Warnf("cluster status is not running in 30 minutes, will not scale out linker connector")
		return
	}

	//scale app
	instance, err := getAppInstance()
	if err != nil {
		return err
	}

	if operation == OPERATION_ADD {
		instance += number
	} else if operation == OPERATION_REMOVE {
		instance -= number
	} else {
		logrus.Errorf("not supported operation type %s", operation)
		return nil
	}

	scaleapp := ScaleApp{}
	scaleapp.Instances = instance

	scaleAppInstance(scaleapp)

	return

}

func getAppInstance() (int, error) {
	url := strings.Join([]string{"master.mesos/marathon", "/v2/apps/", LINKER_CONNECTOR_APPID}, "")
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

func scaleAppInstance(scaleapp ScaleApp) {
	url := strings.Join([]string{"master.mesos/marathon", "/v2/apps/", LINKER_CONNECTOR_APPID, "?force=true"}, "")

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

func getRetFromResponse(data []byte, obj interface{}) (err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	jsonout, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonout, obj)
	if err != nil {
		return err
	}

	return
}