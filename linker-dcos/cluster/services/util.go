package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/cluster/common"
	//	dcosentity "linkernetworks.com/linker_common_lib/entity"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"
)

func getErrorFromResponse(data []byte) (errorCode string, err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return COMMON_ERROR_INTERNAL, err
	}

	errorCode = resp.Error.Code
	err = errors.New(resp.Error.ErrorMsg)
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

func TokenValidation(tokenId string) (errorCode string, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return COMMON_ERROR_INTERNAL, err
	}
	url := strings.Join([]string{userUrl, "/v1/token/?", "token=", tokenId}, "")
	isHttpsEnabled := common.UTIL.Props.GetBool("http.usermgmt.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.usermgmt.https.crt", "")

	logrus.Debugln("token validation url=" + url)

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"})
	}

	if err != nil {
		logrus.Errorf("http get token validate error %v", err)
		return COMMON_ERROR_INTERNAL, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("token validation failed %v", string(data))
		errorCode, err = getErrorFromResponse(data)
		return
	}

	return "", nil
}
func GetTokenById(token string) (currentToken *entity.Token, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return nil, err
	}
	url := strings.Join([]string{userUrl, "/v1/token/", token}, "")
	logrus.Debugln("get token url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.usermgmt.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.usermgmt.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	}
	if err != nil {
		logrus.Errorf("http get token error %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get token by id failed %v", string(data))
		return nil, errors.New("get token by id failed")
	}

	currentToken = new(entity.Token)
	err = getRetFromResponse(data, currentToken)
	return
}

func GetUserByIdForAlert(userId string, token string) (user *entity.User, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return nil, err
	}
	url := strings.Join([]string{userUrl, "/v1/user/forcluster/", userId}, "")
	logrus.Debugln("get user url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.usermgmt.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.usermgmt.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", token})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", token})
	}

	if err != nil {
		logrus.Errorf("http get user error %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get user by id failed %v", string(data))
		return nil, errors.New("get user by id failed")
	}

	user = new(entity.User)
	err = getRetFromResponse(data, user)
	return
}

func GetUserById(userId string, token string) (user *entity.User, err error) {
	userUrl, err := common.UTIL.LbClient.GetUserMgmtEndpoint()
	if err != nil {
		logrus.Errorf("get userMgmt endpoint err is %v", err)
		return nil, err
	}
	url := strings.Join([]string{userUrl, "/v1/user/", userId}, "")
	logrus.Debugln("get user url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.usermgmt.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.usermgmt.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", token})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", token})
	}

	if err != nil {
		logrus.Errorf("http get user error %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get user by id failed %v", string(data))
		return nil, errors.New("get user by id failed")
	}

	user = new(entity.User)
	err = getRetFromResponse(data, user)
	return
}

func GetHostNameByTag(hosts []entity.Host, tag string) []string {
	allhostnames := []string{}
	for _, host := range hosts {
		allhostnames = append(allhostnames, host.HostName)
	}

	if len(tag) > 0 {
		lb := getlb(hosts)
		if len(lb) <= 0 {
			logrus.Errorf("no shared node for specfic cluster!")
			return allhostnames
		}

		hostnames, err := sendGetNodeNameByTagRequest(lb, tag)
		if err != nil {
			logrus.Errorf("query node by tag error %v", err)
			return allhostnames
		}
		return hostnames

	} else {
		return allhostnames
	}

}

func GetHostAdditionalInfos(hosts []entity.Host, hostnames []string, token string) (hostinfos []entity.HostInfo) {
	if len(hosts) <= 0 {
		logrus.Infof("hosts lenght is 0!")
		return hostinfos
	}

	clusterid := hosts[0].ClusterId
	clustername := hosts[0].ClusterName
	getLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	cluster, _, errq := GetClusterService().QueryById(clusterid, token)
	if errq != nil {
		getLog.Errorf("query cluster err is %v", errq)
		return
	}

	getLog.Infof("get node additional information for cluster %s ", cluster.Name)
	var pubkeyId string
	if len(cluster.PubKeyId) != 0 {
		pubkeyId = cluster.PubKeyId[0]
	}

	var pubkeyName string
	if len(pubkeyId) == 0 {
		getLog.Warnf("there is no pubkeyid for clusterid %s", clusterid)
	} else {
		pubkey, _, errp := GetPubKeyService().QueryById(pubkeyId, token)
		if errp != nil {
			getLog.Errorf("query pubkey err is %v", errp)
		}
		pubkeyName = pubkey.Name
	}

	hostinfos = []entity.HostInfo{}
	for _, onehost := range hosts {
		if inArray(hostnames, onehost.HostName) {
			hostinfo := convertToHostInfo(onehost, pubkeyName)
			hostinfos = append(hostinfos, hostinfo)
		}

	}

	//get node check info (check nodes' existence for running and failed cluster)
	if cluster.Status == CLUSTER_STATUS_FAILED || cluster.Status == CLUSTER_STATUS_RUNNING {
		getLog.Infof("get node health check information for cluster %s ", cluster.Name)
		nodechecks, errs := generatNodecheck(cluster.UserId, token, cluster.Name)
		if errs != nil {
			getLog.Errorf("get cluster node's health check error %v", errs)
		} else {
			getLog.Debugf("node health check information is %v", nodechecks)
			isMaster := false
			var onlinesharednum int
			var onlinemasternum int
			for i := 0; i < len(hostinfos); i++ {
				onehost := &hostinfos[i]
				_, exist, status, dockerVersion := getNodecheckInList(onehost.HostName, nodechecks)
				if !exist {
					onehost.Status = HOST_STATUS_OFFLINE
					getLog.Debugf("node %s has been remove in backgroud", onehost.HostName)
					hostinfos[i].Status = HOST_STATUS_OFFLINE
					GetHostService().UpdateStatusById(onehost.HostId, HOST_STATUS_OFFLINE, token)
					if onehost.IsMasterNode {
						isMaster = true
					}
				} else {
					hasDocker := strings.HasPrefix(dockerVersion, "v")
					var state string
					if status == "Running" && hasDocker {
						state = HOST_STATUS_RUNNING
					} else {
						state = HOST_STATUS_OFFLINE
					}
					if onehost.Status == HOST_STATUS_RUNNING {
						if state == HOST_STATUS_OFFLINE {
							hostinfos[i].Status = HOST_STATUS_OFFLINE
							getLog.Infof("host status is running but now offline")
							GetHostService().UpdateStatusById(onehost.HostId, state, token)
						}
					} else if onehost.Status == HOST_STATUS_OFFLINE {
						if state == HOST_STATUS_RUNNING {
							hostinfos[i].Status = HOST_STATUS_RUNNING
							getLog.Infof("host status is offline but now running")
							GetHostService().UpdateStatusById(onehost.HostId, state, token)
						}
					}
					if onehost.IsMasterNode {
						if state == HOST_STATUS_RUNNING {
							onlinemasternum = onlinemasternum + 1
						}
					}
					if onehost.IsSharedNode {
						if state == HOST_STATUS_RUNNING {
							onlinesharednum = onlinesharednum + 1
						}
					}
				}
			}
			getLog.Infof("onlinesharednum is %v", onlinesharednum)
			var comparenum int
			var comparemasternum int
			if cluster.CreateCategory == "ha" {
				comparenum = 2
				comparemasternum = 3
			} else {
				comparenum = 1
				comparemasternum = 1
			}
			if onlinesharednum < comparenum || onlinemasternum < comparemasternum {
				isMaster = true
			}
			getLog.Infof("onlinemasternum is %v", onlinemasternum)
			if isMaster {
				getLog.Infof("cluster %s 's master or shared node has been removed, change it's status to failed", cluster.Name)
				GetClusterService().UpdateStatusById(clusterid, CLUSTER_STATUS_FAILED, token)
			} else {
				if cluster.Status == CLUSTER_STATUS_FAILED {
					getLog.Infof("start to change cluster to running")
					total, _, _, err := GetHostService().QueryHosts(clusterid, 0, 0, HOST_STATUS_RUNNING, token)
					if err != nil {
						getLog.Errorf("query host err is %v", err)
					} else {
						if cluster.CreateCategory == "ha" {
							if (total >= 5) {
								GetClusterService().UpdateStatusById(clusterid, CLUSTER_STATUS_RUNNING, token)
							}
						} else {
							if total >= 2 {
								GetClusterService().UpdateStatusById(clusterid, CLUSTER_STATUS_RUNNING, token)
							}
						}
					}					
				} else if cluster.Status == CLUSTER_STATUS_RUNNING {
					getLog.Infof("cluster is running and is running now, is normal")
				}
			}

		}
	}

	var routerip string
	for _, host := range hosts {
		if host.IsMasterNode {
			routerip = host.IP
			break
		}
	}
	if len(routerip) <= 0 {
		getLog.Warnf("cluster has no adminrouter for clusterId %s", clusterid)
		return
	}
	url := routerip + "/mesos/state-summary"
	getLog.Infof("the url of get hostinfo is %s", url)

	resp, errg := httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})
	if errg != nil {
		getLog.Warnf("http get state-summary error %v", errg)
		return hostinfos
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		getLog.Warnf("get state-summary failed %v", string(data))
		return hostinfos
	}

	state := new(entity.StateSummary)
	err := json.Unmarshal(data, state)
	if err != nil {
		getLog.Errorf("get result error %v", err)
		return hostinfos
	}

	for i := 0; i < len(hostinfos); i++ {
		onehost := &hostinfos[i]
		setHostInfo(onehost, state.Slaves)
	}

	getLog.Debugf("the result is %v", hostinfos)

	return hostinfos

}

func inArray(hostnames []string, onename string) bool {
	if len(hostnames) <= 0 {
		return false
	}

	for _, name := range hostnames {
		if strings.EqualFold(name, onename) {
			return true
		}
	}

	return false
}

func convertToHostInfo(host entity.Host, pubkeyname string) (hostinfo entity.HostInfo) {
	hostinfo = entity.HostInfo{
		HostId:       host.ObjectId.Hex(),
		HostName:     host.HostName,
		ClusterId:    host.ClusterId,
		ClusterName:  host.ClusterName,
		Status:       host.Status,
		IP:           host.IP,
		PrivateIp:    host.PrivateIp,
		IsMasterNode: host.IsMasterNode,
		IsSlaveNode:  host.IsSlaveNode,
		IsSharedNode: host.IsSharedNode,
		IsFullfilled: host.IsFullfilled,
		IsClientNode: host.IsClientNode,
		UserId:       host.UserId,
		TenantId:     host.TenantId,
		PubKeyName:   pubkeyname,
		Type:         host.Type,
		TimeCreate:   host.TimeCreate,
		TimeUpdate:   host.TimeUpdate}

	return hostinfo
}

func setHostInfo(hostinfo *entity.HostInfo, slaves []entity.Slave) {
	slave, err := getSlave(hostinfo.PrivateIp, slaves)
	if err != nil {
		return
	}

	hostinfo.Task = slave.TaskRunning

	hostinfo.CPU = fmt.Sprint(slave.UsedResources.CPUs) + "/" + fmt.Sprint(slave.Resources.CPUs)
	hostinfo.Memory = fmt.Sprint(slave.UsedResources.Mem) + "/" + fmt.Sprint(slave.Resources.Mem)
	hostinfo.GPU = fmt.Sprint(slave.UsedResources.GPUs) + "/" + fmt.Sprint(slave.Resources.GPUs)

	labels := []string{}
	attrs := slave.Attributes
	attrMap := attrs.(map[string]interface{})
	for key, value := range attrMap {
		label := key + "=" + fmt.Sprintf("%v", value)
		labels = append(labels, label)
	}

	hostinfo.Tag = labels

	return
}

func getSlave(ip string, slaves []entity.Slave) (ret entity.Slave, err error) {
	for _, slave := range slaves {
		if ip == slave.HostName {
			return slave, nil
		}
	}

	logrus.Infof("does not find slave with ip %s", ip)
	return ret, errors.New("does not find slave by ip")
}

func getNodecheckInList(hostname string, arrays []entity.NodesCheck) (ret entity.NodesCheck, exist bool, status string, dockerVersion string) {
	if len(hostname) <= 0 {
		return ret, false, status, dockerVersion
	}

	for _, item := range arrays {
		str := strings.Split(item.Nodename, ":")
		var state string
		if len(str) == 3 {
			state = str[1]
			dockerVersion = str[2]
		}
		if strings.EqualFold(hostname, str[0]) {
			status = state
			return item, true, status, dockerVersion
		}
	}

	return ret, false, status, dockerVersion
}

func generatNodecheck(userId, token, clustername string) (ret []entity.NodesCheck, err error) {
	user, erru := GetUserById(userId, token)
	if erru != nil {
		logrus.Errorf("get user by id error %v", erru)
		return
	}

	return sendGetNodeCheckRequest(user.Username, clustername)
}

func getlb(hosts []entity.Host) string {
	for _, host := range hosts {
		if host.IsSharedNode {
			return host.IP
		}
	}

	return ""
}

func sendGetNodeNameByTagRequest(lb, tag string) (ret []string, err error) {
	cmiUrl := common.GetCMIEndpoint(lb)
	url := strings.Join([]string{cmiUrl, "/v1/cmi/nodes?server_tag=", tag}, "")

	var resp *http.Response
	resp, err = httpclient.Http_get(url, "",
		httpclient.Header{"Content-Type", "application/json"})

	if err != nil {
		logrus.Errorf("send http get to cmi get node by tag error %v", err)
		return ret, err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http status code from get node by tag failed %v", string(data))
		return ret, errors.New("http status code from getting node by tag failed")
	}

	ret = []string{}
	err = getRetFromResponse(data, &ret)
	return

}

func sendGetNodeCheckRequest(username, clustername string) (ret []entity.NodesCheck, err error) {
	seLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		seLog.Errorf("get deploy endpoint err is %v", err)
		return ret, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/nodes/healthcheck?username=", username, "&clustername=", clustername}, "")
	seLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		seLog.Errorf("send http get to dcos get node check error %v", err)
		return ret, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		seLog.Errorf("http status code from get node check failed %v", string(data))
		return ret, errors.New("http status code from node check failed")
	}

	ret = []entity.NodesCheck{}
	err = getRetFromResponse(data, &ret)
	return

}

//send request to dcos deployment module to create cluster
func SendCreateClusterRequest(request entity.Request) (ispost bool, err error) {
	scLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	scLog.Infoln("Call deployment to create cluster")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		scLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy"}, "")
	scLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		scLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		scLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		scLog.Errorf("deploy is not success")
		return false, errors.New("deploy is not success!")
	}

	return true, nil
}

//send request to dcos deployment module to delete cluster
func SendDeleteClusterRequest(request *entity.DeleteAllRequest) (deleted bool, err error) {
	sdLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	sdLog.Infoln("Call deployment to delete cluster")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		sdLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy"}, "")
	sdLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_delete(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_delete(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		sdLog.Errorf("send http delete to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		sdLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}
	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("deployment module not success")
	}
	return true, nil
}

//send request to dcos deployment module to add nodes
func SendAddNodesRequest(request *entity.AddNodeRequest) (ispost bool, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("Call deployment to add nodes")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/nodes"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("add node deployment module not success")
	}

	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

//send request to dcos deployment module to delete nodes
func SendDeleteNodesRequest(request *entity.DeleteRequest) (ispost bool, err error) {
	sdLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	sdLog.Infoln("Call deployment to delete nodes")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		sdLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/nodes"}, "")
	sdLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_delete(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_delete(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		sdLog.Errorf("send http delete to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		sdLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}
	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("delete node deployment module not success")
	}
	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

func DeleteOvsNetwork(url, cluster_id, host_name string) (ispost bool, err error) {
	port, err := common.UTIL.LbClient.GetClientPort()
	if err != nil {
		logrus.Errorf("get client port err is %v", err)
		return false, err
	}
	Url := strings.Join([]string{url, ":", port, "/v1/network/ovs?cluster_id=", cluster_id, "&&host_name=", host_name}, "")
	logrus.Infof("delete network url is %v", Url)
	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_delete(Url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_delete(Url, "",
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		logrus.Errorf("send http delete to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}
	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("deployment module not success")
	}
	return true, nil
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

func getCountFromResponse(data []byte) (count int, err error) {
	var resp *response.QueryStruct
	resp = new(response.QueryStruct)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return
	}

	jsonout, err := json.Marshal(resp.Count)
	if err != nil {
		return
	}

	json.Unmarshal(jsonout, &count)

	return
}

func HashString(password string) string {
	encry := sha256.Sum256([]byte(password))
	return hex.EncodeToString(encry[:])
}

//check if ip is a valid IPv4 address
func IsIpAddressValid(ip string) bool {
	reg := regexp.MustCompile(`\b((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4}\b`)
	return reg.MatchString(ip)
}

//check cluster name with regex
//letters (upper or lowercase)
//numbers (0-9)
//underscore (_)
//dash (-)
//length 1-255
//no spaces! or other characters
func IsClusterNameValid(name string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9-]{1,15}$`)
	return reg.MatchString(name)
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetWaitTime(execTime time.Time) int64 {
	one_day := 24 * 60 * 60
	currenTime := time.Now()

	execInt := execTime.Unix()
	currentInt := currenTime.Unix()

	var waitTime int64
	if currentInt <= execInt {
		waitTime = execInt - currentInt
	} else {
		waitTime = (execInt + int64(one_day)) % currentInt
	}

	return waitTime
}

func SendAddPubkeysRequest(request *entity.AddPubkeysRequest) (ispost bool, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("Call deployment to add pubkey")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/addpubkeys"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("add pubkey deployment module not success")
	}

	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

func SendDelePubkeysRequest(request *entity.DeletePubkeysRequest) (ispost bool, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("Call deployment to delete pubkey")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/deletepubkeys"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_delete(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_delete(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("add pubkey deployment module not success")
	}

	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

func SendAddRegistryRequest(request *entity.AddRegistryRequest) (ispost bool, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("Call deployment to add registry")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/addregistry"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("add registry deployment module not success")
	}

	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

func SendDeleteRegistryRequest(request *entity.DeleteRegistryRequest) (ispost bool, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("Call deployment to delete registry")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return false, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/deleteregistry"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_delete(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_delete(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return false, errors.New("http status code from dcos deployment failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("add registry deployment module not success")
	}

	//	servers = new([]dcosentity.Server)
	//	err = getRetFromResponse(data, servers)
	return true, nil
}

func SendComponentCheck(request entity.Components) (componentinfo entity.ComponentsInfo, err error) {
	saLog := logrus.WithFields(logrus.Fields{"clustername": request.ClusterName})
	saLog.Infoln("start to call deployer get componentinfo")
	body, err := json.Marshal(request)
	deployUrl, err := common.UTIL.LbClient.GetDeployEndpoint()
	if err != nil {
		saLog.Errorf("get deploy endpoint err is %v", err)
		return componentinfo, err
	}
	url := strings.Join([]string{deployUrl, "/v1/deploy/components/healthcheck"}, "")
	saLog.Debugln("get deploy url=" + url)

	isHttpsEnabled := common.UTIL.Props.GetBool("http.dcosdeploy.https.enabled", false)
	caCertPath := common.UTIL.Props.GetString("http.dcosdeploy.https.crt", "")

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"})
	}
	if err != nil {
		saLog.Errorf("send http post to dcos deployment error %v", err)
		return componentinfo, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		saLog.Errorf("http status code from dcos deployment failed %v", string(data))
		return componentinfo, errors.New("http status code from dcos deployment failed")
	}

	componentinfo = entity.ComponentsInfo{}
	err = getRetFromResponse(data, &componentinfo)
	return

}

func CreatelogCluster(status string, operation string, querytype string, components string) (newlog entity.LogMessage) {
	newlog.OperateType = operation
	newlog.QueryType = querytype
	newlog.Status = status
	newlog.Comments = components
	return
}
