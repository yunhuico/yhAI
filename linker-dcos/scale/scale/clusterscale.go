package scale

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"strings"

	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/scale/common"
)

func ScaleOut(username, password, clusterlb, clustername string, addnumber int) (err error) {
	logrus.Infof("scale out operation username %s, password %s, clusterlb %s, clustername %s, addmuber %d", username, password, clusterlb, clustername, addnumber)

	logrus.Infof("1. get valid token")
	tokenId, errs := common.GenerateToken(username, password, clusterlb)
	if errs != nil {
		logrus.Error("generate token error %s", errs)
		return errs
	}

	logrus.Infof("2. get cluster by id")
	cluster, errs := common.GetClusterByName(tokenId, clusterlb, clustername, common.CLUSTER_STATUS_RUNNING)
	if errs != nil {
		logrus.Errorf("get cluster by cluster name error %s", errs)
		return errs
	}

	logrus.Infof("3. check cluster. cluster status, type ...")
	if !strings.EqualFold(cluster.Status, "RUNNING") {
		logrus.Errorf("cluster status is not Running, can not do scaleout operation!, the status is %s", cluster.Status)
		return errors.New("cluster status is not Running")
	}

	if !strings.EqualFold(cluster.Type, "amazonec2") && !strings.EqualFold(cluster.Type, "google") {
		logrus.Errorf("currently only support amazonec2 and google platform, the cluster's platform type is %s", cluster.Type)
		return errors.New("specific cluster's creation type is not supported for scaleOut and scaleIn operation")
	}

	logrus.Info("4. begin to do scaleOut VM")
	clusterId := cluster.ObjectId.Hex()
	addrequest := entity.AddRequest{}
	addrequest.ClusterId = clusterId
	addrequest.AddNumber = addnumber
	addrequest.AddMode = "new" //currently only support this mode!!!!

	err = common.SendScaleOut(clusterlb, clusterId, tokenId, addrequest)
	if err != nil {
		logrus.Errorf("scale out operation failed %v", err)
		return
	}
	logrus.Info("send scale out vm operation success!")

	logrus.Info("5. waiting to do scaleOut linkerconnector")
	errl := common.WaitingAndScaleApp(clustername, tokenId, clusterlb, common.OPERATION_ADD, addnumber)
	if errl != nil {
		logrus.Errorf("scale out linkerconnector error %v", errl)
		return
	}

	return

}

func ScaleIn(username, password, clusterlb, clustername string, removeIpList []string) (err error) {
	logrus.Infof("scale in operation username %s, password %s, clusterlb %s clustername %s, remove node list %v", username, password, clusterlb, clustername, removeIpList)

	logrus.Infof("1. get valid token")
	tokenId, errs := common.GenerateToken(username, password, clusterlb)
	if errs != nil {
		logrus.Error("generate token error %s", errs)
		return errs
	}

	logrus.Infof("2. get cluster by id")
	cluster, errs := common.GetClusterByName(tokenId, clusterlb, clustername, common.CLUSTER_STATUS_RUNNING)
	if errs != nil {
		logrus.Errorf("get cluster by cluster name error %s", errs)
		return errs
	}

	logrus.Infof("3. check cluster. cluster status, type...")
	if !strings.EqualFold(cluster.Status, "RUNNING") {
		logrus.Errorf("cluster status is not Running, can not do scalein operation!, the status is %s", cluster.Status)
		return errors.New("cluster status is not Running")
	}

	if !strings.EqualFold(cluster.Type, "amazonec2") && !strings.EqualFold(cluster.Type, "google") {
		logrus.Errorf("currently only support amazonec2 and google platform, the cluster's platform type is %s", cluster.Type)
		return errors.New("specific cluster's creation type is not supported for scaleOut and scaleIn operation")
	}

	clusterId := cluster.ObjectId.Hex()

	logrus.Infof("4. check removed node")
	allhosts, errg := common.GetHostsByClusterId(tokenId, clusterlb, clusterId)
	if errg != nil {
		logrus.Errorf("get cluster host by clusterid error %v", errg)
		return errg
	}

	removedNodeIds := []string{}
	for _, hostinfo := range allhosts {
		if nodeExist(hostinfo, removeIpList) {
			removedNodeIds = append(removedNodeIds, hostinfo.HostId)
		}
	}
	logrus.Infof("the removed nodes id: %v", removedNodeIds)

	logrus.Infof("5. begin to do scale In")
	removeRequest := common.TerminateHostsRequestBody{}
	removeRequest.HostIds = removedNodeIds

	err = common.SendScaleIn(clusterlb, clusterId, tokenId, removeRequest)
	if err != nil {
		logrus.Errorf("scale in operation failed %v", err)
		return
	}

	logrus.Info("send scale in operation success!")

	logrus.Info("6. waiting to do scaleOut linkerconnector")
	errl := common.WaitingAndScaleApp(clustername, tokenId, clusterlb, common.OPERATION_REMOVE, len(removedNodeIds))
	if errl != nil {
		logrus.Errorf("scale in linkerconnector error %v", errl)
		return
	}

	return
	return

}

func nodeExist(hostinfo entity.HostInfo, nodeIpList []string) bool {
	for _, nodeip := range nodeIpList {
		if strings.EqualFold(nodeip, hostinfo.PrivateIp) ||
			strings.EqualFold(nodeip, hostinfo.IP) {
			return true
		}
	}

	return false
}