package entity

import (
	"regexp"
	"strings"
	"time"

	marathon "github.com/LinkerNetworks/go-marathon"
	"gopkg.in/mgo.v2/bson"
)

// this object will be saved in db.
type Appset struct {
	ObjectId      bson.ObjectId  `json:"_id" bson:"_id"`
	Name          string         `json:"name" bson:"name"`
	Status        string         `json:"status" bson:"status"`
	Description   string         `json:"description" bson:"description"`
	TemplateId    string         `json:"template_id" bson:"template_id"`
	CreatedByJson bool           `json:"created_by_json" bson:"created_by_json"`
	Group         marathon.Group `json:"group" bson:"group"`
	TimeDeployed  time.Time      `json:"time_deployed" bson:"time_deployed"`
	TimeCreate    time.Time      `json:"time_create" bson:"time_create"`
	TimeUpdate    time.Time      `json:"time_update" bson:"time_update"`
}

// this object will not be saved, it will be used to return in list appsets to client.
type AppsetListlViewObj struct {
	ObjectId      bson.ObjectId `json:"_id" bson:"_id"`
	Name          string        `json:"name" bson:"name"`
	Status        string        `json:"status" bson:"status"`
	Description   string        `json:"description" bson:"description"`
	TemplateId    string        `json:"template_id" bson:"template_id"`
	CreatedByJson bool          `json:"created_by_json" bson:"created_by_json"`
	TimeDeployed  time.Time     `json:"time_deployed" bson:"time_deployed"`
	TimeCreate    time.Time     `json:"time_create" bson:"time_create"`
	TimeUpdate    time.Time     `json:"time_update" bson:"time_update"`
}

// this object will not be saved, it will be used to return to client.
type AppsetDetailViewObj struct {
	ObjectId         bson.ObjectId      `json:"_id" bson:"_id"`
	Name             string             `json:"name" bson:"name"`
	Status           string             `json:"status" bson:"status"`
	Description      string             `json:"description" bson:"description"`
	TemplateId       string             `json:"template_id" bson:"template_id"`
	TotalCpu         float64            `json:"total_cpu" bson:"total_cpu"`
	TotalMem         float64            `json:"total_mem" bson:"total_mem"`
	TotalGpu         float64            `json:"total_gpu" bosn:"total_gpu"`
	TotalContainer   int                `json:"total_container" bson:"total_container"`
	TotalHost        int                `json:"total_host" bson:"total_host"`
	RunningCpu       float64            `json:"running_cpu" bson:"running_cpu"`
	RunningMem       float64            `json:"running_mem" bson:"running_mem"`
	RunningGpu       float64            `json:"running_gpu" bson:"running_gpu"`
	RunningContainer int                `json:"running_container" bson:"running_container"`
	CreatedByJson    bool               `json:"created_by_json" bson:"created_by_json"`
	Group            marathon.Group     `json:"group" bson:"group"`
	Components       []ComponentViewObj `json:"components" bson:"components"`
	TimeDeployed     time.Time          `json:"time_deployed" bson:"time_deployed"`
	TimeCreate       time.Time          `json:"time_create" bson:"time_create"`
	TimeUpdate       time.Time          `json:"time_update" bson:"time_update"`
}

// this object will not be saved in db, it will be used to return to client.
type ComponentViewObj struct {
	AppsetName    string            `json:"appset_name" bson:"appset_name"`
	TotalCpu      float64           `json:"total_cpu" bson:"total_cpu"`
	TotalMem      float64           `json:"total_mem" bson:"total_mem"`
	TotalGpu      float64           `json:"total_gpu" bosn:"total_gpu"`
	Status        string            `json:"status" bson:"status"`
	CanDelete     bool              `json:"can_delete" bson:"can_delete"`
	CanStop       bool              `json:"can_stop" bson:"can_stop"`
	CanStart      bool              `json:"can_start" bson:"can_start"`
	CanModify     bool              `json:"can_modify" bson:"can_modify"`
	CanScale      bool              `json:"can_scale" bson:"can_scale"`
	CanShell      bool              `json:"can_shell" bson:"can_shell"`
	MonitorURLMap map[string]string `json:"monitorurl_map" bson:"monitorurl_map"`
	// ContainerInfos []ContainerViewObj   `json:"container_infos" bson:"container_infos"`
	App marathon.Application `json:"app,omitempty" bson:"app,omitempty"`
}

// this object will not be saved in db, it will be used to return to client.
type ContainerViewObj struct {
	marathon.Task
	Name string `json:"name" bson:"name"`
}

// type Deployment struct {
// 	Version      time.Time `json:"version"`
// 	DeploymentId string    `json:"deploymentId"`
// }

type Message struct {
	Message string      `json:"message"`
	Details interface{} `json:"details"`
}

// structure of docker config.json(default ~/.docker/config.json)
type DockerAuthConfig struct {
	Auths map[string]Auth `json:"auths"`
}

type Auth struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

// interface wrapper for string array
type RegistryList struct {
	RegistryList []string
}

func (c *ComponentViewObj) IsValid() bool {
	// appset_name can not be empty
	if len(strings.TrimSpace(c.AppsetName)) == 0 {
		return false
	}
	// app id can not be empty
	if len(strings.TrimSpace(c.App.ID)) == 0 {
		return false
	}
	if c.App.Instances == nil {
		return false
	}
	// app cpus and mem can not be zero
	if c.App.CPUs <= 0 || *c.App.Mem <= 0 {
		return false
	}
	return true
}

func (a *Appset) IsValid() bool {
	if len(strings.TrimSpace(a.Name)) == 0 {
		return false
	}
	reg := regexp.MustCompile("^(([a-z0-9]|[a-z0-9][a-z0-9\\-]*[a-z0-9]).)*([a-z0-9]|[a-z0-9][a-z0-9\\-]*[a-z0-9])$")
	return reg.MatchString(a.Name)
}

func (a *Appset) ToListView() AppsetListlViewObj {
	return AppsetListlViewObj{
		ObjectId:      a.ObjectId,
		Name:          a.Name,
		Status:        a.Status,
		Description:   a.Description,
		TemplateId:    a.TemplateId,
		CreatedByJson: a.CreatedByJson,
		TimeDeployed:  a.TimeDeployed, // need?
		TimeCreate:    a.TimeCreate,
		TimeUpdate:    a.TimeUpdate}
}

/**
 * loop all apps of an appset, and exec function f for each one
 */
func (a *Appset) OperateOnAllApps(f interface{}, args ...interface{}) {
	apps := a.GetAllApps()
	// logrus.WithFields(logrus.Fields{"appset": a, "f": f, "args": args}).Infof("get apps [%v]", apps)
	for _, eachapp := range apps {
		if len(args) > 1 {
			// logrus.WithFields(logrus.Fields{"app": eachapp.ID, "f": f}).Infof("call f(app, ...instance{})")
			f.(func(*marathon.Application, ...interface{}))(eachapp, args)
		} else if len(args) == 1 {
			// logrus.WithFields(logrus.Fields{"app": eachapp.ID, "f": f}).Infof("call f(app, instance{})")
			f.(func(*marathon.Application, interface{}))(eachapp, args[0])
		} else {
			// logrus.WithFields(logrus.Fields{"app": eachapp.ID, "f": f}).Infof("call f(app)")
			f.(func(*marathon.Application))(eachapp)
		}
	}
}

/**
 * convert group and app path to absolute path
 */
func (a *Appset) ConvertPath() {
	group := a.Group
	// convert root group first
	if !strings.HasPrefix(group.ID, "/") {
		group.ID = "/" + group.ID
	}
	a.convertPathInGroup(&group)
	a.Group = group
}

func (a *Appset) convertPathInGroup(group *marathon.Group) {
	// loop group.Groups, and append group.ID for each subGroup.ID
	for _, subgroup := range group.Groups {
		if !strings.HasPrefix(subgroup.ID, group.ID) {
			subgroup.ID = group.ID + "/" + subgroup.ID
		}
		a.convertPathInGroup(subgroup)
	}
	// loop group.Apps, and append parentId for each app.ID
	for _, app := range group.Apps {
		if !strings.HasPrefix(app.ID, group.ID) {
			app.ID = group.ID + "/" + app.ID
		}
	}
}

func (a *Appset) GetAllApps() []*marathon.Application {
	return getAppsFromGroup(&a.Group)
}

func getAppsFromGroup(gp *marathon.Group) (apps []*marathon.Application) {
	apps = []*marathon.Application{}
	if gp.Apps != nil {
		for _, app := range gp.Apps {
			apps = append(apps, app)
		}
	}
	if gp.Groups != nil {
		for _, group := range gp.Groups {
			newapps := getAppsFromGroup(group)
			for _, app := range newapps {
				apps = append(apps, app)
			}
		}
	}

	return
}

func (a *Appset) GetAllGroups() []*marathon.Group {
	return getGroupsFromGroup(&a.Group)
}

func getGroupsFromGroup(gp *marathon.Group) (groups []*marathon.Group) {
	groups = []*marathon.Group{}
	if gp.Groups != nil {
		for _, group := range gp.Groups {
			groups = append(groups, group)
			newgroups := getGroupsFromGroup(group)
			for _, newgp := range newgroups {
				groups = append(groups, newgp)
			}
		}
	}
	return
}

type BasicInfo struct {
	MgmtIp      []string `bson:"mgmtIp" json:"mgmtIp"`
	ClusterName string   `bson:"clusterName" json:"clusterName"`
	ClusterId   string   `bson:"clusterId" json:"clusterId"`
	UserName    string   `bson:"userName" json:"userName"`
	UserId      string   `bson:"user_id" json:"user_id"`
	TenantId    string   `bson:"tenant_id" json:"tenant_id"`
	MonitorIp string `bson:"monitorIp" json:"monitorIp"`
}

type SendHostAlertReq struct {
	UserId string `json:"user_id"`
	Subject string `json:"subject"`
	Content string `json:content`
}
