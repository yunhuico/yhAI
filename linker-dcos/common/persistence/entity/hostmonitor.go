package entity

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// ReqPutRules is the request body of PUT /v1/hostrules
type ReqPutRules struct {
	CPUEnabled bool `json:"cpu_enabled"`
	MemEnabled bool `json:"mem_enabled"`
	// Duration: unix time duration like "30s", "5m"
	Duration string `json:"duration"`
	// Thresholds: cpu_high, cpu_low, mem_high, mem_low
	Thresholds map[string]float32 `json:"thresholds"`
}

// HostRules is the structure of rules stored in database
type HostRules struct {
	ReqPutRules
	ObjectId   bson.ObjectId `bson:"_id" json:"_id"`
	TimeCreate time.Time     `bson:"time_create" json:"time_create"`
	TimeUpdate time.Time     `bson:"time_update" json:"time_update"`
}
