package services

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/compose/mejson"
	"gopkg.in/mgo.v2/bson"
	"linkernetworks.com/dcos-backend/cluster/common"
	"linkernetworks.com/dcos-backend/common/persistence/dao"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

var authService *AuthService = nil
var onceAuth sync.Once

type AuthService struct {
}

func GetAuthService() *AuthService {
	onceAuth.Do(func() {
		logrus.Debugf("Once called from authService ......................................")
		authService = new(AuthService)
		authService.InitializeAuth()
	})
	return authService
}

var AUTHDATA = make(map[string]interface{})

type AuthParam struct {
	auth_token          *entity.Token
	auth_instanceId     string
	auth_collectionName string
}

func (p *AuthService) InitializeAuth() (ok bool) {
	logrus.Infoln("begin to initialize auth service's auth file...")

	originData := map[string]string{}
	authFile := common.UTIL.Props.GetString("policy_file_path", "cluster_mgmt_policy.json")
	if len(authFile) <= 0 {
		authFile = "cluster_mgmt_policy.json"
	}
	logrus.Infof("using auth file %v", authFile)

	bytes, err := ioutil.ReadFile(authFile)
	if err != nil {
		logrus.Infoln("read controller's auth file error, %v", err)
		return false
	}
	// logrus.Debugf("auth file content is %v", string(bytes))

	if err := json.Unmarshal(bytes, &originData); err != nil {
		logrus.Infoln("unmarshal controller's auth file error, %v", err)
		return false
	}

	for k, v := range originData {
		if strings.HasPrefix(k, "model:") {
			k = strings.TrimPrefix(k, "model:")
			orcheck, andcheck, basecheck, pos := p.buildCheckObject(v, originData)
			if pos == 1 {
				AUTHDATA[k] = orcheck
			}
			if pos == 2 {
				AUTHDATA[k] = andcheck
			}
			if pos == 3 {
				AUTHDATA[k] = basecheck
			}

		}

	}

	logrus.Infoln("initialize cluster's auth file complete")
	return true
}

func (p *AuthService) buildCheckObject(value string, originMap map[string]string) (orcheck entity.OrCheck, andcheck entity.AndCheck, basecheck entity.BaseCheck, pos int) {
	if len(value) == 0 {
		pos = -1
		return
	}

	//it's an "or" policy
	if strings.Contains(value, " or ") {
		orcheck = entity.OrCheck{}
		orcheck.Basechecks = []entity.BaseCheck{}
		orcheck.Andchecks = []entity.AndCheck{}
		orcheck.Orchecks = []entity.OrCheck{}

		valueArrays := strings.Split(value, " or ")
		for i := 0; i < len(valueArrays); i++ {
			onePolicy := valueArrays[i]

			anotherOrcheck, anotherAndcheck, anotherBasecheck,
				anotherPos := p.buildCheckObject(onePolicy, originMap)
			if anotherPos == 1 {
				orcheck.Orchecks = append(orcheck.Orchecks, anotherOrcheck)
			} else if anotherPos == 2 {
				orcheck.Andchecks = append(orcheck.Andchecks, anotherAndcheck)
			} else if anotherPos == 3 {
				orcheck.Basechecks = append(orcheck.Basechecks, anotherBasecheck)
			}

		}
		pos = 1

	} else if strings.Contains(value, " and ") {
		andcheck = entity.AndCheck{}
		andcheck.Basechecks = []entity.BaseCheck{}
		andcheck.Andchecks = []entity.AndCheck{}
		andcheck.Orchecks = []entity.OrCheck{}
		//it's a 'and' policy

		valueArrays := strings.Split(value, " and ")
		for i := 0; i < len(valueArrays); i++ {
			onePolicy := valueArrays[i]
			//it's a role base check
			anotherOrcheck, anotherAndcheck, anotherBasecheck,
				anotherPos := p.buildCheckObject(onePolicy, originMap)
			if anotherPos == 1 {
				andcheck.Orchecks = append(andcheck.Orchecks, anotherOrcheck)
			} else if anotherPos == 2 {
				andcheck.Andchecks = append(andcheck.Andchecks, anotherAndcheck)
			} else if anotherPos == 3 {
				andcheck.Basechecks = append(andcheck.Basechecks, anotherBasecheck)
			}
		}

		pos = 2
	} else {
		if strings.HasPrefix(value, "role:") {
			value = strings.TrimPrefix(value, "role:")
			basecheck = entity.BaseCheck{}
			basecheck.Checktype = "role"
			basecheck.Value = strings.TrimSpace(value)
			pos = 3
		} else if strings.HasPrefix(value, "field:") {
			value = strings.TrimPrefix(value, "field:")
			basecheck = entity.BaseCheck{}
			basecheck.Checktype = "field"
			basecheck.Value = strings.TrimSpace(value)
			pos = 3
		} else if strings.HasPrefix(value, "rule:") {
			//it's another rule need to be refined
			onePolicy := strings.TrimPrefix(value, "rule:")
			policyValue := originMap[onePolicy]
			orcheck, andcheck, basecheck, pos := p.buildCheckObject(policyValue, originMap)
			return orcheck, andcheck, basecheck, pos
		}
	}

	return
}

//return a query language by authenticate
//this will be used for listAll, list and deleteAll interface
func (p *AuthService) BuildQueryByAuth(operation string, tokenId string) (query bson.M, err error) {
	logrus.Infoln("build query object by auth")

	token, err := GetTokenById(tokenId)
	if err != nil {
		logrus.Errorln("get token error:", err)
		return nil, err
	}

	authParam := &AuthParam{
		auth_token:          token,
		auth_collectionName: "",
		auth_instanceId:     ""}

	policyValue, exist := AUTHDATA[operation]
	if !exist {
		logrus.Infoln("no auth policy for specific operation, operation:", operation)
		query = bson.M{}
		return
	}

	switch checkType := policyValue.(type) {
	case entity.OrCheck:
		orresult, orsuccess := p.orCheck_query(checkType.Basechecks, checkType.Andchecks, checkType.Orchecks, authParam)
		if !orsuccess {
			logrus.Warnln("build auth query error")
			return nil, errors.New("build auth query error")
		}
		query, err = p.format(orresult)
		if err != nil {
			logrus.Warnln("format query result to bson error %v", err)
			return nil, err
		}
		return
	case entity.AndCheck:
		andresult, andsuccess := p.andCheck_query(checkType.Basechecks, checkType.Andchecks, checkType.Orchecks, authParam)
		if !andsuccess {
			logrus.Warnln("build auth query error")
			return nil, errors.New("build auth query error")
		}
		query, err = p.format(andresult)
		if err != nil {
			logrus.Warnln("format query result to bson error %v", err)
			return nil, err
		}
		return
	case entity.BaseCheck:
		baseresult, basesuccess := p.baseCheck_query(checkType.Checktype, checkType.Value, authParam)
		if !basesuccess {
			logrus.Warnln("build auth query error")
			return nil, errors.New("build auth query error")
		}
		query, err = p.format(baseresult)
		if err != nil {
			logrus.Warnln("format query result to bson error %v", err)
			return nil, err
		}
		return
	default:
		logrus.Errorln("unkonwn check type:", checkType)
		return nil, errors.New("unknown check type")
	}

}

func (p *AuthService) orCheck_query(basecheck []entity.BaseCheck, andcheck []entity.AndCheck, orcheck []entity.OrCheck, authParam *AuthParam) (map[string]interface{}, bool) {
	for i := 0; i < len(basecheck); i++ {
		oneBaseCheck := basecheck[i]
		onebasequery, onebaseresult := p.baseCheck_query(oneBaseCheck.Checktype, oneBaseCheck.Value, authParam)
		if onebaseresult {
			return onebasequery, true
		}
	}

	for i := 0; i < len(andcheck); i++ {
		oneAndCheck := andcheck[i]
		oneandquery, oneandresult := p.andCheck_query(oneAndCheck.Basechecks, oneAndCheck.Andchecks, oneAndCheck.Orchecks, authParam)
		if oneandresult {
			return oneandquery, true
		}
	}

	for i := 0; i < len(orcheck); i++ {
		oneOrCheck := orcheck[i]
		oneorquery, oneorresult := p.orCheck_query(oneOrCheck.Basechecks, oneOrCheck.Andchecks, oneOrCheck.Orchecks, authParam)
		if oneorresult {
			return oneorquery, true
		}
	}

	return nil, false
}

func (p *AuthService) andCheck_query(basecheck []entity.BaseCheck, andcheck []entity.AndCheck, orcheck []entity.OrCheck, authParam *AuthParam) (map[string]interface{}, bool) {
	basequery := make(map[string]interface{})
	for i := 0; i < len(basecheck); i++ {
		oneBaseCheck := basecheck[i]
		onebasequery, onebaseresult := p.baseCheck_query(oneBaseCheck.Checktype, oneBaseCheck.Value, authParam)
		if !onebaseresult {
			return onebasequery, false
		}

		p.mergeQuery(basequery, onebasequery)
	}

	for i := 0; i < len(andcheck); i++ {
		oneAndCheck := andcheck[i]
		oneandquery, oneandresult := p.andCheck_query(oneAndCheck.Basechecks, oneAndCheck.Andchecks, oneAndCheck.Orchecks, authParam)
		if !oneandresult {
			return oneandquery, false
		}

		p.mergeQuery(basequery, oneandquery)
	}

	for i := 0; i < len(orcheck); i++ {
		oneOrCheck := orcheck[i]
		oneorquery, oneorresult := p.orCheck_query(oneOrCheck.Basechecks, oneOrCheck.Andchecks, oneOrCheck.Orchecks, authParam)
		if !oneorresult {
			return oneorquery, false
		}

		p.mergeQuery(basequery, oneorquery)
	}

	return basequery, true
}

func (p *AuthService) baseCheck_query(checktype string, value string, authParam *AuthParam) (map[string]interface{}, bool) {
	basequery := make(map[string]interface{})

	if strings.EqualFold(checktype, "role") {
		if strings.EqualFold(value, authParam.auth_token.Role.Rolename) {
			return basequery, true
		} else {
			return nil, false
		}
	} else if strings.EqualFold(checktype, "field") {
		valueArrays := strings.Split(value, "=")
		if len(valueArrays) != 2 {
			logrus.Errorln("a field policy format error! value is :", value)
			return nil, false
		}

		//handle first value
		value1 := valueArrays[0]

		// handle second value(can be a value or from token)
		value2 := valueArrays[1]
		if strings.HasPrefix(value2, "%") {
			key2 := strings.TrimPrefix(value2, "%")
			value2 = p.getValueFromToken(authParam.auth_token, key2)
		}

		basequery[value1] = value2

		return basequery, true

	} else {
		logrus.Errorln("unsupported checktype:", checktype)
		return nil, false
	}
}

func (p *AuthService) mergeQuery(basequery map[string]interface{}, plusquery map[string]interface{}) {
	if plusquery == nil {
		return
	}
	for key, value := range plusquery {
		basequery[key] = value
	}
	return
}

func (p *AuthService) format(ret map[string]interface{}) (bson.M, error) {
	if ret == nil || len(ret) <= 0 {
		return bson.M{}, nil
	}

	query, err := mejson.Unmarshal(ret)
	if err != nil {
		logrus.Warnln("unmarshal query ret to bson error %v", err)
		return nil, errors.New("unmarshal query ret to bson error")
	}

	return query, nil
}

//will return true or false
//this will be used for create, update and delete interface
func (p *AuthService) Authorize(operation string, tokenid string, instanceId string, collectionName string) bool {
	logrus.Infoln("begin to check the operation:", operation)
	token, err := GetTokenById(tokenid)
	if err != nil {
		logrus.Errorln("get token error:", err)
		return false
	}

	authParam := &AuthParam{
		auth_instanceId:     instanceId,
		auth_token:          token,
		auth_collectionName: collectionName}

	policyValue, exist := AUTHDATA[operation]
	if !exist {
		logrus.Infoln("no auth policy for specific operation, operation:", operation)
		return true
	}

	return p.check(policyValue, authParam, nil)
}

func (p *AuthService) AuthOperation(operations []string, tokenid string, objectId string, collectionName string) (map[string]int, error) {
	logrus.Infoln("begin to list authorized operations")
	token, err := GetTokenById(tokenid)
	if err != nil {
		logrus.Errorln("get token error:", err)
		return nil, errors.New("get token error!")
	}

	data := p.getObjectById(objectId, collectionName)

	authParam := &AuthParam{
		auth_instanceId:     "",
		auth_token:          token,
		auth_collectionName: ""}

	result := make(map[string]int)

	for i := 0; i < len(operations); i++ {
		op := operations[i]
		policyValue, exist := AUTHDATA[op]
		if !exist {
			logrus.Infoln("no auth policy for specific operation, operation:", op)
			result[op] = 1
			continue
		}

		if p.check(policyValue, authParam, data) {
			result[op] = 1
		} else {
			result[op] = 0
		}
	}

	return result, nil
}

func (p *AuthService) check(policyValue interface{}, authParam *AuthParam, data interface{}) bool {
	switch checkType := policyValue.(type) {
	case entity.OrCheck:
		orresult := p.orCheck(checkType.Basechecks, checkType.Andchecks, checkType.Orchecks, authParam, data)
		return orresult
	case entity.AndCheck:
		andresult := p.andCheck(checkType.Basechecks, checkType.Andchecks, checkType.Orchecks, authParam, data)
		return andresult
	case entity.BaseCheck:
		baseresult := p.baseCheck(checkType.Checktype, checkType.Value, authParam, data)
		return baseresult
	default:
		logrus.Errorln("unkonwn check type:", checkType)
		return true
	}
}

func (p *AuthService) orCheck(basecheck []entity.BaseCheck, andcheck []entity.AndCheck, orcheck []entity.OrCheck, authParam *AuthParam, data interface{}) bool {
	for i := 0; i < len(basecheck); i++ {
		oneBaseCheck := basecheck[i]
		baseresult := p.baseCheck(oneBaseCheck.Checktype, oneBaseCheck.Value, authParam, data)
		if baseresult {
			return true
		}
	}

	for i := 0; i < len(andcheck); i++ {
		oneAndCheck := andcheck[i]
		andresult := p.andCheck(oneAndCheck.Basechecks, oneAndCheck.Andchecks, oneAndCheck.Orchecks, authParam, data)
		if andresult {
			return true
		}
	}

	for i := 0; i < len(orcheck); i++ {
		oneOrCheck := orcheck[i]
		orresult := p.orCheck(oneOrCheck.Basechecks, oneOrCheck.Andchecks, oneOrCheck.Orchecks, authParam, data)
		if orresult {
			return true
		}
	}

	return false
}

func (p *AuthService) andCheck(basecheck []entity.BaseCheck, andcheck []entity.AndCheck, orcheck []entity.OrCheck, authParam *AuthParam, data interface{}) bool {
	for i := 0; i < len(basecheck); i++ {
		oneBaseCheck := basecheck[i]
		baseresult := p.baseCheck(oneBaseCheck.Checktype, oneBaseCheck.Value, authParam, data)
		if !baseresult {
			return false
		}
	}

	for i := 0; i < len(andcheck); i++ {
		oneAndCheck := andcheck[i]
		andresult := p.andCheck(oneAndCheck.Basechecks, oneAndCheck.Andchecks, oneAndCheck.Orchecks, authParam, data)
		if !andresult {
			return false
		}
	}

	for i := 0; i < len(orcheck); i++ {
		oneOrCheck := orcheck[i]
		orresult := p.orCheck(oneOrCheck.Basechecks, oneOrCheck.Andchecks, oneOrCheck.Orchecks, authParam, data)
		if !orresult {
			return false
		}
	}

	return true
}

func (p *AuthService) baseCheck(checktype string, value string, authParam *AuthParam, data interface{}) bool {
	if strings.EqualFold(checktype, "role") {
		if strings.EqualFold(value, authParam.auth_token.Role.Rolename) {
			return true
		} else {
			return false
		}

	} else if strings.EqualFold(checktype, "field") {
		valueArrays := strings.Split(value, "=")
		if len(valueArrays) != 2 {
			logrus.Errorln("a generic policy format error! value is :", value)
			return false
		}

		//handle field first value (in response or need do request first)
		value1s := p.getValueFromField(authParam, valueArrays[0], data)
		if len(*value1s) == 0 {
			return false
		}

		// handle second value(can be a value or from token)
		value2 := valueArrays[1]
		if strings.HasPrefix(value2, "%") {
			key2 := strings.TrimPrefix(value2, "%")
			value2 = p.getValueFromToken(authParam.auth_token, key2)
		}

		for _, value1 := range *value1s {
			if !strings.EqualFold(value1, value2) {
				return false
			}
		}

		return true

	} else {
		logrus.Errorln("unsupported checktype:", checktype)
		return false
	}
}

func (p *AuthService) getValueFromToken(token *entity.Token, key string) (value string) {
	if strings.EqualFold(key, "user_id") {
		value = token.User.Id
	} else if strings.EqualFold(key, "tenant_id") {
		value = token.Tenant.Id
	} else {
		logrus.Errorln("not supported key:", key)
	}

	return
}

func (p *AuthService) getValueFromField(authParam *AuthParam, key string, data interface{}) *[]string {
	ret := []string{}
	originData := data
	if originData == nil {
		logrus.Infoln("getValueFromField data object is null, will do query first ")
		originData = p.getObjectById(authParam.auth_instanceId, authParam.auth_collectionName)
	}

	if originData == nil {
		return &ret
	}

	//convert to json format
	formatdata, err := mejson.Marshal(originData)
	if err != nil {
		logrus.Errorln("convert origindata object error %v", err)
		return &ret
	}

	value := p.getValueFromObject(formatdata, key)
	if len(value) != 0 {
		ret = append(ret, value)
	}

	return &ret
}

func (p *AuthService) getObjectById(instanceId string, collectionName string) (data interface{}) {
	if !bson.IsObjectIdHex(instanceId) {
		logrus.Errorf("AuthService:getObjectById --- invalid instanceId")
		return nil
	}
	selector := make(bson.M)
	selector["_id"] = bson.ObjectIdHex(instanceId)

	// parse request
	// var fields bson.M = buildFields()
	data = bson.M{}
	queryStruct := dao.QueryStruct{
		CollectionName: collectionName,
		Selector:       selector,
		Skip:           0,
		Limit:          0,
		Sort:           ""}

	// _, _, data, err = p.Dao.HandleQuery(collectionName, selector, true, fields, 0, 1, "", "true")
	err := dao.HandleQueryOne(data, queryStruct)
	if err != nil {
		logrus.Errorf("handle query err is %v", err)
		return nil
	}
	return
}

func (p *AuthService) getValueFromObject(data interface{}, key string) string {
	record := data.(map[string]interface{})
	value := record[key]
	if value == nil {
		return "false"
	} else {
		return value.(string)
	}
}

func (p *AuthService) buildFields() bson.M {
	selector := bson.M{}
	selector["user_id"] = 1
	selector["tenant_id"] = 1

	return selector
}
