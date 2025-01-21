package entity

type CreateRequest struct {
	UserName       string     `json:"userName"`
	SkipInstall    bool       `json:"skipInstall"`
	PrivateKey     string     `json:"privateKey"`
	PrivateNicName string     `json:"privateNicName"`
	Config         DCOSConfig `json:"config"`
}

type AddNodeRequest struct {
	UserName       string `json:"userName"`
	ClusterName    string `json:"clusterName"`
	PrivateNicName string `json:"privateNicName"`
	SshUser        string `json:"sshUser"`
	Nodes          []Node `json:"nodes"`
}

type Node struct {
	Ip          string `json:"ip"`
	SkipInstall bool   `json:"skipInstall"`
	SlaveType   string `json:"slaveType"` //'slave' or 'slave_public'
}
