package services

import (
	"encoding/json"
	"errors"

	"github.com/Sirupsen/logrus"

	common "linkernetworks.com/dcos-backend/cluster/common"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

//call dcos deployment module to create a cluster
func CreateCluster(cluster entity.Cluster, createRequest entity.CreateRequest, logobjid string, x_auth_token string) (err error) {
	var request entity.Request
	clustername := cluster.Name
	cLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	var pubkeyid []string
	if len(cluster.PubKeyId) != 0 {
		for _, sub := range cluster.PubKeyId {
			pubkeyid = append(pubkeyid, sub)
		}
	}

	rePubkey := make([]entity.PubkeyInfo, len(pubkeyid))
	if len(pubkeyid) > 0 {
		for i, subpubid := range pubkeyid {
			pubkey, _, errq := GetPubKeyService().QueryById(subpubid, x_auth_token)
			if errq != nil {
				cLog.Errorf("get pubkey error in create cluster %v", errq)
				continue
			} else {
				rePubkey[i].Value = pubkey.PubKeyValue
				rePubkey[i].Name = pubkey.Name
				rePubkey[i].Id = pubkey.ObjectId.Hex()
			}
		}
	}

	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	//	request.ClusterNumber = cluster.Instances
	request.PubKey = rePubkey
	request.IsLinkerMgmt = false
	request.CreateCategory = cluster.CreateCategory
	request.NodeAttribute = createRequest.NodeAttribute
	request.XAuthToken = x_auth_token //token
	request.LogId = logobjid
	request.UserId = cluster.UserId
	request.TenantId = cluster.TenantId

	if createRequest.CreateCategory == "compact" {
		if createRequest.MasterCount != 1 {
			cLog.Errorf("compact cluster has one masternodes")
			return errors.New("compact cluster has one masternodes")
		}
	} else if createRequest.CreateCategory == "ha" {
		if createRequest.MasterCount != 3 {
			cLog.Errorf("ha cluster has three masternodes")
			return errors.New("ha cluster has three masternodes")
		}
	}
	request.MasterCount = createRequest.MasterCount
	request.SharedCount = createRequest.SharedCount
	request.PureSlaveCount = createRequest.PureSlaveCount

	if len(createRequest.DockerRegistries) > 0 {
		request.DockerRegistries = createRequest.DockerRegistries
	}
	if len(createRequest.EngineOpts) > 0 {
		request.EngineOpts = createRequest.EngineOpts
	}

	cLog.Debugf("Docker registry number: %d", len(request.DockerRegistries))

	if cluster.Type == "customized" {
		request.CreateMode = "reuse"
		request.MasterNodes = createRequest.MasterNodes
		request.SharedNodes = createRequest.SharedNodes
		request.PureSlaveNodes = createRequest.PureSlaveNodes
		request.ProviderInfo = entity.ProviderInfo{
			Properties: map[string]string{"driver": PROVIDER_GENERIC_TYPE}}
	} else {
		request.CreateMode = "new"
		providerInfo, errp := buildProviderInfo(cluster.ProviderId, x_auth_token)
		if errp != nil {
			cLog.Errorf("build provider info error when create cluster %v", errp)
			return errp
		}
		request.ProviderInfo = providerInfo
	}

	clusterId := cluster.ObjectId.Hex()

	request.ClusterId = clusterId

	_, err = SendCreateClusterRequest(request)

	return
}

func buildProviderInfo(id string, token string) (providerInfo entity.ProviderInfo, err error) {
	provider, errp := getProvider(id, token)
	if errp != nil {
		return providerInfo, errp
	}

	properties := make(map[string]string)

	var iaasInfo interface{}
	driver := provider.Type
	if driver == PROVIDER_EC2_TYPE {
		iaasInfo = provider.AwsEC2Info
	} else if driver == PROVIDER_OPENSTACK_TYPE {
		iaasInfo = provider.OpenstackInfo
	} else if driver == PROVIDER_GOOGLE_TYPE {
		iaasInfo = provider.GoogleInfo
	} else {
		logrus.Errorf("not supported provider type %s", driver)
		return providerInfo, errors.New("not supported provider type")
	}

	//convert struct to map
	providerjson, errj := json.Marshal(iaasInfo)
	if errj != nil {
		logrus.Errorf("marshal provider error %v", errj)
		return providerInfo, errj
	}
	err = json.Unmarshal(providerjson, &properties)
	if err != nil {
		logrus.Errorf("unmarshal provider error %v", err)
		return
	}

	properties["driver"] = driver

	providerInfo = entity.ProviderInfo{
		Provider:   entity.Provider{ProviderType: driver, SshUser: provider.SshUser},
		Properties: properties,
	}

	logrus.Debugf("providerInfo  is %v", providerInfo)
	return

}

func getProvider(id string, token string) (provider entity.IaaSProvider, err error) {
	logrus.Infof("get provider by id [%s]", id)

	provider, _, err = GetProviderService().QueryById(id, token)
	if err != nil {
		logrus.Errorf("get provider error [%v]", err)
		return
	}

	if provider.Type == PROVIDER_EC2_TYPE {
		accessKey := provider.AwsEC2Info.AccessKey
		secretKey := provider.AwsEC2Info.SecretKey

		provider.AwsEC2Info.AccessKey = string(common.Base64Encode([]byte(accessKey)))
		provider.AwsEC2Info.SecretKey = string(common.Base64Encode([]byte(secretKey)))
	}

	logrus.Debugf("the provider is %v", provider)

	return
}

//call dcos deployment module to delete a cluster
func DeleteCluster(cluster entity.Cluster, logobjid string, x_auth_token string) (err error) {

	request := new(entity.DeleteAllRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	// request.Servers = servers
	request.XAuthToken = x_auth_token //token
	request.LogId = logobjid
	request.ClusterId = cluster.ObjectId.Hex()
	request.ClusterMgmtIp = cluster.Endpoint

	_, err = SendDeleteClusterRequest(request)

	return
}

//call dcos deployment module to add nodes
func AddNodes(cluster entity.Cluster, addrequest entity.AddRequest, hosts []entity.Host, logobjid string, x_auth_token string) (err error) {
	clustername := cluster.Name
	aLog := logrus.WithFields(logrus.Fields{"clustername": clustername})
	var pubkeyid []string
	if len(cluster.PubKeyId) > 0 {
		for _, subpub := range cluster.PubKeyId {
			pubkeyid = append(pubkeyid, subpub)
		}
	}

	rePubkey := make([]entity.PubkeyInfo, len(pubkeyid))
	if len(pubkeyid) > 0 {
		for i, subpubid := range pubkeyid {
			pubkey, _, errq := GetPubKeyService().QueryById(subpubid, x_auth_token)
			if errq != nil {
				aLog.Errorf("get pubkey error in create cluster %v", errq)
				continue
			} else {
				rePubkey[i].Value = pubkey.PubKeyValue
				rePubkey[i].Name = pubkey.Name
				rePubkey[i].Id = pubkey.ObjectId.Hex()
			}
		}
	}

	//	createNumber := addrequest.AddNumber
	request := new(entity.AddNodeRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	//	request.AddNumber = createNumber
	request.SharedCount = addrequest.SharedCount
	request.PureSlaveCount = addrequest.PureSlaveCount
	request.ExistedNumber = cluster.Instances
	request.PubKey = rePubkey
	request.AddMode = addrequest.AddMode
	//	request.AddNodes = addrequest.AddNode
	request.SharedNodes = addrequest.SharedNodes
	request.PureSlaveNodes = addrequest.PureSlaveNodes
	request.NodeAttribute = addrequest.NodeAttribute
	request.XAuthToken = x_auth_token //token
	request.LogId = logobjid

	if len(addrequest.DockerRegistries) > 0 {
		request.DockerRegistries = addrequest.DockerRegistries
	}
	if len(addrequest.EngineOpts) > 0 {
		request.EngineOpts = addrequest.EngineOpts
	}

	aLog.Infof("Docker registry number: %d", len(request.DockerRegistries))

	if addrequest.AddMode == "new" {
		providerInfo, errp := buildProviderInfo(cluster.ProviderId, x_auth_token)
		if errp != nil {
			aLog.Errorf("build provider info error when add node to cluster %v", errp)
			return errp
		}
		request.ProviderInfo = providerInfo
		//	arry := make([]entity.Node, createNumber)
		//	request.AddNodes.Nodes = arry
		//		arry := make([]entity.Node, addrequest.SharedCount)
		//		request.SharedNodes = arry
		//		arryP := make([]entity.Node, addrequest.PureSlaveCount)
		//		request.PureSlaveNodes = arryP
	} else if addrequest.AddMode == "reuse" {
		request.ProviderInfo = entity.ProviderInfo{
			Properties: map[string]string{"driver": PROVIDER_GENERIC_TYPE}}
	} else {
		aLog.Errorf("no supported mode %s", addrequest.AddMode)
		return errors.New("no supported add mode")
	}

	_, currentHosts, _, err := GetHostService().QueryHosts(cluster.ObjectId.Hex(), 0, 0, "unterminated", x_auth_token)
	if err != nil {
		aLog.Errorf("get current hosts by clusterId error %v", err)
		return err
	}

	request.DnsServers, request.SwarmMaster, request.MonitorServerHostName = getNodeInfo(currentHosts)

	aLog.Debugf("add node request is %v", request)

	_, err = SendAddNodesRequest(request)

	return
}

func buildEndPoint(servers []entity.Server, clusterName string) string {
	logrus.Infof("build endpoint for %v", clusterName)
	address := ""
	for _, server := range servers {
		if server.IsMaster && server.IsClientServer {
			address = server.IpAddress

			url := address

			logrus.Infof("cluster [%v] endpoint is : %v", clusterName, url)
			return url
		}
	}

	logrus.Warnf("does find cluster [%s] endpoint", clusterName)
	return ""
}

func getNodeInfo(hosts []entity.Host) (dns []entity.Server, swarm, monitorHostname string) {
	if hosts == nil || len(hosts) <= 0 {
		logrus.Warnf("no node for current cluster!")
		return
	}

	dns = []entity.Server{}
	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		if host.IsMasterNode {
			swarm = host.HostName
		}

		if host.IsMonitorServer {
			monitorHostname = host.HostName
		}

		if host.IsMasterNode {
			server := entity.Server{}

			server.Hostname = host.HostName
			server.IpAddress = host.IP
			server.PrivateIpAddress = host.PrivateIp
			server.IsMaster = host.IsMasterNode
			server.IsSlave = host.IsSlaveNode
			server.IsFullfilled = host.IsFullfilled
			server.IsSharedServer = host.IsSharedNode
			server.IsMonitorServer = host.IsMonitorServer

			dns = append(dns, server)
		}

	}

	return
}

//call dcos deployment module to delete nodes
func DeleteNodes(cluster entity.Cluster, hosts []entity.Host, logobjid string, x_auth_token string, nowshared int) (err error) {
	// var servers []dcosentity.Server
	servers := []entity.Server{}
	for _, host := range hosts {
		server := entity.Server{}
		server.Hostname = host.HostName
		server.IpAddress = host.IP
		server.PrivateIpAddress = host.PrivateIp
		server.IsMaster = host.IsMasterNode
		server.IsSlave = host.IsSlaveNode
		server.IsSharedServer = host.IsSharedNode
		server.IsMonitorServer = host.IsMonitorServer

		servers = append(servers, server)
	}

	request := new(entity.DeleteRequest)
	request.UserName = cluster.Owner
	request.ClusterName = cluster.Name
	request.Servers = servers
	// request.ExistedNumber = cluster.Instances
	request.XAuthToken = x_auth_token
	request.LogId = logobjid
	request.NowShared = nowshared

	_, currentHosts, _, err := GetHostService().QueryHosts(cluster.ObjectId.Hex(), 0, 0, "unterminated", x_auth_token)
	if err != nil {
		logrus.Errorf("get current hosts by clusterId error %v", err)
		return err
	}

	request.DnsServers, _, _ = getNodeInfo(currentHosts)

	_, err = SendDeleteNodesRequest(request)

	return
}

func AddPub(clusterName string, userName string, hosts []entity.Host, sshUser string, pubkeyIds []string, x_auth_token string) (err error) {
	logrus.Infof("start to create add pubkey request to %v", clusterName)
	request := new(entity.AddPubkeysRequest)
	request.UserName = userName
	request.ClusterName = clusterName
	request.XAuthToken = x_auth_token
	arry := make([]entity.HostsPubInfo, len(hosts))
	for i, host := range hosts {
		logrus.Infof("hostname is %v", host.HostName)
		arry[i].HostName = host.HostName
		arry[i].IP = host.IP
		if sshUser == "" {
			arry[i].SshUser = host.SshUser
		} else {
			arry[i].SshUser = sshUser
		}

	}
	request.Hosts = arry
	logrus.Infof("request host is %v", request.Hosts)

	keys := make([]entity.PubkeyInfo, len(pubkeyIds))
	logrus.Infof("pubkeyids is %v", pubkeyIds)
	for i, pubkeyid := range pubkeyIds {
		if pubkeyid != "" {
			value, _, errq := GetPubKeyService().QueryById(pubkeyid, x_auth_token)
			logrus.Infof("value is %v", value)
			if errq != nil {
				logrus.Errorf("query pubkey err is %v", errq)
				continue
			}
			keys[i].Id = pubkeyid
			keys[i].Value = value.PubKeyValue
			keys[i].Name = value.Name
		}

	}
	request.Pubkey = keys

	logrus.Infof("request to add pubkey is %v", request)

	_, err = SendAddPubkeysRequest(request)
	return
}

func DeletePub(clusterName string, userName string, hosts []entity.Host, sshUser string, pubkeyIds []string, x_auth_token string) (err error) {
	logrus.Infof("start to create delete pubkey request to %v", clusterName)
	if len(pubkeyIds) == 0 {
		logrus.Errorf("there is no pubkey to delete")
		return
	}
	request := new(entity.DeletePubkeysRequest)
	request.UserName = userName
	request.ClusterName = clusterName
	arry := make([]entity.HostsPubInfo, len(hosts))
	for i, host := range hosts {
		logrus.Infof("hostname is %v", host.HostName)
		arry[i].HostName = host.HostName
		arry[i].IP = host.IP
		if sshUser == "" {
			arry[i].SshUser = host.SshUser
		} else {
			arry[i].SshUser = sshUser
		}
	}
	request.Hosts = arry
	logrus.Infof("request host is %v", request.Hosts)

	keys := make([]entity.PubkeyInfo, len(pubkeyIds))
	logrus.Infof("delete pubkeyid is %v", pubkeyIds)
	for i, pubkeyid := range pubkeyIds {
		logrus.Infof("start to query pubkey!!")
		if pubkeyid != "" {
			value, _, errq := GetPubKeyService().QueryById(pubkeyid, x_auth_token)
			logrus.Infof("value is %v", value)
			if errq != nil {
				logrus.Errorf("query pubkey err is %v", errq)
				continue
			}
			keys[i].Id = pubkeyid
			keys[i].Value = value.PubKeyValue
			keys[i].Name = value.Name
		}
	}
	request.Pubkey = keys
	logrus.Infof("request to delete pubkey is %v", request)

	_, err = SendDelePubkeysRequest(request)
	return
}

func AddRegi(clusterName string, userName string, hosts []entity.Host, sshUser string, registrys []entity.DockerRegistry, x_auth_token string) (err error) {
	logrus.Infof("start to create add registry request to %v", clusterName)
	request := new(entity.AddRegistryRequest)
	request.UserName = userName
	request.ClusterName = clusterName
	request.XAuthToken = x_auth_token
	arry := make([]entity.HostsPubInfo, len(hosts))
	for i, host := range hosts {
		logrus.Infof("hostname is %v", host.HostName)
		arry[i].HostName = host.HostName
		arry[i].IP = host.IP
		if sshUser == "" {
			arry[i].SshUser = host.SshUser
		} else {
			arry[i].SshUser = sshUser
		}

	}
	request.Hosts = arry
	logrus.Infof("request host is %v", request.Hosts)

	if len(registrys) == 0 {
		logrus.Errorf("add registry is 0")
		return
	}

	request.Registrys = registrys
	logrus.Infof("request to add registry is %v", request)

	_, err = SendAddRegistryRequest(request)
	return

}

func DeleRegi(clusterName string, userName string, hosts []entity.Host, sshUser string, registrys []entity.DockerRegistry, x_auth_token string) (err error) {
	logrus.Infof("start to create delete registry request to %v", clusterName)
	request := new(entity.DeleteRegistryRequest)
	request.UserName = userName
	request.ClusterName = clusterName
	request.XAuthToken = x_auth_token

	arry := make([]entity.HostsPubInfo, len(hosts))
	for i, host := range hosts {
		logrus.Infof("hostname is %v", host.HostName)
		arry[i].HostName = host.HostName
		arry[i].IP = host.IP
		if sshUser == "" {
			arry[i].SshUser = host.SshUser
		} else {
			arry[i].SshUser = sshUser
		}

	}
	request.Hosts = arry
	logrus.Infof("request host is %v", request.Hosts)

	if len(registrys) == 0 {
		logrus.Errorf("delete registry is 0")
		return
	}
	request.Registrys = registrys
	logrus.Infof("request to delete registry is %v", request)

	_, err = SendDeleteRegistryRequest(request)
	return

}
