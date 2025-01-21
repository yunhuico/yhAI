package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
	//	dcosentity "linkernetworks.com/dcos-backend/common/entity"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/deployer/common"
	"linkernetworks.com/dcos-backend/common/utils"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"
)

var Log utils.Log

func CallbackPubkey(clustername, username string, ids []string, x_auth_token string) (ispost bool, err error) {
	clusterLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	clusterLog.Infoln("start to call back pubkey")
	clusterurl, err := common.GetClusterEndpoint()
	if err != nil {
		clusterLog.Errorf("get clusterMgmt endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{clusterurl, "/v1/cluster/pubkey/notify"}, "")
	clusterLog.Infoln("url is %v", url)
	pubkeunotify := entity.NotifyPubkey{}
	pubkeunotify.ClusterName = clustername
	pubkeunotify.UserName = username
	pubkeunotify.PubkeyIds = ids
	body, err := json.Marshal(pubkeunotify)
	if err != nil {
		return false, err
	}

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response
	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	}

	if err != nil {
		clusterLog.Errorf("send http post to create token error %v", err)
		return false, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		clusterLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from token create failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("delete node deployment module not success")
	}

	return true, nil
}

func CallbackCluster(clustername string, username string, isSuccess bool, operation string, logobjid string, comments string, x_auth_token string) (ispost bool, err error) {
	clusterLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	clusterLog.Infoln("start to call back cluster")
	clusterurl, err := common.GetClusterEndpoint()
	if err != nil {
		clusterLog.Errorf("get clusterMgmt endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{clusterurl, "/v1/cluster/notify"}, "")
	clusterLog.Infoln("url is %v", url)
	clusternotify := entity.NotifyCluster{}
	clusternotify.ClusterName = clustername
	clusternotify.UserName = username
	clusternotify.IsSuccess = isSuccess
	clusternotify.Operation = operation
	clusternotify.LogId = logobjid
	clusternotify.Comments = comments
	body, err := json.Marshal(clusternotify)
	if err != nil {
		return false, err
	}
	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response
	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	}

	if err != nil {
		clusterLog.Errorf("send http post to create token error %v", err)
		return false, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		clusterLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from token create failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("delete node deployment module not success")
	}

	return true, nil

}

func CallbackHost(clustername string, username string, isSuccess bool, operation string, ser []entity.Server, x_auth_token string) (ispost bool, err error) {
	hostLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	hostLog.Infoln("start to call back host!")
	clusterurl, err := common.GetClusterEndpoint()
	if err != nil {
		hostLog.Errorf("get clusterMgmt endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{clusterurl, "/v1/cluster/hosts/notify"}, "")
	hostnotify := entity.NotifyHost{}
	hostnotify.ClusterName = clustername
	hostnotify.UserName = username
	hostnotify.IsSuccess = isSuccess
	hostnotify.Operation = operation
	hostnotify.Servers = ser
	body, err := json.Marshal(hostnotify)
	if err != nil {
		return false, err
	}
	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response
	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", x_auth_token})
	}

	if err != nil {
		hostLog.Errorf("send http post to create token error %v", err)
		return false, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		hostLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from token create failed")
	}
	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("delete node deployment module not success")
	}

	return true, nil

}

// func CreateDefaultNetwork(request entity.Request, mgmtServers, slaveServers []entity.Server) (newNetwork *entity.ClusterNetwork, err error) {
// 	netLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
// 	netLog.Debugf("create overlay network with name is %s", request.Network.Name)
// 	//XINSHI TODO call the DCOS Client Service to create default overlay network
// 	clientEndpoint := fmt.Sprintf("%s:10004", mgmtServers[0].IpAddress)
// 	netLog.Debugf("client endpoint is %s", clientEndpoint)

// 	//call the client to create the network
// 	url := strings.Join([]string{clientEndpoint, "/v1/network"}, "")
// 	netLog.Debugln("get client network url=" + url)

// 	//prepare cluster Network
// 	clusterNetwork := entity.ClusterNetwork{}
// 	clusterNetwork.ClusterHostName = slaveServers[0].Hostname
// 	clusterNetwork.ClusterId = request.ClusterId
// 	clusterNetwork.ClusterName = request.ClusterName
// 	clusterNetwork.Network = request.Network
// 	body, err := json.Marshal(clusterNetwork)
// 	netLog.Debugln("cluster network body = " + string(body))

// 	var resp *http.Response

// 	resp, err = httpclient.Http_post(url, string(body),
// 		httpclient.Header{"Content-Type", "application/json"},
// 		httpclient.Header{"X-Auth-Token", request.XAuthToken})

// 	if err != nil {
// 		netLog.Errorf("send http post to dcos client for network create error %v", err)
// 		return newNetwork, err
// 	}
// 	defer resp.Body.Close()
// 	data, _ := ioutil.ReadAll(resp.Body)
// 	if resp.StatusCode >= 400 {
// 		netLog.Errorf("http status code from dcos client for network create failed %v", string(data))
// 		return newNetwork, errors.New("http status code from dcos client for network create failed")
// 	}

// 	newNetwork = new(entity.ClusterNetwork)
// 	err = getRetFromResponse(data, newNetwork)
// 	return newNetwork, err
// }

func CleanNetwork(clusterId, token, mgmtIp string) (err error) {
	logrus.Debugf("clean overlay networks for cluster: %s", clusterId)
	// call the DCOS Client Service to create default overlay network
	clientEndpoint := fmt.Sprintf("%s:10004", mgmtIp)
	logrus.Debugf("client endpoint is %s", clientEndpoint)

	//call the client to create the network
	url := strings.Join([]string{clientEndpoint, "/v1/network/?", "cluster_id=", clusterId}, "")
	logrus.Debugln("delete client network url=" + url)

	var resp *http.Response

	resp, err = httpclient.Http_delete(url, "",
		httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	if err != nil {
		logrus.Errorf("send http delete to dcos client network error %v", err)
		return err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http status code from dcos client network failed %v", string(data))
		return errors.New("http status code from dcos client network failed")
	}
	success := isResponseSuccess(data)
	if !success {
		return errors.New("delete  dcos client network module not success")
	}

	return nil
}

func GenerateHostName(clustername, username string) string {
	//get a random charactor
	chars := "abcdefghijklmnopqrstuvwxyz"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pos := r.Intn(len(chars))
	first := string(chars[pos])

	//current time
	second := strconv.FormatInt((int64)(time.Now().Nanosecond()), 10)

	return first + "-" + second + "-" + clustername + "-" + username
}

//zk://xxxx:2181,xxxx2181
func BuildDiscoveryZKList(serverList []entity.Server) string {
	if len(serverList) <= 0 {
		logrus.Warnln("server list is 0, can not return zookeeper list!")
		return ""
	}

	length := len(serverList)
	var ret bytes.Buffer
	ret.WriteString("zk://")
	for i := 0; i < length; i++ {
		ret.WriteString(serverList[i].PrivateIpAddress)
		ret.WriteString(":2181")

		if i < length-1 {
			ret.WriteString(",")
		}
	}

	logrus.Debugf("the discovery zookeeper list is %s", ret.String())
	return ret.String()
}

//xxxx:2888:3888,xxxx:2888:3888
func BuildZookeeperList(serverList []entity.Server) string {
	if len(serverList) <= 0 {
		logrus.Warnln("server list is 0, can not return zookeeper list!")
		return ""
	}

	length := len(serverList)
	var ret bytes.Buffer
	for i := 0; i < length; i++ {
		ret.WriteString(serverList[i].PrivateIpAddress)
		ret.WriteString(":2888:3888")

		if i < length-1 {
			ret.WriteString(",")
		}
	}

	logrus.Debugf("the zookeeper list is %s", ret.String())
	return ret.String()
}

func ParseNodecheck(output string) (ret []entity.NodesCheck) {
	arrays := strings.Split(output, "\n")
	for _, array := range arrays {
		// value := strings.TrimSpace(array)
		//   	valuearray:= strings.Split(array, ":")
		//    	if len(valuearray) != 2{
		//   		logrus.Warnf("node status error, node is %s", array)
		//   		continue
		//  	}

		nodecheck := entity.NodesCheck{}
		nodecheck.Nodename = array
		//   	nodecheck.Errormsg = strings.TrimSpace(valuearray[1])

		ret = append(ret, nodecheck)
	}

	logrus.Debugf("parsed node check array is %v", ret)
	return
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

	json.Unmarshal(jsonout, obj)

	return
}

//check if http response return success
func isResponseSuccess(data []byte) bool {
	var resp *response.Response
	resp = new(response.Response)
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return false
	}

	return resp.Success
}
