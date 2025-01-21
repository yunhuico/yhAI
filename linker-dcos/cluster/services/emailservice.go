package services

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/gomail.v2"
	"linkernetworks.com/dcos-backend/cluster/common"
	"linkernetworks.com/dcos-backend/common/persistence/entity"
)

const (
	EMAIL_ERROR_SEND string = "E50200"
)

var (
	emailService *EmailService = nil
	onceEmail    sync.Once
)

type EmailService struct {
	ServiceName string
}

func GetEmailService() *EmailService {
	onceEmail.Do(func() {
		logrus.Debugf("Once called from emailService ......................................")
		emailService = &EmailService{"emailservice"}
	})
	return emailService
}

var subjectTemplate = []byte(`[Linker Cloud Platform] Your cluster has been deployed!`)

var bodyTemplate = []byte(`
<USER_NAME>, 这封邮件是由领科云发送的。

<ENDPOINT>

如果有任何问题，请发送邮件到 support@linkernetworks.com 
请勿回复该邮件


此致
领科云管理团队



<USER_NAME>, This email is sent by Linker Cloud Platform.

<ENDPOINT>

Any problems, please send mail to support@linkernetworks.com
Please DO NOT reply this mail

Thanks & BestRegards!

Linker Cloud Platform Team
`)

func (p *EmailService) SendAlertEmail(sendemail entity.SendHostAlertReq) (errorCode string, err error) {
	logrus.Infof("start to send alert email")
	userId := sendemail.UserId
	token := common.UTIL.Props.GetString("cluster.user.token", "")
	if token == "" {
		logrus.Errorf("token cannot be null")
		return COMMON_ERROR_INTERNAL, errors.New("token cannot be null")
	}
	logrus.Infof("token is %v", token)
	user, err := GetUserByIdForAlert(userId, token)
	if err != nil {
		logrus.Errorf("get user err is %v", err)
		return COMMON_ERROR_INTERNAL, errors.New("get user is err")
	}
	
	body := []byte(sendemail.Content)
	emailAddress := user.Email
	logrus.Infof("user email is %v", emailAddress)
	if !isEmailAddressValid(emailAddress) {
		logrus.Errorln("invalid email of user with id %s:%s", userId, emailAddress)
		return COMMON_ERROR_INTERNAL, errors.New("invalid email of the cluster owner")
	}
	subject := sendemail.Subject
	err = p.sendConfigedEmail(emailAddress, string(subject), string(body), token)
	if err != nil {
		logrus.Errorf("fail to send email to %s,reason %v", emailAddress, err)
		return EMAIL_ERROR_SEND, err
	}
	return
}

func (p *EmailService) SendClusterDeployedEmail(clusterId string, token string) (errorCode string, err error) {
	//clusterId->entity.Cluster
	errorCode, err = TokenValidation(token)
	if err != nil {
		logrus.Errorf("token validate err is %v", err)
		return
	}

	cluster, errorCode, err := GetClusterService().QueryById(clusterId, token)
	if err != nil {
		logrus.Errorln("error query cluster, %v", err)
		return CLUSTER_ERROR_QUERY, err
	}
	//cluster->Owner
	owner := cluster.Owner
	if len(strings.TrimSpace(owner)) == 0 {
		logrus.Errorln("invalid field Owner in entity cluster %s:%s", clusterId, owner)
		return COMMON_ERROR_INTERNAL, errors.New("invalid field cluster.Owner")
	}

	//cluster->Endpoint
	endpoint := cluster.Endpoint
	if len(endpoint) <= 0 {
		logrus.Errorln("no endpoint for cluster %s", clusterId)
		return COMMON_ERROR_INTERNAL, errors.New("invalid field cluster.Endpoint")
	}

	//cluster->UserId
	userId := cluster.UserId
	if len(strings.TrimSpace(userId)) == 0 {
		logrus.Errorln("invalid UserId in entity cluster %s:%s", clusterId, userId)
		return COMMON_ERROR_INTERNAL, errors.New("invalid filed cluster.UserId")
	}

	//userId->email
	user, err := GetUserById(userId, token)
	if err != nil {
		logrus.Errorln("error get user by id %s,%v", userId, err)
		return COMMON_ERROR_INTERNAL, err
	}
	email := user.Email
	if len(strings.TrimSpace(userId)) == 0 || !isEmailAddressValid(email) {
		logrus.Errorln("invalid email of user with id %s:%s", userId, email)
		return COMMON_ERROR_INTERNAL, errors.New("invalid email of the cluster owner")
	}

	//replace body <USER_NAME>,<ENDPOINT>
	body := p.replaceEmailBody(bodyTemplate, owner, endpoint)
	subject := subjectTemplate

	logrus.Infof("sending email to %s... clusterId %s .userId %s", email, clusterId, userId)

	//send mail
	err = p.sendConfigedEmail(email, string(subject), string(body), token)
	if err != nil {
		logrus.Errorf("fail to send email to %s,reason %v", email, err)
		return EMAIL_ERROR_SEND, err
	}
	return
}

func (p *EmailService) replaceEmailBody(bodyTemplate []byte, userName string, endpoint string) (body []byte) {
	//-1 means replace all
	// allendpoints := p.buildArray(endpoints)
	body = bytes.Replace(bodyTemplate, []byte("<USER_NAME>"), []byte(userName), -1)
	body = bytes.Replace(body, []byte("<ENDPOINT>"), []byte(endpoint), -1)
	return body
}

// func (p *EmailService) buildArray(arrays []string) string {
// 	if len(arrays) <= 0 {
// 		logrus.Warnf("no endpoints to build!")
// 		return ""
// 	}

// 	var commandTextBuffer bytes.Buffer
// 	for _, value := range arrays {
// 		commandTextBuffer.WriteString(value)
// 		commandTextBuffer.WriteString("\n")
// 	}

// 	return commandTextBuffer.String()
// }

//send email with configed host,from,password
func (p *EmailService) sendConfigedEmail(to string, subject string, body string, token string) (err error) {
	total, smtps, _, errq := GetSmtpService().QueryStmp("", "", 0, 0, "", token)
	if errq != nil {
		logrus.Errorf("query smtp info err is %v", err)
		return errq
	}
	if total != 1 {
		logrus.Errorf("the number of smtp is err")
		return
	}
	Smtp := smtps[0]

	emailHost := Smtp.Address
	emailUsername := Smtp.Name
	passwd, errd := common.Base64Decode([]byte(Smtp.PassWd))
	logrus.Infof("smtp password is %v", Smtp.PassWd)
	if errd != nil {
		logrus.Errorf("decode password err is %v", errd)
		return
	}
	emailPasswd := string(passwd)
	logrus.Infof("host is %v, username is %v, password is %v", emailHost, emailUsername, emailPasswd)

	if len(strings.TrimSpace(emailHost)) == 0 {
		return errors.New("read email host from properties error")
	}
	if len(strings.TrimSpace(emailUsername)) == 0 {
		return errors.New("read email username from properties error")
	}
	if len(strings.TrimSpace(emailPasswd)) == 0 {
		return errors.New("read email password from properties error")
	}

	go p.sendEmail(emailHost, emailUsername, emailPasswd, to, subject, body)
	return
}

//send email
func (p *EmailService) sendEmail(host string, from string, password string, to string,
	subject string, body string) (err error) {

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	//port 25
	d := gomail.NewPlainDialer(host, 25, from, password)

	err = d.DialAndSend(m)

	if err != nil {
		logrus.Warnln("send email error %v", err)
	}
	return
}

//check email address with regex
func isEmailAddressValid(email string) bool {
	reg := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return reg.MatchString(email)
}
