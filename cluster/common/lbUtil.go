package common

import (
	"encoding/base64"
	"github.com/magiconair/properties"
)

const (
	base64Table = "ABCDEFGHIJKLMNOPQRSTpqrstuvwxyz0123456789+/UVWXYZabcdefghijklmno"

	cmiPort = "10030"
)

var coder = base64.NewEncoding(base64Table)

var (
	UTIL *Util
)

type Util struct {
	Props    *properties.Properties
	LbClient *LbClient
}

type LbClient struct {
	Host string
}

func (p *LbClient) GetUserMgmtEndpoint() (endpoint string, err error) {
	userMgmtPort := UTIL.Props.MustGetString("lb.usermgmt.port")
	endpoint = p.Host + ":" + userMgmtPort
	return
}

func (p *LbClient) GetDeployEndpoint() (endpoint string, err error) {
	deployPort := UTIL.Props.MustGetString("lb.deploy.port")
	endpoint = p.Host + ":" + deployPort
	return
}

func (p *LbClient) GetClientPort() (port string, err error) {
	port = UTIL.Props.MustGetString("client.port")
	return
}

func Base64Encode(src []byte) []byte {
	return []byte(coder.EncodeToString(src))
}

func Base64Decode(src []byte) ([]byte, error) {
	return coder.DecodeString(string(src))
}

func GetCMIEndpoint(ip string) (endpoint string) {
	endpoint = ip + ":" + cmiPort
	return
}
