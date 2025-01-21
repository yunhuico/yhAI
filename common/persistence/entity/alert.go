package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// AlertMessage is the raw request body from alertmanager
// {
//    "receiver":"linker-webhook",
//    "status":"firing",
//    "alerts":[
//       {
//          "status":"firing",
//          "labels":{
//             "__name__":"host_memory_usage",
//             "alert":"true",
//             "alert_name":"HostHighMemoryAlert",
//             "alertname":"HostHighMemAlert",
//             "host_ip":"192.168.3.53",
//             "instance":"master.mesos:10005",
//             "job":"prometheus",
//             "monitor":"linker-metrics"
//          },
//          "annotations":{
//             "description":"High memory usage for host machine on 192.168.3.53, (current value: 84.69264)",
//             "summary":"High memory usage alert for host machine"
//          },
//          "startsAt":"2017-10-31T11:11:37.806Z",
//          "endsAt":"0001-01-01T00:00:00Z",
//          "generatorURL":"http://q-731704137-2-sysadmin:9090/graph#%5B%7B%22expr%22%3A%22host_memory_usage%20%3E%2082%22%2C%22tab%22%3A0%7D%5D"
//       },
//       {
//          "status":"firing",
//          "labels":{
//             "__name__":"host_memory_usage",
//             "alert":"true",
//             "alert_name":"HostLowMemoryAlert",
//             "alertname":"HostHighMemAlert",
//             "host_ip":"192.168.3.53",
//             "instance":"master.mesos:10005",
//             "job":"prometheus",
//             "monitor":"linker-metrics"
//          },
//          "annotations":{
//             "description":"High memory usage for host machine on 192.168.3.53, (current value: 84.69264)",
//             "summary":"High memory usage alert for host machine"
//          },
//          "startsAt":"2017-10-31T11:11:37.806Z",
//          "endsAt":"0001-01-01T00:00:00Z",
//          "generatorURL":"http://q-731704137-2-sysadmin:9090/graph#%5B%7B%22expr%22%3A%22host_memory_usage%20%3E%2082%22%2C%22tab%22%3A0%7D%5D"
//       }
//    ],
//    "groupLabels":{
//       "alertname":"HostHighMemAlert"
//    },
//    "commonLabels":{
//       "__name__":"host_memory_usage",
//       "alert":"true",
//       "alertname":"HostHighMemAlert",
//       "host_ip":"192.168.3.53",
//       "instance":"master.mesos:10005",
//       "job":"prometheus",
//       "monitor":"linker-metrics"
//    },
//    "commonAnnotations":{
//       "description":"High memory usage for host machine on 192.168.3.53, (current value: 84.69264)",
//       "summary":"High memory usage alert for host machine"
//    },
//    "externalURL":"http://q-731704137-2-sysadmin:9093",
//    "version":"3",
//    "groupKey":9775564197830797045
// }
type AlertMessage struct {
	ObjectId   bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
	Receiver   string        `bson:"receiver" json:"receiver,omitempty"`
	Status     string        `bson:"status" json:"status,omitempty"`
	Alert      []Alert       `bson:"alerts" json:"alerts,omitempty"`
	Version    string        `bson:"version" json:"version,omitempty"`
	TimeCreate time.Time     `bson:"time_create" json:"time_create,omitempty"`
	TimeUpdate time.Time     `bson:"time_update" json:"time_update,omitempty"`
}

type Alert struct {
	ObjectId     bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
	Status       string        `bson:"status" json:"status"`
	Labels       AlertLabel    `bson:"labels" json:"labels"`
	Annotations  Annotations   `bson:"annotations" json:"annotations"`
	StartsAt     string        `bson:"starts_at" json:"startsAt"`
	EndsAt       string        `bson:"ends_at" json:"endsAt"`
	GeneratorURL string        `bson:"generator_url" json:"generatorURL"`
	TimeCreate   time.Time     `bson:"time_create" json:"time_create,omitempty"`
	TimeUpdate   time.Time     `bson:"time_update" json:"time_update,omitempty"`
}

// AlertResp is the response body of GET /v1/alerts
type AlertResp struct {
	ObjectId   bson.ObjectId `json:"_id"`
	AppID      string        `json:"app_id"`
	GroupID    string        `json:"group_id"`
	AlertName  string        `json:"alert_name"`
	Action     string        `json:"action"`
	TimeUpdate time.Time     `json:"time_update"`
}

type AlertLabel struct {
	// BuiltInAlertName is the original hidden label 'alertname' from alertmanager request body
	// This is same with alert name on Prometheus UI
	BuiltInAlertName string `bson:"alertname" json:"alertname"`
	// AlertName is a duplicated label defined in cadvisor metrics by LinkerNetworks
	AlertName              string `bson:"alert_name" json:"alert_name"`
	Image                  string `bson:"image" json:"image"`
	Name                   string `bson:"name" json:"name"`
	Id                     string `bson:"id"  json:"id"`
	ServiceGroupId         string `bson:"service_group_id"  json:"service_group_id"`
	ServiceGroupInstanceId string `bson:"service_group_instance_id"  json:"service_group_instance_id"`
	ServiceOrderId         string `bson:"service_order_id"  json:"service_order_id"`
	GroupId                string `bson:"group_id"  json:"group_id"`
	AppId                  string `bson:"app_id"  json:"app_id"`
	RepairTempalteId       string `bson:"repair_template_id"  json:"repair_template_id"`
	MesosTaskId            string `bson:"mesos_task_id" json:""`
	AppContainerId         string `bson:"app_container_id"  json:"app_container_id"`
	CpuUsage               string `bson:"cpu_usage"  json:"cpu_usage"`
	CpuUsageLowResult      string `bson:"cpu_usage_low_result"  json:"cpu_usage_low_result"`
	CpuUsageHighResult     string `bson:"cpu_usage_high_result"  json:"cpu_usage_high_result"`
	MemoryUsage            string `bson:"memory_usage"  json:"memory_usage"`
	MemoryUsageLowResult   string `bson:"memory_usage_low_result"  json:"memory_usage_low_result"`
	MemoryUsageHighResult  string `bson:"memory_usage_high_result"  json:"memory_usage_high_result"`
	Job                    string `bson:"job" json:"job"`
	Monitor                string `bson:"monitor" json:"monitor"`
	HostIP                 string `bson:"host_ip" json:"host_ip"`
}

type Annotations struct {
	Summary     string `bson:"summary" json:"summary"`
	Description string `bson:"description" json:"description"`
}

type GroupLabels struct {
	Alertname string `bson:"alertname" json:"alertname"`
}

type PayLoad struct {
	ActiveSince  string `bson:"activeSince" json:"activeSince"`
	AlertingRule string `bson:"alertingRule" json:"alertingRule"`
	GeneratorURL string `bson:"generatorURL" json:"generatorURL"`
	Value        string `bson:"value" json:"value"`
}
