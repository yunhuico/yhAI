package auth_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	cryptoMock "jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto/mock"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/gitlab"
)

func TestMain(m *testing.M) {
	log.Init("go-testing", log.DebugLevel)
	os.Exit(m.Run())
}

func TestNewAuthorizerError(t *testing.T) {
	_, err := auth.NewAuthorizer(nil, nil)
	assert.Error(t, err)
}

func TestAccessTokenAuthorizerWork(t *testing.T) {
	authHeaderKey := "Authorization"
	authHeaderValue := "this-is-access-token"
	mux := http.NewServeMux()
	mux.HandleFunc("/", checkAuthHeaderHandler(authHeaderKey, authHeaderValue))

	credential := &model.Credential{
		EditableCredential: model.EditableCredential{
			Type:         model.CredentialTypeAccessToken,
			AdapterClass: "ultrafox/gitlab",
		},
	}

	t.Run("new authorizer failed because the data invalid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCipher := cryptoMock.NewMockCryptoCipher(ctrl)
		mockCipher.EXPECT().Decrypt(gomock.Any()).Return([]byte(`invalid json`), nil)
		_, err := auth.NewAuthorizer(mockCipher, credential, auth.WithRequestSignMethod(auth.NewTokenSignMethod(authHeaderKey)))
		assert.Error(t, err)
	})
}

func checkAuthHeaderHandler(authHeaderKey string, authHeaderValue string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		authorization := request.Header.Get(authHeaderKey)
		if authorization == "" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(authorization, authHeaderValue) {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}
}

func TestKVAuthorizer(t *testing.T) {
	credential := &model.Credential{
		EditableCredential: model.EditableCredential{
			Type:         model.CredentialTypeAccessToken,
			AdapterClass: "ultrafox/gitlab",
		},
	}

	t.Run("test decode error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCipher := cryptoMock.NewMockCryptoCipher(ctrl)
		mockCipher.EXPECT().Decrypt(gomock.Any()).Return([]byte(`{`), nil)

		_, err := auth.NewAuthorizer(mockCipher, credential)
		assert.Error(t, err)
	})

	t.Run("test decode success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCipher := cryptoMock.NewMockCryptoCipher(ctrl)
		mockCipher.EXPECT().Decrypt(gomock.Any()).Return([]byte(`{"server": "ultrafox.io", "accessToken": ""}`), nil)

		decoder, err := auth.NewAuthorizer(mockCipher, credential)
		assert.NoError(t, err)

		type myServer struct {
			Server string `json:"server"`
		}
		meta := &myServer{}
		err = decoder.DecodeMeta(meta)
		assert.NoError(t, err)
		assert.Equal(t, "ultrafox.io", meta.Server)
	})
}
