package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type DockerRegistry struct {
	ObjectId         bson.ObjectId `bson:"_id" json:"_id,omitempty"`
	Name             string        `bson:"name" json:"name"`
	Registry         string        `bson:"registry" json:"registry"`
	Secure           bool          `bson:"secure" json:"secure"`
	CAText           string        `bson:"ca_text" json:"ca_text"`
	Username         string        `bson:"username" json:"username"`
	Password         string        `bson:"password" json:"password"`
	UserId           string        `bson:"user_id" json:"user_id"`
	TenantId         string        `bson:"tenant_id" json:"tenant_id"`
	IsUse            bool          `bson:"isUse" json:"isUse"`
	IsSystemRegistry bool          `bson:"isSystemRegistry" json:"isSystemRegistry"`
	TimeCreate       time.Time     `bson:"time_create"`
}

//type DockerRegistryInfo struct {
//	DockerRegistryId string    `bson:"dockerRegistryId" json:"dockerRegistryId"`
//	Name             string    `bson:"name" json:"name"`
//	Registry         string    `bson:"registry" json:"registry"`
//	Secure           bool      `bson:"secure" json:"secure"`
//	CAText           string    `bson:"ca_text" json:"ca_text"`
//	Username         string    `bson:"username" json:"username"`
//	Password         string    `bson:"password" json:"password"`
//	UserId           string    `bson:"user_id" json:"user_id"`
//	TenantId         string    `bson:"tenant_id" json:"tenant_id"`
//	IsUse            bool      `bson:"isUse" json:"isUse"`
//	TimeCreate       time.Time `bson:"time_create"`
//}

//type DepDockerRegistry struct {
//	Name     string `json:"name"`
//	Registry string `json:"registry"`
//	Secure   bool   `json:"secure"`
//	CAText   string `json:"ca_text"`
//	Username string `json:"username"`
//	Password string `json:"password"`

//}
