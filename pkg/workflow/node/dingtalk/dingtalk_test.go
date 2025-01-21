package dingtalk

import (
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
)

func TestDingtalkCredentialTemplate(t *testing.T) {
	manager := adapter.GetAdapterManager()
	meta := manager.LookupAdapter("ultrafox/dingtalk")
	meta.TestCredentialTemplate(adapter.AccessTokenCredentialType, map[string]any{
		"accessToken": "xxxxx",
	})

	meta = manager.LookupAdapter("ultrafox/dingtalkCorpBot")
	meta.TestCredentialTemplate(adapter.CustomCredentialType, map[string]any{
		"accessToken": "xxxxx",
	})
}

func Test_makeWebhookUrl(t *testing.T) {
	type args struct {
		accessToken string
		secret      string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "old accessToken",
			args: args{
				accessToken: "token",
			},
			want:    "https://oapi.dingtalk.com/robot/send?access_token=token",
			wantErr: false,
		},
		{
			name: "new accessToken is webhookUrl",
			args: args{
				accessToken: "https://oapi.dingtalk.com/robot/send?access_token=token",
			},
			want:    "https://oapi.dingtalk.com/robot/send?access_token=token",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makeWebhookUrl(tt.args.accessToken, tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeWebhookUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("makeWebhookUrl() got = %v, want %v", got, tt.want)
			}
		})
	}
}
