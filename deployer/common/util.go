package common

import (
	"bytes"
	"encoding/base64"
	"errors"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/magiconair/properties"
)

const (
	base64Table    = "ABCDEFGHIJKLMNOPQRSTpqrstuvwxyz0123456789+/UVWXYZabcdefghijklmno"
	REGISTRY_LABLE = "{registry}"
)

var coder = base64.NewEncoding(base64Table)

var UTIL *Util

type Util struct {
	Props *properties.Properties
}

func GetClusterEndpoint() (endpoint string, err error) {
	host := UTIL.Props.MustGetString("lb.host")
	clusterPort := UTIL.Props.MustGetString("lb.cluster.port")
	if len(host) <= 0 || len(clusterPort) <= 0 {
		logrus.Errorf("can not get lb host or cluster port!")
		return endpoint, errors.New("no lb host or cluster port configured!")
	}
	endpoint = host + ":" + clusterPort
	return
}

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func GetUserEndpoint() (endpoint string, err error) {
	host := UTIL.Props.MustGetString("lb.host")
	userPort := UTIL.Props.MustGetString("lb.usermgmt.port")
	if len(host) <= 0 || len(userPort) <= 0 {
		logrus.Errorf("can not get lb host or user port!")
		return endpoint, errors.New("no lb host or user port configured!")
	}
	endpoint = host + ":" + userPort
	return
}

func Base64Encode(src []byte) []byte {
	return []byte(coder.EncodeToString(src))
}

func Base64Decode(src []byte) ([]byte, error) {
	return coder.DecodeString(string(src))
}

//If we don't do this, POST method without Content-type (even with empty body) will fail
func ParseForm(r *http.Request) error {
	if r == nil {
		return nil
	}
	if err := r.ParseForm(); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

func matchesContentType(contentType, expectedType string) bool {
	mimetype, _, err := mime.ParseMediaType(contentType)
	return err == nil && mimetype == expectedType
}

//generate docker registry for config file. (yml and json)
func GenRegistry(registry string, configFile []byte) []byte {

	replacedStr := REGISTRY_LABLE
	if strings.EqualFold(registry, "") {
		replacedStr = REGISTRY_LABLE + "/"
	}
	return bytes.Replace(configFile, []byte(replacedStr), []byte(registry), -1)
}
