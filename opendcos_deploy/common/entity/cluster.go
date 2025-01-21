package entity

//parameters of config.yaml
type DCOSConfig struct {
	Agent_list                []string `yaml:"agent_list,omitempty" json:"agent_list"`
	Bootstrap_url             string   `yaml:"bootstrap_url,omitempty" json:"bootstrap_url"` //default: file:///opt/dcos_install_tmp
	Cluster_name              string   `yaml:"cluster_name" json:"cluster_name"`
	Exhibitor_storage_backend string   `yaml:"exhibitor_storage_backend,omitempty" json:"exhibitor_storage_backend"` //static/zookeeper/aws_s3/azure
	Master_discovery          string   `yaml:"master_discovery,omitempty" json:"master_discovery"`                   //static/master_http_loadbalancer
	Master_list               []string `yaml:"master_list,omitempty" json:"master_list"`
	Public_agent_list         []string `yaml:"public_agent_list,omitempty" json:"public_agent_list"`
	Dcos_overlay_enable       bool     `yaml:"dcos_overlay_enable,omitempty" json:"dcos_overlay_enable"`
	Dns_search                string   `yaml:"dns_search,omitempty" json:"dns_search"` //<dc1.example.com dc1.example.com example.com dc1.example.com dc2.example.com example.com>
	Resolvers                 []string `yaml:"resolvers,omitempty" json:"resolvers"`
	Use_proxy                 bool     `yaml:"use_proxy,omitempty" json:"use_proxy"`                     //DEFAULT 'false'
	Check_time                string   `yaml:"check_time,omitempty" json:"check_time"`                   //Check if NTP is enabled during startu
	Docker_remove_delay       int      `yaml:"docker_remove_delay,omitempty" json:"docker_remove_delay"` //DEFAULT 1 hour
	Gc_delay                  int      `yaml:"gc_delay,omitempty" json:"gc_delay"`                       //DEFAULT 2 days
	Log_directory             string   `yaml:"log_directory,omitempty" json:"log_directory"`             //DEFAULT /genconf/logs
	Process_timeout           int      `yaml:"process_timeout,omitempty" json:"process_timeout"`         //DEFAULT 120 seconds
	Oauth_enabled             bool     `yaml:"oauth_enabled,omitempty" json:"oauth_enabled"`             //DEFAULT 'true'
	Telemetry_enabled         bool     `yaml:"telemetry_enabled,omitempty" json:"telemetry_enabled"`     //Enable anonymous data sharing. DEFAULT 'true'
	Ssh_key_path              string   `yaml:"ssh_key_path,omitempty" json:"ssh_key_path"`               //DEFAULT /genconf/ssh-key
	Ssh_port                  int      `yaml:"ssh_port,omitempty" json:"ssh_port"`
	Ssh_user                  string   `yaml:"ssh_user,omitempty" json:"ssh_user"`

	//when exhibitor_storage_backend: zookeeper, required
	Exhibitor_zk_hosts string `yaml:"exhibitor_zk_hosts,omitempty" json:"exhibitor_zk_hosts"` //<ZK_IP>:<ZK_PORT>, <ZK_IP>:<ZK_PORT>, <ZK_IP>:<ZK_PORT>
	Exhibitor_zk_path  string `yaml:"exhibitor_zk_path,omitempty" json:"exhibitor_zk_path"`
	//when exhibitor_storage_backend: aws_s3, required
	Aws_access_key_id       string `yaml:"aws_access_key_id,omitempty" json:"aws_access_key_id"`             //<AWS key ID>
	Aws_region              string `yaml:"aws_region,omitempty" json:"aws_region"`                           //<AWS region for your S3 bucket>
	Aws_secret_access_key   string `yaml:"aws_secret_access_key,omitempty" json:"aws_secret_access_key"`     //<AWS secret access key>
	Exhibitor_explicit_keys bool   `yaml:"exhibitor_explicit_keys,omitempty" json:"exhibitor_explicit_keys"` //'true'/'false'
	S3_bucket               string `yaml:"s3_bucket,omitempty" json:"s3_bucket"`                             //name of your S3 bucket
	S3_prefix               string `yaml:"s3_prefix,omitempty" json:"s3_prefix"`                             //S3 prefix to be used within your S3 bucket to be used by Exhibitor
	//when exhibitor_storage_backend: azure, required
	Exhibitor_azure_account_name string `yaml:"exhibitor_azure_account_name,omitempty" json:"exhibitor_azure_account_name"` //<the Azure Storage Account Name>
	Exhibitor_azure_account_key  string `yaml:"exhibitor_azure_account_key,omitempty" json:"exhibitor_azure_account_key"`   //a secret key to access the Azure Storage Account
	Exhibitor_azure_prefix       string `yaml:"exhibitor_azure_prefix,omitempty" json:"exhibitor_azure_prefix"`             //the blob prefix to be used within your Storage Account to be used by Exhibitor

	//when master_discovery: master_http_loadbalancer, required
	Exhibitor_address string `yaml:"exhibitor_address,omitempty" json:"exhibitor_address"`
	Num_masters       int    `yaml:"num_masters,omitempty" json:"num_masters"`

	//when dcos_overlay_enable: 'true'
	Dcos_overlay_config_attempts int                `yaml:"dcos_overlay_config_attempts,omitempty" json:"dcos_overlay_config_attempts"` //how many failed configuration attempts are allowed
	Dcos_overlay_mtu             int                `yaml:"dcos_overlay_mtu,omitempty" json:"dcos_overlay_mtu"`                         //the maximum transmission unit (MTU) of the Virtual Ethernet (vEth) on the containers
	Dcos_overlay_network         DcosOverlayNetwork `yaml:"dcos_overlay_network,omitempty" json:"dcos_overlay_network"`

	//when use_proxy: 'true'
	Http_proxy  string   `yaml:"http_proxy,omitempty" json:"http_proxy"`
	Https_proxy string   `yaml:"https_proxy,omitempty" json:"https_proxy"`
	No_proxy    []string `yaml:"no_proxy,omitempty" json:"no_proxy"` // addresses to exclude from the proxy
}

type DcosOverlayNetwork struct {
	Vtep_subnet  string    `yaml:"vtep_subnet,omitempty" json:"vtep_subnet"`   //example: 44.128.0.0/20
	Vtep_mac_oui string    `yaml:"vtep_mac_oui,omitempty" json:"vtep_mac_oui"` //<MAC address> example:70:B3:D5:00:00:00
	Overlays     []Overlay `yaml:"overlays,omitempty" json:"overlays"`
}

type Overlay struct {
	Name   string `yaml:"name,omitempty" json:"name"`
	Subnet string `yaml:"subnet,omitempty" json:"subnet"` //example: 9.0.0.0/8
	Prefix int    `yaml:"prefix,omitempty" json:"prefix"` //the size of the subnet. example:26
}
