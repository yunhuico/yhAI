package constant

const (
	// AddrSeparator is separator of cAdvisor addresses
	// env CADVISORS looks like "192.168.1.100:8080;192.168.1.102:8081"
	AddrSeparator = ";"
)

const (
	// Example key-vaule map in metrics line
	// {
	//  alert="true",alert_name="HighCpuAlert",app_container_id="",
	//  app_id="/stress/stress",group_id="/stress",id="/docker/3b20e51c5060a62947fa9e1bb4ffb11bbe35ef6805378c1d819a6e64c9fac47f",
	//  image="zyfdedh/stress",mesos_task_id="stress_stress.a2f38000-710d-11e7-b4cc-2e73556a9d37",
	//  name="mesos-46d54b48-5de4-4e53-a2fa-9d4cfd478958-S0.239f4222-eab0-4503-b143-ae57136ef6c8",
	//  repair_template_id="stress",service_group_id="",service_group_instance_id="",service_order_id=""
	// }

	// KeyAppID is one of fields in metrics lines
	KeyAppID = "app_id"
	// KeyAlert is one of fields in metrics lines
	KeyAlert = "alert"
	// KeyAlertName is one of fields in metrics lines
	KeyAlertName = "alert_name"
	// KeyAppContainerID is one of fields in metrics lines
	KeyAppContainerID = "app_container_id"
	// KeyGroupID is one of fields in metrics lines
	KeyGroupID = "group_id"
	// KeyImage is one of fields in metrics lines
	KeyImage = "image"
	// KeyRepairTemplateID is one of fields in metrics lines
	KeyRepairTemplateID = "repair_template_id"
	// KeyServiceGroupID is one of fields in metrics lines
	KeyServiceGroupID = "service_group_id"
	// KeyServiceGroupInstanceID is one of fields in metrics lines
	KeyServiceGroupInstanceID = "service_group_instance_id"
	// KeyServiceOrderID is one of fields in metrics lines
	KeyServiceOrderID = "service_order_id"
	// KeyHostIP is one of fields in metrics lines
	KeyHostIP = "host_ip"
)

const (
	HighCpuAlert    = "HighCpuAlert"
	LowCpuAlert     = "LowCpuAlert"
	HighMemoryAlert = "HighMemoryAlert"
	LowMemoryAlert  = "LowMemoryAlert"
)
