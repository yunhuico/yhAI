package entity

type NotifyHost struct {
	ClusterName string   `bson:"clusterName" json:"clusterName"`
	IsSuccess   bool     `bson:"isSuccess" json:"isSuccess"`
	Servers     []Server `bson:"servers" json:"servers"`
	Operation   string   `bson:"operation" json:"operation"`
	UserName    string   `bson:"userName" json:"userName"`
}

type NotifyCluster struct {
	ClusterName string `bson:"clusterName" json:"clusterName"`
	IsSuccess   bool   `bson:"isSuccess" json:"isSuccess"`
	Operation   string `bson:"operation" json:"operation"`
	UserName    string `bson:"userName" json:"userName"`
	LogId       string `json:"logId" json:"logId"`
	Comments    string `json:"comments" json:"comments"`
}

type NotifyPubkey struct {
	ClusterName string   `bson:"clusterName" json:"clusterName"`
	UserName    string   `bson:"userName" json:"userName"`
	PubkeyIds   []string `bson:"pubkeyIds" json:"pubkeyIds"`
}
