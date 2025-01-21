package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type FrameworkTemplate struct {
	ObjectId     bson.ObjectId `json:"_id" bson:"_id"`
	Name         string        `json:"name" bson:"name"`
	Description  string        `json:"description" bson:"description"`
	LogoUrl      string        `json:"logo_url" bson:"logo_url"`
	OfficialSite string        `json:"official_site" bson:"official_site"`
	CanDeploy    bool          `json:"can_deploy" bson:"can_deploy"`
	CanUninstall bool          `json:"can_uninstall" bson:"can_uninstall"`
	Status       string        `json:"status" bson:"status"`
	TimeCreate   time.Time     `json:"time_create" bson:"time_create"`
	TimeUpdate   time.Time     `json:"time_update" bson:"time_update"`
}

type FrameworkInstance struct {
	ObjectId     bson.ObjectId `json:"_id" bson:"_id"`
	TemplateName string        `json:"template_name" bson:"template_name"`
	Name         string        `json:"name" bson:"name"`
	Status       string        `json:"status" bson:"status"`
	Endpoint     string        `json:"endpoint" bson:"endpoint"`
	TaskCount    int           `json:"task_count" bson:"task_count"`
	Cpu          float32       `json:"cpu" bson:"cpu"`
	Mem          int           `json:"mem" bson:"mem"`
	TimeStart    time.Time     `json:"time_start" bson:"time_start"`
	CanDelete    bool          `json:"can_delete" bson:"can_delete"`
	FinishedIds  []string      `json:"finished_task_ids" bson:"finished_task_ids"`
	TimeCreate   time.Time     `json:"time_create" bson:"time_create"`
	TimeUpdate   time.Time     `json:"time_update" bson:"time_update"`
}

type FinishedTask struct {
	ObjectId          bson.ObjectId `json:"_id" bson:"_id"`
	TemplateName      string        `json:"template_name" bson:"template_name"`
	TaskId            string        `json:"task_id" bson:"task_id"`
	Name              string        `json:"name" bson:"name"`
	Host              string        `json:"host" bson:"host"`
	Status            string        `json:"status" bson:"status"`
	TimeStart         string        `json:"time_start" bson:"time_start"`
	TimeFinish        string        `json:"time_finish" bson:"time_finish"`
	DurationInSeconds int64         `json:"duration_in_seconds" bson:"duration_in_seconds"`
	TimeCreate        time.Time     `json:"time_create" bson:"time_create"`
	TimeUpdate        time.Time     `json:"time_update" bson:"time_update"`
}

type MesosFramework struct {
	Frameworks []FrameWork `json:"frameworks"`
}

type NetworkInfo struct {
	IpAddress string `json:"ip_address"`
}
type ContainerStatus struct {
	Network_infos []NetworkInfo `json:"network_infos"`
}
type Status struct {
	Timestamp       float64         `json:"timestamp"`
	ContainerStatus ContainerStatus `json:"container_status"`
}
type MesosTask struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	State    string   `json:"state"`
	Statuses []Status `json:"statuses"`
	SlaveId  string   `json:"slave_id"`
}

//type MesosCompleteTask struct {
//	Id       string   `json:"id"`
//	Name     string   `json:"name"`
//	State    string   `json:"state"`
//	Statuses []Status `json:"statuses"`
//}
type FrameWork struct {
	Id    string      `json:"id"`
	Name  string      `json:"name"`
	Tasks []MesosTask `json:"tasks"`
	//	CompleteTasks []MesosCompleteTask `json:"completed_tasks"`
}

type MesosState struct {
	FrameWorks []FrameWork `json:"frameworks"`
}
type MesosSlave struct {
	Id       string `json:"id"`
	HostName string `json:"hostname"`
}
type MesosSlaves struct {
	Slaves []MesosSlave `json:"slaves"`
}
