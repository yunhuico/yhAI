package entity

import (
	"gopkg.in/mgo.v2/bson"
)

type RepairRecord struct {
	ObjectId               bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
	RepairId               string        `bson:"repair_id" json:"repair_id"`
	ServiceGroupInstanceId string        `bson:"service_group_instance_id" json:"service_group_instance_id"`
	ServiceGroupId         string        `bson:"service_group_id" json:"service_group_id"`
	AppCointainerId        string        `bson:"app_container_id" json:"app_container_id"`
	DeploymentId           string        `bson:"deployment_id" json:"deployment_id"`
	AlertId                string        `bson:"alert_id" json:"alert_id"`
	TimeCreate             string        `bson:"time_create" json:"time_create"`
	TimeUpdate             string        `bson:"time_update" json:"time_update"`
	AlertName              string        `bson:"alert_name" json:"alert_name"`
	Action                 string        `bson:"action" json:"action"`
	Status                 string        `bson:"status" json:"status"`
}

type RepairPolicy struct {
	ObjectId         bson.ObjectId `bson:"_id,omitempty" json:"_id,omitempty"`
	RepairTemplateId string        `bson:"repair_template_id" json:"repair_template_id"`
	ServiceGroupId   string        `bson:"service_group_id" json:"service_group_id"`
	AppCointainerId  string        `bson:"app_container_id" json:"app_container_id"`
	TimeCreate       string        `bson:"time_create" json:"time_create"`
	TimeUpdate       string        `bson:"time_update" json:"time_update"`
	User_id          string        `bson:"user_id" json:"user_id"`
	Tenant_id        string        `bson:"tenant_id" json:"tenant_id"`
	Polices          []Policy      `bson:"polices" json:"polices"`
	InstanceMaxNum string   `bson:"intance_max_number" json:"instance_max_number"`
	InstanceMinNum string    `bson:"instance_min_number" json:"instance_min_number`
}

type Policy struct {
	Conditions    []RepairCondition    `bson:"conditions" json:"conditions"`
	Actions       []RepairAction       `bson:"actions" json:"actions"`
	Notifications []RepairNotification `bson:"notifications" json:"notifications"`
}

type RepairCondition struct {
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
}

type RepairAction struct {
	Type           string            `bson:"type" json:"type"`
	AppContainerId string            `bson:"app_container_id" json:"app_container_id"`
	Parameters     []RepairParameter `bson:"parameter" json:"parameter"`
}

type RepairParameter struct {
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
}

type RepairNotification struct {
	Type     string `bson:"type" json:"type"`
	Receiver string `bson:"receiver" json:"receiver"`
}
