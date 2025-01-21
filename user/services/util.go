package services

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"encoding/json"
	"github.com/Sirupsen/logrus"
	"linkernetworks.com/dcos-backend/common/httpclient"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
	"linkernetworks.com/dcos-backend/common/rest/response"
	"linkernetworks.com/dcos-backend/user/common"
)

var (
	COMMON_ERROR_INVALIDATE   = "E12002"
	COMMON_ERROR_UNAUTHORIZED = "E12004"
	COMMON_ERROR_UNKNOWN      = "E12001"
	COMMON_ERROR_INTERNAL     = "E12003"
)

type UserParam struct {
	UserName string
	Email    string
	//	Password string
	Company string
	RoleType string
}

/*func IsFirstNodeInZK() bool {
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Warnln("get host name error!", err)
		return false
	}

	path, err := common.UTIL.ZkClient.GetFirstUserMgmtPath()
	if err != nil {
		logrus.Warnln("get usermgmt node from zookeeper error!", err)
		return false
	}

	return strings.HasPrefix(path, hostname)

}*/

func HashString(password string) string {
	encry := md5.Sum([]byte(password))
	return hex.EncodeToString(encry[:])
}

func IsUserNameValid(name string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9.-]{1,10}$`)
	return reg.MatchString(name)
}

func GetWaitTime(execTime time.Time) int64 {
	one_day := 24 * 60 * 60
	currenTime := time.Now()

	execInt := execTime.Unix()
	currentInt := currenTime.Unix()

	var waitTime int64
	if currentInt <= execInt {
		waitTime = execInt - currentInt
	} else {
		waitTime = (execInt + int64(one_day)) % currentInt
	}

	return waitTime
}

//default expire time is 6 hours
func GenerateExpireTime(expire int64) float64 {
	t := time.Now().Unix()

	t += expire

	return float64(t)
}

func GetClusterByUser(userid string, x_auth_token string) (cluster []entity.Cluster, err error) {
	clusterurl := common.UTIL.Props.GetString("lb.url", "")

	caCertPath := common.UTIL.Props.GetString("http.cluster.https.crt", "")
	isHttpsEnabled := common.UTIL.Props.GetBool("http.cluster.https.enabled", false)

	url := strings.Join([]string{clusterurl, ":10002", "/v1/cluster?user_id=", userid, "&status=unterminated"}, "")
	logrus.Debugln("get cluster url=" + url)

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", x_auth_token})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", x_auth_token})
	}

	if err != nil {
		logrus.Errorf("http get cluster error %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get cluster by username failed %v", string(data))
		return nil, errors.New("get cluster by username failed")
	}

	cluster = []entity.Cluster{}
	err = getRetFromResponse(data, &cluster)
	return

}

func GetSmtpInfo(x_auth_token string) (smtps []entity.Smtp, err error) {
	clusterurl := common.UTIL.Props.GetString("lb.url", "")
	caCertPath := common.UTIL.Props.GetString("http.cluster.https.crt", "")
	isHttpsEnabled := common.UTIL.Props.GetBool("http.cluster.https.enabled", false)

	url := strings.Join([]string{clusterurl, ":10002", "/v1/smtp"}, "")
	logrus.Debugln("get smtp url=" + url)

	var resp *http.Response

	if isHttpsEnabled {
		resp, err = httpclient.Https_get(url, "", caCertPath,
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", x_auth_token})
	} else {
		resp, err = httpclient.Http_get(url, "",
			httpclient.Header{"Content-Type", "application/json"},
			httpclient.Header{"X-Auth-Token", x_auth_token})
	}
	if err != nil {
		logrus.Errorf("http get smtp error %v", err)
		return nil, err
	}

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("get smtp by username failed %v", string(data))
		return nil, errors.New("get smtp  failed")
	}

	smtps = []entity.Smtp{}
	err = getRetFromResponse(data, &smtps)
	return
}

func SendLog (errs error, operationType, queryType, comments, token string) (ispost bool, err error) {
	var status string
	if errs != nil {
		status = "fail"
	} else {
		status = "success"
	}
	currentToken, _ := GetTokenService().GetTokenById(token)
	var loguser entity.LogMessage
	loguser.OperateType = operationType
	loguser.QueryType = queryType
	loguser.Comments = comments
	loguser.Status = status
	loguser.UserId = currentToken.User.Id
	loguser.TenantId = currentToken.Tenant.Id
	loguser.Username = currentToken.User.Username
	
	clusterurl := common.UTIL.Props.GetString("lb.url", "")
	caCertPath := common.UTIL.Props.GetString("http.cluster.https.crt", "")
	isHttpsEnabled := common.UTIL.Props.GetBool("http.cluster.https.enabled", false)

	url := strings.Join([]string{clusterurl, ":10002", "/v1/logs/create"}, "")
	var resp *http.Response
	
	body, err := json.Marshal(loguser)
	logrus.Infof("body is %v", body)
	if err != nil {
		logrus.Errorf("get cluster endpoint err is %v", err)
		return false, err
	}
	if isHttpsEnabled {
		resp, err = httpclient.Https_post(url, string(body), caCertPath,
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	} else {
		resp, err = httpclient.Http_post(url, string(body),
			httpclient.Header{"Content-Type", "application/json"}, httpclient.Header{"X-Auth-Token", token})
	}
	if err != nil {
		logrus.Errorf("send http post to dcos cluster error %v", err)
		return false, err
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		logrus.Errorf("http status code from dcos cluster failed %v", string(data))
		return false, errors.New("http status code from dcos cluster failed")
	}

	success := isResponseSuccess(data)
	if !success {
		return false, errors.New("send log not success")
	}

	return true, nil
	
}

func getRetFromResponse(data []byte, obj interface{}) (err error) {
	var resp *response.Response
	resp = new(response.Response)
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return err
	}

	jsonout, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	json.Unmarshal(jsonout, obj)

	return
}

func isResponseSuccess(data []byte) bool {
	var resp *response.Response
	resp = new(response.Response)
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return false
	}

	return resp.Success
}
