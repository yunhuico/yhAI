package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
	//	"linkernetworks.com/linker_common_lib/entity"
)

type CreateRequest struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`
	//	Instances        int              `json:"instances"`
	MasterCount    int      `json:"masterCount"`
	SharedCount    int      `json:"sharedCount"`
	PureSlaveCount int      `json:"pureslaveCount"`
	PubKeyId       []string `json:"pubkeyId"`
	ProviderId     string   `json:"providerId"`
	Details        string   `json:"details"`
	UserId         string   `json:"user_id"`
	//	CreateNode       NodesInfo        `json:"createNode"`
	MasterNodes      []Node           `json:"masterNodes"`
	SharedNodes      []Node           `json:"sharedNodes"`
	PureSlaveNodes   []Node           `json:"pureslaveNodes`
	Type             string           `json:"type"`           // "amazonec2" "google" "customized"
	CreateCategory   string           `json:"createCategory"` //compact or ha
	NodeAttribute    string           `json:"nodeAttribute"`
	DockerRegistries []DockerRegistry `json:"dockerRegistries"`
	EngineOpts       []EngineOpt      `json:"engineOpts"`
}

type AddRequest struct {
	ClusterId string `json:"clusterId"`
	//	AddNumber        int              `json:"addNumber"`
	//	AddNode          NodesInfo        `json:"addNode"`
	SharedCount      int              `json:"sharedCount"`
	PureSlaveCount   int              `json:"pureslaveCount"`
	AddMode          string           `json:"addMode"` // "reuse" "new"
	SharedNodes      []Node           `json:"sharedNodes"`
	PureSlaveNodes   []Node           `json:"pureslaveNodes"`
	NodeAttribute    string           `json:"nodeAttribute"`
	DockerRegistries []DockerRegistry `json:"dockerRegistries"`
	EngineOpts       []EngineOpt      `json:"engineOpts"`
}

type Cluster struct {
	ObjectId         bson.ObjectId    `bson:"_id" json:"_id"`
	Name             string           `bson:"name" json:"name"`
	Owner            string           `bson:"owner" json:"owner"`
	Endpoint         string           `bson:"endPoint" json:"endPoint"`
	Instances        int              `bson:"instances" json:"instances"`
	PubKeyId         []string         `bson:"pubkeyId" json:"pubkeyId"`
	ProviderId       string           `bson:"providerId" json:"providerId"`
	Details          string           `bson:"details" json:"details"`
	Status           string           `bson:"status" json:"status"`
	Type             string           `bson:"type" json:"type"`                     // "amazonec2" "openstack" "customized"
	CreateCategory   string           `json:"createCategory" json:"createCategory"` // compact or ha
	UserId           string           `bson:"user_id" json:"user_id"`
	TenantId         string           `bson:"tenant_id" json:"tenant_id"`
	TimeCreate       time.Time        `bson:"time_create" json:"time_create"`
	TimeUpdate       time.Time        `bson:"time_update" json:"time_update"`
	DockerRegistries []DockerRegistry `bson:"dockerRegistries" json:"dockerRegistries" `
	SetProjectValue  Project          `bson:"setProjectvalue" json:"setProjectvalue"`
}

type Project struct {
	Cmi bool `bson:"cmi" json:"cmi"`
}

type Host struct {
	ObjectId        bson.ObjectId `bson:"_id" json:"_id"`
	HostName        string        `bson:"hostName" json:"hostName"`
	ClusterId       string        `bson:"clusterId" json:"clusterId"`
	ClusterName     string        `bson:"clusterName" json:"clusterName"`
	Status          string        `bson:"status" json:"status"`
	IP              string        `bson:"ip" json:"ip"`
	PrivateIp       string        `bson:"privateIp" json:"privateIp"`
	IsMasterNode    bool          `bson:"isMasterNode" json:"isMasterNode"`
	IsSlaveNode     bool          `bson:"isSlaveNode" json:"isSlaveNode"`
	IsMonitorServer bool          `bson:"isMonitorServer" json:"isMonitorServer"`
	IsSharedNode    bool          `bson:"isSharedNode" json:"isSharedNode"`
	IsFullfilled    bool          `bson:"isFullfilled" json:"isFullfilled"`
	IsClientNode    bool          `bson:"isClientNode" json:"isClientNode"`
	UserId          string        `bson:"user_id" json:"user_id"`
	TenantId        string        `bson:"tenant_id" json:"tenant_id"`
	Type            string        `bson:"type" json:"type"`
	SshUser         string        `bson:"sshUser" json:"sshUser"`
	TimeCreate      time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate      time.Time     `bson:"time_update" json:"time_update"`
}

type HostInfo struct {
	HostId       string    `json:"hostId"`
	HostName     string    `json:"hostName"`
	ClusterId    string    `json:"clusterId"`
	ClusterName  string    `json:"clusterName"`
	Status       string    `json:"status"`
	IP           string    `json:"ip"`
	PrivateIp    string    `json:"privateIp"`
	Task         int       `json:"task"`
	CPU          string    `json:"cpu"`
	Memory       string    `json:"memory"`
	GPU          string    `json:"gpu"`
	Tag          []string  `json:"tag"`
	IsMasterNode bool      `json:"isMasterNode"`
	IsSlaveNode  bool      `json:"isSlaveNode"`
	IsSharedNode bool      `json:"isSharedNode"`
	IsFullfilled bool      `json:"isFullfilled"`
	IsClientNode bool      `bson:"isClientNode" json:"isClientNode"`
	UserId       string    `json:"user_id"`
	TenantId     string    `json:"tenant_id"`
	PubKeyName   string    `json:"pubkeyName"`
	Type         string    `bson:"type" json:"type"`
	TimeCreate   time.Time `bson:"time_create" json:"time_create"`
	TimeUpdate   time.Time `bson:"time_update" json:"time_update"`
}

type StateSummary struct {
	Slaves []Slave `json:"slaves"`
}

type Slave struct {
	TaskRunning   int         `json:"TASK_RUNNING"`
	HostName      string      `json:"hostname"`
	Attributes    interface{} `json:"attributes"`
	Resources     Resource    `json:"resources"`
	UsedResources Resource    `json:"used_resources"`
}
type Resource struct {
	CPUs float64 `json:"cpus"`
	Mem  float64 `json:"mem"`
	GPUs float64 `json:"gpus"`
}

type IaaSProvider struct {
	ObjectId      bson.ObjectId `bson:"_id" json:"_id"`
	Name          string        `bson:"name" json:"name"`
	Type          string        `bson:"type" json:"type"`
	SshUser       string        `bson:"sshUser" json:"sshUser"`
	OpenstackInfo Openstack     `bson:"openstackInfo,omitempty" json:"openstackInfo,omitempty"`
	AwsEC2Info    AwsEC2        `bson:"awsEc2Info,omitempty" json:"awsEc2Info,omitempty"`
	GoogleInfo    Google        `bson:"googleInfo,omitempty" json:"googleInfo,omitempty"`
	UserId        string        `bson:"user_id" json:"user_id"`
	TenantId      string        `bson:"tenant_id" json:"tenant_id"`
	TimeCreate    time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate    time.Time     `bson:"time_update" json:"time_update"`
}

type EngineOpt struct {
	OptKey   string `json:"optkey"`
	OptValue string `json:"optvalue"`
}

type ProviderListInfo struct {
	ObjectId      bson.ObjectId `bson:"_id" json:"_id"`
	Name          string        `bson:"name" json:"name"`
	Type          string        `bson:"type" json:"type"`
	SshUser       string        `bson:"sshUser" json:"sshUser"`
	OpenstackInfo Openstack     `bson:"openstackInfo,omitempty" json:"openstackInfo,omitempty"`
	AwsEC2Info    AwsEC2        `bson:"awsEc2Info,omitempty" json:"awsEc2Info,omitempty"`
	GoogleInfo    Google        `bson:"googleInfo,omitempty" json:"googleInfo,omitempty"`
	UserId        string        `bson:"user_id" json:"user_id"`
	TenantId      string        `bson:"tenant_id" json:"tenant_id"`
	IsUse         bool          `bson:"isuse" json:"isuse"`
	TimeCreate    time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate    time.Time     `bson:"time_update" json:"time_update"`
}

type PubKey struct {
	ObjectId    bson.ObjectId `bson:"_id" json:"_id"`
	PubKeyValue string        `bson:"pubkeyValue" json:"pubkeyValue"`
	Name        string        `bson:"name" json:"name"`
	Owner       string        `bson:"owner" json:"owner"`
	UserId      string        `bson:"user_id" json:"user_id"`
	IsUse       bool          `bson:"isuse" json:"isuse"`
	TenantId    string        `bson:"tenant_id" json:"tenant_id"`
	TimeCreate  time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate  time.Time     `bson:"time_update" json:"time_update"`
}

type Smtp struct {
	ObjectId bson.ObjectId `bson:"_id" json:"_id"`
	Name     string        `bson:"name" json:"name"`
	Address  string        `bson:"address" json:"address"`
	PassWd   string        `bson:"passwd" json:"passwd"`
}

type LogMessage struct {
	ObjectId    bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
	ClusterName string        `bson:"clusterName" json:"clusterName"`
	ClusterId   string        `bson:"clusterId" json:"clusterId"`
	OperateType string        `bson:"operateType" json:"operateType"`
	QueryType string  `bson:"queryType" json:"queryType"`
	Username    string        `bson:"userName" json:"userName"`
	UserId      string        `bson:"user_id" json:"user_id"`
	TenantId    string        `bson:"tenant_id" json:"tenant_id"`
	Status      string        `bson:"status" json:"status"`
	Comments    string        `bson:"comments" json:"comments"`
	TimeCreate  time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate  time.Time     `bson:"time_update" json:"time_update"`
}

type HostNames struct {
	Names []string `json:"host_names"`
}

type Total struct {
	Num int `json:"num"`
}

type Components struct {
	ObjectId          bson.ObjectId `bson:"_id" json:"_id"`
	ClusterName       string        `bson:"clusterName" json:"clusterName"`
	ClusterId         string        `bson:"clusterId" json:"clusterId"`
	UserName          string        `bson:"userName" json:"userName"`
	UserId            string        `bson:"user_id" json:"user_id"`
	TenantId          string        `bson:"tenant_id" json:"tenant_id"`
	MasterComponents  Image         `bson:"masterComponents" json:"masterComponents"`   // theses image in master node and num is equal to master num
	SlaveComponents   Image         `bson:"slaveComponents" json:"slaveComponents"`     //theses image in slave node and num is equal to slave num
	OnlyOneComponents Image         `bson:"onlyoneComponents" json:"onlyoneComponents"` // theses image in master node and num is 1
	AllComponents     Image         `bson:"allComponents" json:"allComponents"`         // theses image in master and slave node and num is equal to all node num
	SwarmName         string        `bson:"swarmName" json:"swarmName"`
	ClientIp          string        `bson:"clientIp" json:"clientIp"`
	MonitorIp         string        `bson:"monitorIp" json:"monitorIp"`
	TimeCreate        time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate        time.Time     `bson:"time_update" json:"time_update"`
}

type Image struct {
	ImageName []string     `bson:"imageName" json:"imageName"`
	NodeInfo  []IpHostName `bson:"ipHostname" json:"ipHostname"`
}

type IpHostName struct {
	IP       string `bson:"ip" json:"ip"`
	HostName string `bson:"hostName" json:"hostName"`
}

type ComponentsInfo struct {
	UserName         string             `bson:"userName" json:"userName"`
	ClusterName      string             `bson:"clusterName" json:"clusterName"`
	ClusterId        string             `bson:"clusterId" json:"clusterId"`
	ComponentsStatus []ComponentsStatus `bson:"componentStatus" json:"componentStatus"`
}

type ComponentsStatus struct {
	ComponentName string `bson:"componentName" json:"componentName"`
	Ip            string `bson:"ip" json:"ip"`
	Status        string `bson:"status" json:"status"`
}
