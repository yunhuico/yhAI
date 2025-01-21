package services

import (
	"errors"
	// "fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/mgo.v2/bson"

	"strconv"
	"time"

	commandExec "linkernetworks.com/dcos-backend/common/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"

	"linkernetworks.com/dcos-backend/client/common"
	"linkernetworks.com/dcos-backend/common/httpclient"
)

var (
	frameworkService     *FrameworkService = nil
	onceFrameworkService sync.Once
)

const (
	//E72000~E72999  Framework
	TEMPLATE_ERR_QUERY      string = "E72000"
	TEMPLATE_ERR_NAME_EXIST string = "E72001"
	TEMPLATE_ERR_DELETE     string = "E72002"

	INSTANCE_ERR_QUERY  string = "E72100"
	INSTANCE_ERR_DELETE string = "E72101"
	INSTANCE_ERR_DEPLOY string = "E72102"

	INSTANCE_STATUS_IDLE      string = "IDLE"
	INSTANCE_STATUS_RUNNING   string = "RUNNING"
	INSTANCE_STATUS_FAILED    string = "FAILED"
	INSTANCE_STATUS_DEPLOYING string = "DEPLOYING"
	INSTANCE_STATUS_CREATED   string = "CREATED"
	INSTANCE_STATUS_DELETING  string = "DELETING"
)

type FrameworkService struct {
	collectionFrameworkTemplate string
	collectionFrameworkInstance string
}

func GetFrameworkService() *FrameworkService {
	onceFrameworkService.Do(func() {
		logrus.Debugf("Once called from frameworkService ......................................")
		frameworkService = &FrameworkService{
			collectionFrameworkTemplate: "frameworktemplate",
			collectionFrameworkInstance: "frameworkinstance",
		}

		frameworkService.initialize()
	})
	return frameworkService
}

func (p *FrameworkService) initialize() {
	logrus.Infof("initialize supported frameworks")

	frameworkList := []entity.FrameworkTemplate{}
	// framework:= entity.FrameworkTemplate{Name: "arangodb", Description: "A distributed free and open-source database with a flexible data model for documents, graphs, and key-values. Build high performance applications using a convenient SQL-like query language or JavaScript extensions.",
	//                                      LogoUrl: "https://raw.githubusercontent.com/arangodb/arangodb-dcos/master/icons/arangodb_medium.png"}
	// frameworkList = append(frameworkList, framework)

	framework := entity.FrameworkTemplate{Name: "cassandra", Description: "Apache Cassandra running on Apache Mesos",
		LogoUrl: "https://downloads.mesosphere.com/cassandra-mesos/assets/cassandra-medium.png"}
	frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "chronos", Description: "A fault tolerant job scheduler for Mesos which handles dependencies and ISO8601 based schedules.",
		LogoUrl: "https://downloads.mesosphere.com/universe/assets/icon-service-chronos-medium.png"}
	frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "hdfs", Description: "Hadoop Distributed File System (HDFS), Highly Available",
		LogoUrl: "https://downloads.mesosphere.com/hdfs/assets/icon-service-hdfs-medium.png"}
	frameworkList = append(frameworkList, framework)

	// framework= entity.FrameworkTemplate{Name: "jenkins", Description: "Jenkins is an award-winning, cross-platform, continuous integration and continuous delivery application that increases your productivity. Use Jenkins to build and test your software projects continuously making it easier for developers to integrate changes to the project, and making it easier for users to obtain a fresh build. It also allows you to continuously deliver your software by providing powerful ways to define your build pipelines and integrating with a large number of testing and deployment technologies.",
	//                                      LogoUrl: "https://downloads.mesosphere.com/jenkins/assets/icon-service-jenkins-medium.png"}
	// frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "kafka", Description: "Apache Kafka running on top of Apache Mesos",
		LogoUrl: "https://d1vubr0evspla.cloudfront.net/img/icon-medium.png"}
	frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "marathon", Description: "A cluster-wide init and control system for services in cgroups or Docker containers.",
		LogoUrl:   "https://downloads.mesosphere.com/marathon/assets/icon-service-marathon-medium.png",
		CanDeploy: false, CanUninstall: false, Status: INSTANCE_STATUS_CREATED}
	frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "spark", Description: "Spark is a fast and general cluster computing system for Big Data",
		LogoUrl: "https://downloads.mesosphere.io/spark/assets/icon-service-spark-medium.png"}
	frameworkList = append(frameworkList, framework)

	framework = entity.FrameworkTemplate{Name: "kubernetes", Description: "Manage an infrastructure cluster as a single system to simplify container operations.",
		LogoUrl: "https://downloads.mesosphere.com/kubernetes/artifacts/images/k8s-96x96.png"}
	frameworkList = append(frameworkList, framework)

	p.createTemplates(frameworkList)

}

func (p *FrameworkService) createTemplates(frameworkList []entity.FrameworkTemplate) {
	if len(frameworkList) <= 0 {
		logrus.Infof("no framework will be created!")
		return
	}

	for _, framework := range frameworkList {
		_, _, err := p.insertTemplate(framework)
		if err != nil {
			logrus.Warnf("create framework template error %v", err)
			continue
		}
	}
}

//insert FrameWorkTemplate
func (p *FrameworkService) insertTemplate(template entity.FrameworkTemplate) (newTemplate *entity.FrameworkTemplate,
	errorCode string, err error) {

	logrus.Debugf("insert framework template: %v", template)
	if len(strings.TrimSpace(template.Name)) == 0 {
		return nil, COMMON_ERROR_INVALIDATE, errors.New("template name can not be null")
	}

	//check if name exist
	conflict, err := p.isTemplateConflict(template.Name)
	if err != nil {
		return nil, COMMON_ERROR_INTERNAL, err
	}
	if conflict {
		return nil, TEMPLATE_ERR_NAME_EXIST, errors.New("template name already exists")
	}

	//save DB
	template.TimeCreate = dao.GetCurrentTime()
	template.TimeUpdate = template.TimeCreate
	template.ObjectId = bson.NewObjectId()

	err = dao.HandleInsert(p.collectionFrameworkTemplate, template)
	if err != nil {
		logrus.Errorf("insert FrameWorkTemplate to db failed, %v", err)
		return nil, COMMON_ERROR_INTERNAL, err
	}

	newTemplate = &template
	return
}

func (p *FrameworkService) ListTemplates(skip int, limit int, sort string) (total int,
	templates []entity.FrameworkTemplate, errorCode string, err error) {

	total, templates, errorCode, err = p.queryTemplates(bson.M{}, skip, limit, sort)

	frameworksIndocs, errs := p.getDcosInstalledFramework()
	if errs != nil {
		logrus.Warnf("get current framework error %v", errs)
	}

	_, midStatusInstances, _, err := p.queryInstances(bson.M{}, 0, 0, "")
	if err != nil {
		logrus.Warnf("get framework instance error %v", err)
	}

	for i := range templates {
		template := &templates[i]
		setTemplateStatus(template, frameworksIndocs, midStatusInstances)
	}

	return
}

func setTemplateStatus(template *entity.FrameworkTemplate, currentFrameworks []string, midInstances []entity.FrameworkInstance) {
	if strings.EqualFold(template.Name, "marathon") {
		return
	}

	name := template.Name
	if isInstalled(name, currentFrameworks) {
		if isDeleting(name, midInstances) {
			template.CanDeploy = false
			template.CanUninstall = false
			template.Status = INSTANCE_STATUS_DELETING
		} else {
			template.CanDeploy = false
			template.CanUninstall = true
			template.Status = INSTANCE_STATUS_CREATED
		}
	} else {
		if isDeploying(name, midInstances) {
			template.CanDeploy = false
			template.CanUninstall = false
			template.Status = INSTANCE_STATUS_DEPLOYING
		} else {
			template.CanDeploy = true
			template.CanUninstall = false
			template.Status = INSTANCE_STATUS_IDLE
		}
	}

	return
}

func isInstalled(name string, installList []string) bool {
	if len(installList) <= 0 {
		return false
	}

	for _, value := range installList {
		if strings.EqualFold(name, value) {
			return true
		}
	}

	return false
}

func isDeploying(name string, midInstances []entity.FrameworkInstance) bool {
	if len(midInstances) <= 0 {
		return false
	}

	for _, instance := range midInstances {
		if strings.EqualFold(name, instance.Name) && strings.EqualFold(INSTANCE_STATUS_DEPLOYING, instance.Status) {
			return true
		}
	}

	return false
}

func isDeleting(name string, midInstances []entity.FrameworkInstance) bool {
	if len(midInstances) <= 0 {
		return false
	}

	for _, instance := range midInstances {
		if strings.EqualFold(name, instance.Name) && strings.EqualFold(INSTANCE_STATUS_DELETING, instance.Status) {
			return true
		}
	}

	return false
}

func (p *FrameworkService) ListInstances(skip int, limit int, sort string) (total int,
	frameworks []entity.FrameWork, errorCode string, err error) {

	frameworksIndocs, errs := p.getDcosInstalledFramework()
	if errs != nil {
		logrus.Warnf("get current framework error %v", errs)
		return 0, nil, errorCode, errs
	}

	for i := range frameworksIndocs {
		frameworkName := frameworksIndocs[i]
		framework := entity.FrameWork{Name: frameworkName}
		frameworks = append(frameworks, framework)
	}

	return len(frameworks), frameworks, errorCode, err
}

func (p *FrameworkService) queryInstances(selector bson.M, skip int, limit int, sort string) (total int, instances []entity.FrameworkInstance,
	errorCode string, err error) {

	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionFrameworkInstance,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}

	instances = []entity.FrameworkInstance{}

	total, err = dao.HandleQueryAll(&instances, queryStruct)
	if err != nil {
		logrus.Errorf("query framework instances error is %v", err)
		errorCode = INSTANCE_ERR_QUERY
		return
	}
	return
}

func (p *FrameworkService) GetTemplateDetail(name string) (template *entity.FrameworkTemplate, errorCode string, err error) {
	logrus.Infof("query framework template with name: %s", name)
	return p.queryTemplateByName(name)
}

func (p *FrameworkService) DeleteInstance(name string) (errorCode string, err error) {
	logrus.Infof("delete framework instance with name: %s", name)

	instance := entity.FrameworkInstance{Name: name}
	instance.ObjectId = bson.NewObjectId()
	instance.TimeCreate = dao.GetCurrentTime()
	instance.TimeUpdate = instance.TimeCreate
	instance.Status = INSTANCE_STATUS_DELETING

	//save
	errorCode, err = p.saveInstance(instance)
	if err != nil {
		return
	}

	go p.goRemoveFramework(name)

	return
}

func (p *FrameworkService) CreateInstance(instance entity.FrameworkInstance) (newInstance *entity.FrameworkInstance,
	errorCode string, err error) {

	if len(strings.TrimSpace(instance.Name)) == 0 {
		return nil, COMMON_ERROR_INVALIDATE, errors.New("name not set")
	}
	if len(strings.TrimSpace(instance.TemplateName)) == 0 {
		return nil, COMMON_ERROR_INVALIDATE, errors.New("template_name not set")
	}

	_, errorCode, err = p.queryTemplateByName(instance.TemplateName)
	if err != nil {
		logrus.Errorf("get framework template by template name error %v", err)
		return
	}

	instance.ObjectId = bson.NewObjectId()
	instance.TimeCreate = dao.GetCurrentTime()
	instance.TimeUpdate = instance.TimeCreate
	instance.Status = INSTANCE_STATUS_DEPLOYING

	//save
	errorCode, err = p.saveInstance(instance)
	if err != nil {
		return
	}

	newInstance = &instance

	go p.goDeployFramework(instance.TemplateName)

	return
}

func (p *FrameworkService) saveInstance(instance entity.FrameworkInstance) (errorCode string, err error) {
	err = dao.HandleInsert(p.collectionFrameworkInstance, instance)
	if err != nil {
		logrus.Errorf("insert FrameworkInstance to db failed, %v", err)
		return COMMON_ERROR_INTERNAL, err
	}
	return
}

func (p *FrameworkService) isTemplateConflict(name string) (conflict bool, err error) {
	selector := bson.M{}
	selector["name"] = name

	n, _, _, err := p.queryTemplates(selector, 0, 0, "")
	if err != nil {
		logrus.Errorf("query db for framework templates failed, %v", err)
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	return false, nil
}

func (p *FrameworkService) queryTemplateByName(name string) (template *entity.FrameworkTemplate,
	errorCode string, err error) {

	selector := bson.M{}
	selector["name"] = name

	n, templates, _, err := p.queryTemplates(selector, 0, 0, "")
	if err != nil {
		logrus.Errorf("query framework template failed, %v", err)
		return nil, COMMON_ERROR_INTERNAL, errors.New("query framework template by name failed")
	}

	if n == 1 {
		template = &templates[0]
		return
	} else if n <= 0 {
		return nil, COMMON_ERROR_INTERNAL, errors.New("framework template not found")
	} else {
		return nil, COMMON_ERROR_INTERNAL, errors.New("multiple framework templates found")
	}
}

func (p *FrameworkService) deleteInstanceByName(name string) (errorCode string, err error) {
	logrus.Infof("start to delete framework instance with name [%s]", name)

	var selector = bson.M{}
	selector["name"] = name

	err = dao.HandleDelete(p.collectionFrameworkInstance, true, selector)
	if err != nil {
		errorCode = INSTANCE_ERR_DELETE
		logrus.Errorf("delete framework instance failed, name is [%v] , error is %v", name, err)
		return errorCode, err
	}
	return
}

func (p *FrameworkService) queryTemplates(selector bson.M, skip int, limit int, sort string) (total int, templates []entity.FrameworkTemplate,
	errorCode string, err error) {

	queryStruct := dao.QueryStruct{
		CollectionName: p.collectionFrameworkTemplate,
		Selector:       selector,
		Skip:           skip,
		Limit:          limit,
		Sort:           sort,
	}

	templates = []entity.FrameworkTemplate{}

	total, err = dao.HandleQueryAll(&templates, queryStruct)
	if err != nil {
		logrus.Errorf("query framework templates error is %v", err)
		errorCode = TEMPLATE_ERR_QUERY
		return
	}
	return
}

func (p *FrameworkService) ListFinishedTask(skip int, limit int, sort string, frameWorkName string, hostIP string) (total int,
	finishedTasks []entity.FinishedTask, errorCode string, err error) {

	endpoint, err := common.UTIL.LbClient.GetMesosEndpoint()
	if err != nil {
		return 0, nil, COMMON_ERROR_INTERNAL, err
	}

	return httpMesosTask(endpoint, skip, limit, sort, frameWorkName, hostIP)
}

func httpMesosTask(endpoint string, skip int, limit int, sort string, frameWorkName string, hostIP string) (total int,
	finishedTasks []entity.FinishedTask, errorCode string, err error) {

	urlMesosFrameWorks := endpoint + "/state"
	//	url := endpoint + "/tasks?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(skip) + "&sort=" + sort

	resp, err := httpclient.Http_get(urlMesosFrameWorks, "", httpclient.Header{"Content-Type", "application/json"})
	if err != nil {
		logrus.Errorf("http mesos to get state failed1, %v", err)
		return 0, nil, COMMON_ERROR_INTERNAL, err
	}
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		//decode
		MesosState := entity.MesosState{}
		MesosSlave := entity.MesosSlaves{}

		finishedTasks := []entity.FinishedTask{}

		err = getRetFromResponse(data, &MesosSlave)
		if err != nil {
			logrus.Errorf("http mesos to get slaves failed3, %v", err)
			return 0, nil, COMMON_ERROR_INTERNAL, err
		}
		err = getRetFromResponse(data, &MesosState)
		if err != nil {
			logrus.Errorf("http mesos to get state failed2, %v", err)
			return 0, nil, COMMON_ERROR_INTERNAL, err
		}

		//[]slaves info
		MapSlave := make(map[string]string)

		for _, slave := range MesosSlave.Slaves {
			MapSlave[slave.Id] = slave.HostName
		}

		logrus.Infof("mapslave:(%v+)", MapSlave)
		for _, frameWork := range MesosState.FrameWorks {
			if len(strings.TrimSpace(frameWorkName)) != 0 && frameWorkName != frameWork.Name {
				continue
			}
			for _, mesosTask := range frameWork.Tasks {
				if mesosTask.State != "TASK_RUNNING" {
					continue
				}
				for i, status := range mesosTask.Statuses {

					if i == 0 {
						if len(strings.TrimSpace(hostIP)) != 0 && hostIP != MapSlave[mesosTask.SlaveId] {
							continue
						}

						finishedTask := entity.FinishedTask{}
						finishedTask.Name = mesosTask.Name
						finishedTask.Status = mesosTask.State
						finishedTask.TaskId = mesosTask.Id
						//			finishedTask.TimeCreate =
						//			finishedTask.TimeUpdate =
						timestartstring := strings.Split(strconv.FormatFloat(status.Timestamp, 'f', 5, 64), ".")[0]
						timestartInt, _ := strconv.ParseInt(timestartstring, 10, 64)
						finishedTask.TimeStart = time.Unix(timestartInt, 0).Format("2006-01-02 03:04:05")

						finishedTask.Host = MapSlave[mesosTask.SlaveId]
						finishedTask.TemplateName = frameWork.Name

						//						finishedTask.TimeFinish =
						//				finishedTask.DurationInSeconds =
						finishedTasks = append(finishedTasks, finishedTask)
					}
				}
			}
		}
		if total = len(finishedTasks); total > 0 {

			if skip >= total {
				return total, []entity.FinishedTask{}, COMMON_ERROR_INTERNAL, err
			}

			if (skip + limit) >= len(finishedTasks) {
				return total, finishedTasks[skip:], COMMON_ERROR_INTERNAL, err
			}

			return total, finishedTasks[skip:(skip + limit)], COMMON_ERROR_INTERNAL, err
		}

		return 0, finishedTasks, "", err

	}
	return 0, nil, COMMON_ERROR_INTERNAL, errors.New("http mesos to get tasks failed!")
}

func (p *FrameworkService) getDcosInstalledFramework() (ret []string, err error) {
	logrus.Infof("get dcos installed framework")

	// cmd := exec.Command("dcos", "package", "list")
	cmd := "dcos package list | awk '{print $1 \"\t\" $3}'"
	output, _, errc := commandExec.ExecCommand(cmd)
	if errc != nil {
		logrus.Errorf("dcos package list error. %v", errc)
	} else {
		logrus.Infof("dcos package list success, output is %s ", output)

		result := strings.Split(output, "\n")
		for i := 1; i < len(result); i++ { //skip first value, as first value is title
			value := getDeployedFramework(strings.TrimSpace(result[i]))
			if len(value) > 0 {
				ret = append(ret, value)
			}

		}

		ret = append(ret, "marathon")
	}

	logrus.Debugf("current framework list in dcos is: %s", ret)
	return
}

//check wether the value is a deployed framework such as :"cassandra       ---"
func getDeployedFramework(value string) (framework string) {
	if len(value) <= 0 {
		return
	}

	result := strings.Fields(value)
	if len(result) != 2 {
		logrus.Warnf("failed to get installed framework for %s", value)
		return
	}

	appfield := result[1]
	if strings.EqualFold(appfield, "---") {
		return
	} else {
		framework = result[0]
		return
	}
}

////deployment

func (p *FrameworkService) goDeployFramework(name string) {

	cmd := exec.Command("dcos", "package", "install", "--yes", name)
	logrus.Infof("start install package on dcos, package is %s:", name)
	err := cmd.Run()
	if err != nil {
		logrus.Warnf("dcos install error. %v", err)
	} else {
		logrus.Info("dcos package install  success !")
	}

	logrus.Infof("remove the framework instance %s ", name)
	p.deleteInstanceByName(name)

	return
}

func (p *FrameworkService) goRemoveFramework(name string) {

	cmd := exec.Command("dcos", "package", "uninstall", name)
	logrus.Infof("start uninstall %s ", name)
	err := cmd.Run()
	if err != nil {
		logrus.Warnf("dcos uninstall fail, %v", err)
	} else {

		logrus.Info("dcos package uninstall successfully. Note to clear cache in Zookeeper!")
	}

	logrus.Infof("remove the framework instance %s ", name)
	p.deleteInstanceByName(name)
	return
}

func configDcosCli(configName string, configValue string) (err error) {
	cmd := exec.Command("dcos", "config", "set", configName, configValue)
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("dcos config set %v to %v failed, %v", configName, configValue, err)
		return err
	}
	return err
}

//config core.dcos_url for mesosphere dcos client
//example mesosMasterUrl: "http://192.168.10.130"
func configMasterUrl(mesosMasterUrl string) (err error) {
	return configDcosCli("core.dcos_url", mesosMasterUrl)
}

////callback
