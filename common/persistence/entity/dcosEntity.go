package entity

type Server struct {
	Hostname         string `json:"hostName"`
	IpAddress        string `json:"ipAddress"`
	SshUser          string `json:"sshuser"`
	PrivateIpAddress string `json:"privateIpAddress"`
	IsMaster         bool   `json:"isMaster"`
	IsSlave          bool   `json:"isSlave"`
	IsFullfilled     bool   `json:"isFullfilled"`
	IsSharedServer   bool   `json:"isSharedServer"`
	IsMonitorServer  bool   `json:"isMonitorServer"`
	IsClientServer   bool   `json:"isClientServer"`
}

type Request struct {
	UserName    string       `json:"userName"`
	ClusterName string       `json:"clusterName"`
	PubKey      []PubkeyInfo `json:"pubkey"`
	//	ClusterNumber    int              `json:"clusterNumber"`
	MasterCount    int    `json:"masterCount"`
	SharedCount    int    `json:"sharedCount"`
	PureSlaveCount int    `json:"pureslaveCount"`
	NodeAttribute  string `json:"nodeAttribute"`
	IsLinkerMgmt   bool   `json:"isLinkerMgmt"`
	CreateMode     string `json:"createMode"`     //reuse or new
	CreateCategory string `json:"createCategory"` //compact or ha
	//	CreateNodes      NodesInfo        `json:"createNodes"`
	MasterNodes      []Node           `json:"masterNodes"`
	SharedNodes      []Node           `json:"sharedNodes"`
	PureSlaveNodes   []Node           `json:"pureslaveNodes"`
	ProviderInfo     ProviderInfo     `json:"providerInfo"`
	XAuthToken       string           `json:"x_auth_token"`
	ClusterId        string           `json:"clusterId"`
	LogId            string           `json:"logId"`
	UserId           string           `bson:"user_id" json:"user_id"`
	TenantId         string           `bson:"tenant_id" json:"tenant_id"`
	DockerRegistries []DockerRegistry `json:"dockerRegistries"`
	EngineOpts       []EngineOpt      `json:"engineOpts"`
}

func GetRequestNodes(c Request) (nodes []Node) {
	if len(c.MasterNodes) > 0 {
		for _, masternode := range c.MasterNodes {
			nodes = append(nodes, masternode)
		}
	}
	if len(c.SharedNodes) > 0 {
		for _, sharednode := range c.SharedNodes {
			nodes = append(nodes, sharednode)
		}
	}
	if len(c.PureSlaveNodes) > 0 {
		for _, pureslavenode := range c.PureSlaveNodes {
			nodes = append(nodes, pureslavenode)
		}
	}
	return
}

type AddNodeRequest struct {
	UserName    string `json:"userName"`
	ClusterName string `json:"clusterName"`
	//	AddNumber             int              `json:"addNumber"`
	SharedCount    int          `json:"sharedCount"`
	PureSlaveCount int          `json:"pureslaveCount"`
	PubKey         []PubkeyInfo `json:"pubkey"`
	NodeAttribute  string       `json:"nodeAttribute"`
	ExistedNumber  int          `json:"existedNumber"`
	AddMode        string       `json:"addMode"` //reuse or new
	//	AddNodes              NodesInfo        `json:"addNodes"`
	SharedNodes           []Node           `json:"sharedNodes"`
	PureSlaveNodes        []Node           `json:"pureslaveNodes"`
	ProviderInfo          ProviderInfo     `json:"providerInfo"`
	DnsServers            []Server         `json:"dnsServers"`
	SwarmMaster           string           `json:"swarmMaster"`
	XAuthToken            string           `json:"x_auth_token"`
	LogId                 string           `json:"logId"`
	MonitorServerHostName string           `json:"monitorServerHostName"`
	DockerRegistries      []DockerRegistry `json:"dockerRegistries"`
	EngineOpts            []EngineOpt      `json:"engineOpts"`
}

func GetAddrequestNodes(a AddNodeRequest) (nodes []Node) {
	if len(a.SharedNodes) > 0 {
		for _, sharednode := range a.SharedNodes {
			nodes = append(nodes, sharednode)
		}
	}
	if len(a.PureSlaveNodes) > 0 {
		for _, pureslavenode := range a.PureSlaveNodes {
			nodes = append(nodes, pureslavenode)
		}
	}
	return
}

type DeleteRequest struct {
	UserName    string   `json:"userName"`
	ClusterName string   `json:"clusterName"`
	Servers     []Server `json:"servers"`
	NowShared   int      `json:"nowShared"`
	XAuthToken  string   `json:"x_auth_token"`
	LogId       string   `json:"logId"`
	DnsServers  []Server `json:"dnsServers"`
}

type DeleteAllRequest struct {
	UserName      string `json:"userName"`
	ClusterName   string `json:"clusterName"`
	XAuthToken    string `json:"x_auth_token"`
	LogId         string `json:"logId"`
	ClusterId     string `json:"clusterId"`
	ClusterMgmtIp string `json:"mgmtId"`
}

type AddPubkeysRequest struct {
	UserName    string         `json:"userName"`
	ClusterName string         `json:"clusterName"`
	XAuthToken  string         `json:"x_auth_token"`
	Pubkey      []PubkeyInfo   `json:"pubkeyValue"`
	Hosts       []HostsPubInfo `json:"hosts"`
}

type PubkeyInfo struct {
	Id    string `json:"id"`
	Value string `json:"value"`
	Name  string `json:"name"`
}

type HostsPubInfo struct {
	HostName string `json:"hostName"`
	SshUser  string `json:"sshUser"`
	IP       string `json:"IP"`
}

type DeletePubkeysRequest struct {
	Pubkey      []PubkeyInfo   `json:"pubkeyValue"`
	Hosts       []HostsPubInfo `json:"hosts"`
	UserName    string         `json:"userName"`
	ClusterName string         `json:"clusterName"`
}

type AddRegistryRequest struct {
	UserName    string           `json:"userName"`
	ClusterName string           `json:"clusterName"`
	XAuthToken  string           `json:"x_auth_token"`
	Registrys   []DockerRegistry `json:"dockerRegistry"`
	Hosts       []HostsPubInfo   `json:"hosts"`
}

type DeleteRegistryRequest struct {
	UserName    string           `json:"userName"`
	ClusterName string           `json:"clusterName"`
	XAuthToken  string           `json:"x_auth_token"`
	Registrys   []DockerRegistry `json:"dockerRegistry"`
	Hosts       []HostsPubInfo   `json:"hosts"`
}

type DnsConfig struct {
	Zookeeper      string   `json:"zk"`
	Masters        []string `json:"masters"`
	RefreshSeconds int      `json:"refreshSeconds"`
	TimeToLive     int      `json:"ttl"`
	Domain         string   `json:"domain"`
	Port           int      `json:"port"`
	Resolvers      []string `json:"resolvers"`
	Timeout        int      `json:"timeout"`
	HTTPon         bool     `json:"httpon"`
	DNSon          bool     `json:"dnson"`
	HttpPort       int      `json:"httpport"`
	ExternalOn     bool     `json:"externalon"`
	Listener       string   `json:"listener"`
	SOAMname       string   `json:"SOAMname"`
	SOARname       string   `json:"SOARname"`
	SOARefresh     int      `json:"SOARefresh"`
	SOARetry       int      `json:"SOARetry"`
	SOAExpire      int      `json:"SOAExpire"`
	SOAMinttl      int      `json:"SOAMinttl"`
	IPSources      []string `json:"IPSources"`
}

type Openstack struct {
	AuthUrl       string `bson:"openstack-auth-url" json:"openstack-auth-url"`
	Username      string `bson:"openstack-username" json:"openstack-username"`
	Password      string `bson:"openstack-password" json:"openstack-password"`
	TenantName    string `bson:"openstack-tenant-name" json:"openstack-tenant-name"`
	FlavorName    string `bson:"openstack-flavor-name" json:"openstack-flavor-name"`
	ImageName     string `bson:"openstack-image-name" json:"openstack-image-name"`
	SshUser       string `bson:"openstack-ssh-user" json:"openstack-ssh-user"`
	SecurityGroup string `bson:"openstack-sec-groups" json:"openstack-sec-groups"`
	IpPoolName    string `bson:"openstack-floatingip-pool" json:"openstack-floatingip-pool"`
	NovaNetwork   string `bson:"openstack-nova-network" json:"openstack-nova-network"`
}

type Google struct {
	Project       string `bson:"google-project" json:"google-project"`
	Zone          string `bson:"google-zone" json:"google-zone"`
	MachineType   string `bson:"google-machine-type" json:"google-machine-type"`
	MachineImage  string `bson:"google-machine-image" json:"google-machine-image"`
	Network       string `bson:"google-network" json:"google-network"`
	SshUser       string `bson:"google-username" json:"google-username"`
	DiskSize      string `bson:"google-disk-size" json:"google-disk-size"`
	DiskType      string `bson:"google-disk-type" json:"google-disk-type"`
	UseInternalIP string `bson:"google-use-internal-ip" json:"google-use-internal-ip"`
	Tags          string `bson:"google-tags,omitempty" json:"google-tags,omitempty"`
	Credentials   string `bson:"google-application-credentials" json:"google-application-credentials"`
}

type AwsEC2 struct {
	AccessKey    string `bson:"amazonec2-access-key" json:"amazonec2-access-key"`
	SecretKey    string `bson:"amazonec2-secret-key" json:"amazonec2-secret-key"`
	ImageId      string `bson:"amazonec2-ami" json:"amazonec2-ami"`
	InstanceType string `bson:"amazonec2-instance-type" json:"amazonec2-instance-type"`
	RootSize     string `bson:"amazonec2-root-size" json:"amazonec2-root-size"`
	Region       string `bson:"amazonec2-region" json:"amazonec2-region"`
	VpcId        string `bson:"amazonec2-vpc-id" json:"amazonec2-vpc-id"`
	SshUser      string `bson:"amazonec2-ssh-user" json:"amazonec2-ssh-user"`
}

//type NodesInfo struct {
//	Nodes          []Node `json:"nodes"`
//	PrivateKey     string `json:"privateKey"`
//	PrivateNicName string `json:"privateNicName"` //private nic name
//}

type Node struct {
	IP       string `json:"ip"`
	SshUser  string `json:"sshUser"`
	Port     string `json:"port"`
	HostName string `json:"hostname"`
	Password string `json:"password"`
	//	PrivateKeyPath string `json:"privateKeyPath"`
	PrivateKey     string `json:"privateKey"`
	PrivateNicName string `json:"privateNicName"`
}

type ProviderInfo struct {
	Provider   Provider          `json:"provider"`
	Properties map[string]string `json:"properties"`
}

type Provider struct {
	ProviderType string `json:"providerType"`
	SshUser      string `json:"sshUser"`
}

type NodesCheck struct {
	Nodename string `json:"nodeName"`
	Errormsg string `json:"errorMsg"`
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type MgmtIps struct {
	MgmtIps []string `bson:"mgmtIps" json:"mgmtIps"`
}
