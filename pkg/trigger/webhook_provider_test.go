package trigger

import (
	"context"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"testing"
)

func TestWebhookContext_GetWebhookURL(t *testing.T) {
	ctx := context.Background()
	host, err := serverhost.New(serverhost.Opt{
		API:     "https://example.com/api",
		WebHook: "https://example.com/hooks",
	})

	if err != nil {
		panic(err)
	}

	wb := newWebhookContext(webhookContextOpt{
		Ctx:           ctx,
		ConfigObject:  map[string]any{},
		AuthSignature: nil,
		ServerHost:    host,
		IsSalesforce:  true,
	})

	url := wb.GetWebhookURL()
	assert.Equal(t, url, "https://example.com/salesforce/hooks/")
	wb.IsSalesforce = false
	url = wb.GetWebhookURL()
	assert.Equal(t, url, "https://example.com/hooks/")
}
