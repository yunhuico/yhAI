package entity

type Monitor struct {
}

//simplified task,partial of marathon `task`
//wanted elements from `task` defined by marathon API v2
type Task struct {
	Id      string `json:"id"`
	AppId   string `json:"appId"`
	Host    string `json:"host"`
	SlaveId string `json:"slaveId"`
	Ports   []int  `json:"ports"`
	//...
}

//marathon returned response
type MarathonResp struct {
	Tasks []Task `json:"tasks"`
}

type ContainerInfo struct {
	ContainerName string `json:"containername"`
	Host          string `json:"host"`
	AppId         string `json:"appid"`
	SlaveId       string `json:"slaveid"`
}
