package dingtalk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed customBotAdapter
var customBotAdapterDir embed.FS

//go:embed customBotAdapter.json
var customAdapterDefinition string

//go:embed corpBotAdapter
var corpBotAdapterDir embed.FS

//go:embed corpBotAdapter.json
var corpAdapterDefinition string

func init() {
	customBotAdapter := adapter.RegisterAdapterByRaw([]byte(customAdapterDefinition))
	customBotAdapter.RegisterSpecsByDir(customBotAdapterDir)

	customBotAdapter.RegisterCredentialTemplate(adapter.AccessTokenCredentialType, `{
	"metaData": {
		"secret": "{{ .secret }}"
	},
	"accessToken": "{{ .accessToken }}"
}`)
	customBotAdapter.RegisterCredentialTestingFunc(testCustomBotCredential)

	workflow.RegistryNodeMeta(&SendTextMessage{})
	workflow.RegistryNodeMeta(&SendMarkdownMessage{})
	workflow.RegistryNodeMeta(&SendLinkMessage{})
	workflow.RegistryNodeMeta(&SendFeedCardMessage{})
	workflow.RegistryNodeMeta(&SendActionCardMessage{})
	workflow.RegistryNodeMeta(&SendMessage{})

	corpBotAdapter := adapter.RegisterAdapterByRaw([]byte(corpAdapterDefinition))
	corpBotAdapter.RegisterSpecsByDir(corpBotAdapterDir)
	workflow.RegistryNodeMeta(&CorpBotMessageTrigger{})
	workflow.RegistryNodeMeta(&ReplyMessage{})
}

var testTemplate = `测试密钥：账户授权成功。
> Ultrafox`

func testCustomBotCredential(ctx context.Context, _ model.CredentialType, fields model.InputFields) (err error) {
	// TODO: https://jihulab.com/ultrafox/ultrafox/-/issues/635#note_2124129
	return

	// accessToken, ok := fields.GetString2("accessToken", false)
	// if !ok {
	// 	err = fmt.Errorf("accessToken is required")
	// 	return
	// }
	// webhookURL, err := makeWebhookUrl(accessToken, fields.GetString("secret"))
	// if err != nil {
	// 	err = fmt.Errorf("make dingtalk webhook URL: %w", err)
	// 	return
	// }
	// err = sendWebhook(ctx, webhookURL, &DingTalkMessage{
	// 	MessageType: "markdown",
	// 	MarkdownMessage: MarkdownMessage{
	// 		Title: "Ultrafox 测试秘钥",
	// 		Text:  testTemplate,
	// 	},
	// })
}

const SendSuccessStatus = 0

type dingTalkAuthMeta struct {
	Secret string `json:"secret"`
}

type DingResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// TODO: accessToken is webhookUrl actually since 0.12
func makeWebhookUrl(accessToken, secret string) (string, error) {
	webhookUrl := accessToken
	if !strings.HasPrefix(accessToken, "https://") {
		webhookUrl = fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", accessToken)
	}

	if secret == "" {
		return webhookUrl, nil
	}

	timestamp, sign := makeSign(secret)

	webhookUrl += fmt.Sprintf("&timestamp=%d&sign=%s", timestamp, sign)
	return webhookUrl, nil
}

func makeSign(secret string) (int64, string) {
	timestamp := time.Now().UnixMilli()

	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(stringToSign))
	signData := hash.Sum(nil)
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(signData))

	return timestamp, sign
}
