package entity

import (
	"gopkg.in/mgo.v2/bson"
	//	entity2 "linkernetworks.com/linker_common_lib/entity"
	"time"
)

type ClusterNetwork struct {
	ObjectId        bson.ObjectId `bson:"_id" json:"_id"`
	ClusterId       string        `bson:"cluster_id" json:"cluster_id"`
	ClusterName     string        `bson:"cluster_name" json:"cluster_name"`
	UserName        string        `bson:"user_name" json:"user_name"`
	ClusterHostName string        `bson:"clust_host_name" json:"clust_host_name"`
	Network         Network       `bson:"network" json:"network"`
	NetworkId       string        `bson:"network_id" json:"network_id"`
	TimeCreate      time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate      time.Time     `bson:"time_update" json:"time_update"`
}

type Network struct {
	Name     string            `bson:"name" json:"name"`
	Subnet   []string          `bson:"subnet" json:"subnet"`
	Gateway  []string          `bson:"gateway" json:"gateway"`
	IPRange  []string          `bson:"ipRange" json:"ipRange"`
	Internal string            `bson:"internal" json:"internal"`
	Driver   string            `bson:"driver" json:"driver"`
	Options  map[string]string `bson:"options" json:"options"`
}
