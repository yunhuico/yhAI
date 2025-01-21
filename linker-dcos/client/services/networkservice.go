package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"

	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/jmoiron/jsonq"
	"gopkg.in/mgo.v2/bson"
	command "linkernetworks.com/dcos-backend/common/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	NETWORK_ERROR_CREATE         string = "E71001"
	NETWORK_ERROR_QUERY          string = "E71002"
	NETWORK_ERROR_DELETE         string = "E71003"
	NETWORK_ERROR_ISEXIST        string = "E71004"
	NETWORK_ERROR_NETWORKISUSING string = "E71005"
	NETWORK_ERROR_CHECKFAILED    string = "E71006"
	NETWORK_ERROR_PARSEFAILED    string = "E71007"
	NETWORK_ERROR_NUM            string = "E71008"

	DOCKERMACHINE_STORAGEPATH_PREFIX string = "/linker/docker/"
)

var (
	networkService *NetworkService = nil
	onceNetwork    sync.Once
)

type NetworkService struct {
	collectionName string
}

func GetNetworkService() *NetworkService {
	onceNetwork.Do(func() {
		logrus.Debugf("Once called from networkService ......................................")
		networkService = &NetworkService{"network"}
	})
	return networkService
}

func (p *NetworkService) GetOvsNetwork(cluster_id string, hostname entity.HostNames, token string) (total entity.Total, errcode string, err error) {
	logrus.Infof("start to get ovs num from db")
	if len(hostname.Names) == 0 {
		logrus.Errorf("the hostname can not be 0")
		errcode = NETWORK_ERROR_NUM
		err = errors.New("hostname len cannot be 0")
		return
	}

	var Num int
	for _, host := range hostname.Names {
		selector, selector1, selector2, selector3 := bson.M{}, bson.M{}, bson.M{}, bson.M{}
		selector["cluster_id"] = cluster_id
		selector1["clust_host_name"] = host
		selector2["network.driver"] = "ovs"
		selector3["$and"] = []bson.M{selector, selector1, selector2}
		queryStruct := dao.QueryStruct{
			CollectionName: p.collectionName,
			Selector:       selector3,
			Skip:           0,
			Limit:          0,
			Sort:           ""}

		network := []entity.ClusterNetwork{}
		_, errH := dao.HandleQueryAll(&network, queryStruct)
		if errH != nil {
			logrus.Errorln("query user by state error %v", errH)
			errcode = NETWORK_ERROR_QUERY
			return total, errcode, errH
		}
		logrus.Infof("network is %v", network)

		Num = Num + len(network)
	}
	total.Num = Num
	return
}

func (p *NetworkService) CreateNetwork(clusterNetwork entity.ClusterNetwork, x_auth_token string) (newNetwork *entity.ClusterNetwork,
	errorCode string, err error) {
	logrus.Infof("start to create network [%v]", clusterNetwork)

	// do authorize first
	// if authorized := services.GetAuthService().Authorize("create_network", x_auth_token, "", p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("create network [%v] error is %v", clusterNetwork, err)
	// 	return
	// }

	//call docker machine to create the overlay network
	output, _, err := DockerMachineCreateNetwork(clusterNetwork)
	if err != nil {
		errorCode = NETWORK_ERROR_CREATE
		logrus.Errorf("create network [%v] to in docker error is %v", clusterNetwork, err)
		return
	}

	// generate ObjectId
	clusterNetwork.ObjectId = bson.NewObjectId()
	clusterNetwork.NetworkId = output

	// set created_time and updated_time
	clusterNetwork.TimeCreate = dao.GetCurrentTime()
	clusterNetwork.TimeUpdate = clusterNetwork.TimeCreate

	// insert bson to mongodb
	err = dao.HandleInsert(p.collectionName, clusterNetwork)
	if err != nil {
		errorCode = NETWORK_ERROR_CREATE
		logrus.Errorf("create network [%v] to db error is %v", clusterNetwork, err)
		return
	}

	newNetwork = &clusterNetwork

	return
}

func (p *NetworkService) CleanOvs(cluster_id, host_name, x_auth_token string) (errorCode string, err error) {
	logrus.Infof("start to delete ovs network")
	if !bson.IsObjectIdHex(cluster_id) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	selector, selector1, selector2, selector3 := bson.M{}, bson.M{}, bson.M{}, bson.M{}
	selector["cluster_id"] = cluster_id
	selector1["clust_host_name"] = host_name
	selector2["network.driver"] = "ovs"
	selector3["$and"] = []bson.M{selector, selector1, selector2}
	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionName,
		Selector:       selector3,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	network := []entity.ClusterNetwork{}
	_, errH := dao.HandleQueryAll(&network, queryStruct)
	if errH != nil {
		logrus.Errorln("query user by state error %v", errH)
		return
	}
	logrus.Infof("network is %v", network)

	if len(network) != 0 {
		for _, ovsnet := range network {
			var selector4 = bson.M{}
			selector4["_id"] = ovsnet.ObjectId.Hex()
			logrus.Infof("want delete ovs id is %v", ovsnet.ObjectId.Hex())
			err = dao.HandleDelete(p.collectionName, true, selector3)
			if err != nil {
				logrus.Errorf("delete network [objectId=%v] in db error is %v", cluster_id, err)
				errorCode = NETWORK_ERROR_DELETE
				continue
			}
		}
	}

	return
}

func (p *NetworkService) CleanCluster(cluster_id, x_auth_token string) (
	errorCode string, err error) {
	if !bson.IsObjectIdHex(cluster_id) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// do authorize first
	// if authorized := services.GetAuthService().Authorize("delete_networks", x_auth_token, cluster_id, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("delete networks with clusterId [%v] error is %v", cluster_id, err)
	// 	return
	// }

	var selector = bson.M{}
	selector["cluster_id"] = cluster_id

	err = dao.HandleDelete(p.collectionName, false, selector)
	if err != nil {
		logrus.Errorf("delete networks [cluster_id=%v] error is %v", cluster_id, err)
		errorCode = NETWORK_ERROR_DELETE
	}
	return

}

func (p *NetworkService) DeleteById(objectId string, x_auth_token string) (network entity.ClusterNetwork,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	clusterNetwork, code, err := p.QueryById(objectId, x_auth_token)
	if err != nil {
		logrus.Errorf("delete network [objectId=%v] error is %v", objectId, err)
		errorCode = code
		return
	}
	network = clusterNetwork

	// do authorize first
	// if authorized := services.GetAuthService().Authorize("delete_network", x_auth_token, objectId, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("delete network with objectId [%v] error is %v", objectId, err)
	// 	return
	// }
	output, _, err := DockerMachineCheckNetwork(clusterNetwork)
	if err != nil {
		logrus.Errorf("delete network [objectId=%v]  error is %v", objectId, err)
		errorCode = NETWORK_ERROR_CHECKFAILED
		return
	}

	output = strings.TrimPrefix(output, "[")
	output = strings.TrimSuffix(output, "]")
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(output))
	dec.Decode(&data)
	jq := jsonq.NewQuery(data)

	container, err := jq.Object("Containers")

	if err != nil {
		logrus.Errorf("delete network [objectId=%v], docker network inspect json string parse error, details is %v", objectId, err)
		errorCode = NETWORK_ERROR_PARSEFAILED
		return
	}

	if len(container) != 0 {
		err = errors.New("some services are using the network, can't delete")
		logrus.Errorf("delete network [objectId=%v]  error is %v", objectId, err)
		errorCode = NETWORK_ERROR_NETWORKISUSING
		return
	}
	//call docker machine to delete the overlay network
	_, _, err = DockerMachineDeleteNetwork(clusterNetwork)
	if err != nil {
		logrus.Errorf("delete network [objectId=%v] in docker error is %v", objectId, err)
		errorCode = NETWORK_ERROR_DELETE
		return
	}

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)

	err = dao.HandleDelete(p.collectionName, true, selector)
	if err != nil {
		logrus.Errorf("delete network [objectId=%v] in db error is %v", objectId, err)
		errorCode = NETWORK_ERROR_DELETE
		return
	}
	return
}

func (p *NetworkService) QueryById(objectId string, x_auth_token string) (network entity.ClusterNetwork,
	errorCode string, err error) {
	if !bson.IsObjectIdHex(objectId) {
		err = errors.New("invalide ObjectId.")
		errorCode = COMMON_ERROR_INVALIDATE
		return
	}

	// do authorize first
	// if authorized := services.GetAuthService().Authorize("get_network", x_auth_token, objectId, p.collectionName); !authorized {
	// 	err = errors.New("required opertion is not authorized!")
	// 	errorCode = COMMON_ERROR_UNAUTHORIZED
	// 	logrus.Errorf("get network with objectId [%v] error is %v", objectId, err)
	// 	return
	// }

	var selector = bson.M{}
	selector["_id"] = bson.ObjectIdHex(objectId)
	network = entity.ClusterNetwork{}
	err = dao.HandleQueryOne(&network, dao.QueryStruct{p.collectionName, selector, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query network [objectId=%v] error is %v", objectId, err)
		errorCode = NETWORK_ERROR_QUERY
	}
	return
}

func (p *NetworkService) QueryAllByClusterId(clusterId string, skip int,
	limit int, sort string, x_auth_token string) (total int, networks []entity.ClusterNetwork,
	errorCode string, err error) {

	query := bson.M{}
	query["cluster_id"] = clusterId

	return p.queryByQuery(query, skip, limit, sort, x_auth_token, false)
}

func (p *NetworkService) queryByQuery(query bson.M, skip int, limit int,
	sort string, x_auth_token string, skipAuth bool) (total int, networks []entity.ClusterNetwork,
	errorCode string, err error) {
	// authQuery := bson.M{}
	// if !skipAuth {
	// 	// get auth query from auth first
	// 	authQuery, err = services.GetAuthService().BuildQueryByAuth("list_network", x_auth_token)
	// 	if err != nil {
	// 		logrus.Errorf("get auth query by token [%v] error is %v", x_auth_token, err)
	// 		errorCode = COMMON_ERROR_INTERNAL
	// 		return
	// 	}
	// }

	// selector := services.GenerateQueryWithAuth(query, authQuery)
	selector := query
	networks = []entity.ClusterNetwork{}

	queryStruct := dao.QueryStruct{p.collectionName, selector, skip, limit, sort}
	total, err = dao.HandleQueryAll(&networks, queryStruct)
	if err != nil {
		logrus.Errorf("query networks by query [%v] error is %v", query, err)
		errorCode = NETWORK_ERROR_QUERY
	}
	return
}

func DockerMachineCreateNetwork(network entity.ClusterNetwork) (output string, errput string, err error) {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + network.UserName + "/" + network.ClusterName

	envpath := storagePath + "/googlecredentials/service-account.json"
	if !CheckFileIsExist(envpath) {
		logrus.Infof("envpath does not exist %v", envpath)
	} else {
		logrus.Infof("start to set google env")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", envpath)
		env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		logrus.Infof("env is %v", env)
	}

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine  ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(network.ClusterHostName + ") ")
	commandTextBuffer.WriteString("network create ")
	if strings.TrimSpace(network.Network.Driver) != "" {
		commandTextBuffer.WriteString("--driver=" + network.Network.Driver + " ")
	}

	if strings.TrimSpace(network.Network.Internal) != "" {
		commandTextBuffer.WriteString("--internal=" + network.Network.Internal + " ")
	}

	for _, gateway := range network.Network.Gateway {
		commandTextBuffer.WriteString("--gateway=" + gateway + " ")
	}

	for _, iprange := range network.Network.IPRange {
		commandTextBuffer.WriteString("--ip-range=" + iprange + " ")
	}

	for _, subnet := range network.Network.Subnet {
		commandTextBuffer.WriteString("--subnet=" + subnet + " ")
	}

	if network.Network.Options != nil {
		for key, value := range network.Network.Options {
			commandTextBuffer.WriteString("-o " + key + "=" + value + " ")
		}

		for ke, _ := range network.Network.Options {
			if ke == "linker.net.ovs.bridge.type" && strings.TrimSpace(network.Network.Driver) == "ovs" {
				commandTextBuffer.WriteString("-o linker.net.ovs.network.name=" + network.Network.Name + " ")
				break
			}
		}
	}

	commandTextBuffer.WriteString(network.Network.Name)

	logrus.Infof("Executing Create Network command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	if len(errput) > 0 {
		err = errors.New(errput)
	}
	return
}

func DockerMachineDeleteNetwork(network entity.ClusterNetwork) (output string, errput string, err error) {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + network.UserName + "/" + network.ClusterName

	envpath := storagePath + "/googlecredentials/service-account.json"
	if !CheckFileIsExist(envpath) {
		logrus.Infof("envpath does not exist %v", envpath)
	} else {
		logrus.Infof("start to set google env")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", envpath)
		env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		logrus.Infof("env is %v", env)
	}

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine  ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(network.ClusterHostName + ") ")
	commandTextBuffer.WriteString("network rm ")
	commandTextBuffer.WriteString(network.Network.Name)

	logrus.Infof("Executing Delete Network command: %s", commandTextBuffer.String())
	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	if err != nil {
		logrus.Errorf("execute command to delete network[%s] error: %v", network.Network.Name, err)
		return
	}
	if len(errput) > 0 {
		err = errors.New(errput)
		logrus.Errorf("execute command to delete network[%s] return error: %v", network.Network.Name, err)
		return
	}
	return
}

func DockerMachineCheckNetwork(network entity.ClusterNetwork) (output string, errput string, err error) {
	storagePath := DOCKERMACHINE_STORAGEPATH_PREFIX + network.UserName + "/" + network.ClusterName

	envpath := storagePath + "/googlecredentials/service-account.json"
	if !CheckFileIsExist(envpath) {
		logrus.Infof("envpath does not exist %v", envpath)
	} else {
		logrus.Infof("start to set google env")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", envpath)
		env := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		logrus.Infof("env is %v", env)
	}

	var commandTextBuffer bytes.Buffer
	commandTextBuffer.WriteString("docker $(docker-machine  ")
	commandTextBuffer.WriteString("--storage-path " + storagePath + " ")
	commandTextBuffer.WriteString("config ")
	commandTextBuffer.WriteString(network.ClusterHostName + ") ")
	commandTextBuffer.WriteString("network inspect ")
	commandTextBuffer.WriteString(network.Network.Name)

	logrus.Infof("Executing Check Network command: %s", commandTextBuffer.String())

	output, errput, err = command.ExecCommand(commandTextBuffer.String())
	if len(errput) > 0 {
		err = errors.New(errput)
		return
	}
	return
}

func (p *NetworkService) CheckNetworkName(username string, networkname string, token string) (errorCode string, err error) {
	query := bson.M{}
	if len(username) == 0 {
		logrus.Errorf("username is empty")
		return "", errors.New("username is empty")
	}
	query["user_name"] = username
	networks := []entity.ClusterNetwork{}
	_, err = dao.HandleQueryAll(&networks, dao.QueryStruct{p.collectionName, query, 0, 0, ""})
	if err != nil {
		logrus.Errorf("query network err is %v", err)
		return NETWORK_ERROR_QUERY, err
	}
	var names []string
	if len(networks) != 0 {
		for _, network := range networks {
			names = append(names, network.Network.Name)
		}
		for _, name := range names {
			if name == networkname {
				logrus.Errorf("network name is exist")
				return NETWORK_ERROR_ISEXIST, errors.New("network name is exist")
			}
		}
	}
	return
}

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}
