package tencentcloud

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func Test_testCredential(t *testing.T) {
	type args struct {
		fields model.InputFields
	}
	tests := []struct {
		name        string
		args        args
		mockFn      func()
		deferMockFn func()
		wantErr     bool
	}{
		{
			name: "empty secretId",
			args: args{
				fields: map[string]any{
					"secretId":  "",
					"secretKey": "key",
				},
			},
			wantErr: true,
		},
		{
			name: "empty secretKey",
			args: args{
				fields: map[string]any{
					"secretId":  "id",
					"secretKey": "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid secretId",
			args: args{
				fields: map[string]any{
					"secretId":  "id",
					"secretKey": "key",
				},
			},
			wantErr: true,
		},
		{
			name: "valid secretId and secretKey",
			args: args{
				fields: map[string]any{
					"secretId":  "id",
					"secretKey": "key",
				},
			},
			mockFn: func() {
				httpmock.Activate()
				httpmock.RegisterResponder("POST", "https://cvm.tencentcloudapi.com/",
					func(req *http.Request) (*http.Response, error) {
						resp, _ := httpmock.NewJsonResponse(200, map[string]string{})
						return resp, nil
					},
				)
			},
			deferMockFn: func() {
				httpmock.DeactivateAndReset()
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockFn != nil {
				tt.mockFn()
			}
			if tt.deferMockFn != nil {
				defer tt.deferMockFn()
			}

			if err := testCredential(context.TODO(), model.CredentialTypeCustom, tt.args.fields); (err != nil) != tt.wantErr {
				t.Errorf("testCredential() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
