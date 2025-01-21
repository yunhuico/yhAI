package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type DeploymentInfo struct {
	ObjectId     bson.ObjectId `bson:"_id"`
	Operation    string        `bson:"operation"`
	GroupId      string        `bson:"group_id"`
	AppId        string        `bson:"app_id"`
	DeploymentId string        `bson:"deployment_id"`
	TimeCreate   time.Time     `bson:"time_create"`
}
