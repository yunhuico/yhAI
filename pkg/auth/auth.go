package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/service"
)

type (
	// Authorizer is an interface for authorizing all external application.
	Authorizer interface {
		AccessTokenAccess
		CredentialMetaDecoder

		// CredentialType return the type of credential
		//
		// DEPRECATED: use GetCredential instead, Authorizer may be removed after refactoring the authorization.
		CredentialType() model.CredentialType
	}

	// CredentialMetaDecoder decode, you must need know:
	// one node only have one kv credential.
	CredentialMetaDecoder interface {
		// DecodeMeta decode the metadata
		DecodeMeta(meta interface{}) error
		// DecodeTokenMetaData decode the metadata of token
		DecodeTokenMetaData(ctx context.Context, meta interface{}) error
	}

	// AccessTokenAccess provider access token
	AccessTokenAccess interface {
		// GetAccessToken return the access token directly.
		// Some node use sdk depend on access token, so we need to get it directly
		GetAccessToken(context.Context) (string, error)
	}

	baseAuthorizer struct {
		credential *model.Credential
		credData   *model.CredentialData
		cipher     crypto.CryptoCipher
	}

	accessTokenAuthorizer struct {
		*baseAuthorizer
		signMethod *TokenSignMethod
	}

	// oauth2Authorizer serves for legacy OAuth2 functions
	// DEPRECATED: use oauthAuthorizer instead
	oauth2Authorizer struct {
		*baseAuthorizer
		updateOAuth2CredentialTokenFunc UpdateOAuth2CredentialTokenFunc // DEPRECATED: this field serves for legacy OAuth2 credentials, which are being phased out.
	}

	oauthAuthorizer struct {
		*baseAuthorizer
		credentialUpdater OAuthCredentialUpdater
	}

	customAuthorizer struct {
		*baseAuthorizer
	}

	// TokenSignMethod for custom how to sign request in accessTokenAuthorizer
	TokenSignMethod struct {
		methodType tokenSignMethodType
		properties map[string]string
	}

	tokenSignMethodType string

	// UpdateOAuth2CredentialTokenFunc for update the credential connection data when oauth2 token refreshed.
	//
	// DEPRECATED: this field serves for legacy OAuth2 credentials, which are being phased out.
	UpdateOAuth2CredentialTokenFunc func(ctx context.Context, credentialID string, data []byte) error
)

const (
	// SignHeader sign request by header
	SignHeader tokenSignMethodType = "headerSign"
)

var (
	// ErrInvalidAuthorizer invalid authorizer when create authorizer
	ErrInvalidAuthorizer = errors.New("authorizer need credential and connection")

	// ErrInvalidCredentialType invalid credential type
	ErrInvalidCredentialType = errors.New("credential type is not supported")

	// ErrInvalidSignMethod invalid sign method
	ErrInvalidSignMethod = errors.New("sign method is not supported")

	// ErrCustomCredentialNotContainsAccessToken when a custom credential connection call GetAccessToken
	ErrCustomCredentialNotContainsAccessToken = errors.New("custom credential not contains access token")
)

type (
	signOpt struct {
		tokenSignMethod           *TokenSignMethod
		updateCredentialTokenFunc UpdateOAuth2CredentialTokenFunc
		oauthCredentialUpdater    OAuthCredentialUpdater
		ctx                       context.Context
	}

	// WithOpt is option for signaturer.
	WithOpt func(credential *model.Credential, opt *signOpt)
)

const (
	headerKeyProperty = "headerKey"
)

// WithRequestSignMethod set the sign method for request.
// this option only for accessTokenAuthorizer.
func WithRequestSignMethod(m *TokenSignMethod) WithOpt {
	return func(credential *model.Credential, opt *signOpt) {
		if credential.Type != model.CredentialTypeAccessToken {
			return
		}

		opt.tokenSignMethod = m
	}
}

// WithUpdateCredentialTokenFunc set the function for update credential connection data when oauth2 token refreshed.
// this option only for oauth2Authorizer.
func WithUpdateCredentialTokenFunc(f UpdateOAuth2CredentialTokenFunc) WithOpt {
	return func(credential *model.Credential, opt *signOpt) {
		if credential.Type != model.CredentialTypeOAuth2 {
			return
		}

		opt.updateCredentialTokenFunc = f
	}
}

func WithOAuthCredentialUpdater(f OAuthCredentialUpdater) WithOpt {
	return func(credential *model.Credential, opt *signOpt) {
		opt.oauthCredentialUpdater = f
	}
}

// WithContext provide a context.
func WithContext(ctx context.Context) WithOpt {
	return func(credential *model.Credential, opt *signOpt) {
		opt.ctx = ctx
	}
}

// NewTokenSignMethod create a header token method, headerKey will set to header.
func NewTokenSignMethod(headerKey string) *TokenSignMethod {
	return &TokenSignMethod{
		methodType: SignHeader,
		properties: map[string]string{
			headerKeyProperty: headerKey,
		},
	}
}

func newDefaultSignOpt() *signOpt {
	return &signOpt{
		tokenSignMethod: defaultTokenSignMethod,
	}
}

var defaultTokenSignMethod = &TokenSignMethod{
	methodType: SignHeader,
	properties: map[string]string{
		headerKeyProperty: "AccessToken",
	},
}

// NewAuthorizer create authorizer from credential and credential connection.
// authorizer depends on adapter credential form
func NewAuthorizer(cipher crypto.CryptoCipher, credential *model.Credential, fns ...WithOpt) (Authorizer, error) {
	if credential == nil {
		return nil, ErrInvalidAuthorizer
	}

	opt := newDefaultSignOpt()
	for _, f := range fns {
		f(credential, opt)
	}

	rawData, err := cipher.Decrypt(credential.Data)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential data: %w", err)
	}

	credData, err := service.ComposeCredentialData(credential.AdapterClass, string(credential.Type), rawData)
	if err != nil {
		return nil, err
	}

	baseAuthorizer := &baseAuthorizer{
		credential: credential,
		cipher:     cipher,
		credData:   credData,
	}

	if credential.Type == model.CredentialTypeAccessToken {
		return &accessTokenAuthorizer{
			baseAuthorizer: baseAuthorizer,
			signMethod:     opt.tokenSignMethod,
		}, nil
	}

	if credential.Type == model.CredentialTypeOAuth2 {
		return &oauth2Authorizer{
			baseAuthorizer:                  baseAuthorizer,
			updateOAuth2CredentialTokenFunc: opt.updateCredentialTokenFunc,
		}, nil
	}

	if credential.Type == model.CredentialTypeCustom {
		return &customAuthorizer{baseAuthorizer}, nil
	}

	if credential.Type == model.CredentialTypeOAuth {
		return &oauthAuthorizer{
			baseAuthorizer:    baseAuthorizer,
			credentialUpdater: opt.oauthCredentialUpdater,
		}, nil
	}

	return nil, ErrInvalidCredentialType
}

func (s *baseAuthorizer) DecodeMeta(meta interface{}) error {
	return json.Unmarshal(s.credData.MetaData, meta)
}

func (s *baseAuthorizer) CredentialType() model.CredentialType {
	return s.credential.Type
}

func (s *customAuthorizer) GetAccessToken(ctx context.Context) (string, error) {
	return "", ErrCustomCredentialNotContainsAccessToken
}

func (s *customAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return ErrCustomCredentialNotContainsAccessToken
}

func (s *accessTokenAuthorizer) GetAccessToken(context.Context) (string, error) {
	return s.credData.AccessToken, nil
}

func (s *accessTokenAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return nil
}

func (s *oauth2Authorizer) getAvailableToken(ctx context.Context) (token *model.OAuth2Token, err error) {
	token = &model.OAuth2Token{}
	err = decryptData(s.cipher, s.credential.Token, token)
	if err != nil {
		err = fmt.Errorf("unmarshal connection data: %w", err)
		return
	}
	originalTokenMetaData := token.MetaData
	oauth2Config := s.credData.GetOAuth2Config()
	tokenSource := oauth2Config.TokenSource(ctx, &token.Token)

	// if the latestToken is expired, try to refresh it by call TokenSource.RefreshToken
	var latestToken *oauth2.Token

	latestToken, err = tokenSource.Token()
	if err != nil {
		err = fmt.Errorf("get oauth2 latestToken: %w", err)
		return
	}

	// update latest oauth2 latestToken to database.
	if latestToken.AccessToken != token.AccessToken {
		// TODO(sword): go test for this

		// salesforce returns access token with zero expiry, add salesforce default expiry
		if latestToken.Expiry.IsZero() {
			latestToken.Expiry = time.Now().Add(2 * time.Hour)
		}

		token = &model.OAuth2Token{
			Token: *latestToken,
			// inherit the original meta data, link: https://jihulab.com/ultrafox/ultrafox/-/merge_requests/new?merge_request%5Bsource_branch%5D=fix%2Fslack_refresh_token
			MetaData: originalTokenMetaData,
		}

		var tokenBytes []byte
		tokenBytes, err = s.cipher.Marshal(token)
		if err != nil {
			return nil, fmt.Errorf("marshal oauth2 latestToken curOAuth2Token error: %w", err)
		}
		s.credential.Token = tokenBytes

		if s.updateOAuth2CredentialTokenFunc != nil {
			err = s.updateOAuth2CredentialTokenFunc(ctx, s.credential.ID, tokenBytes)
			if err != nil {
				return nil, fmt.Errorf("update credential connection curOAuth2Token error: %w", err)
			}
		}
	}

	return
}

func (s *oauth2Authorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	token, err := s.getAvailableToken(ctx)
	if err != nil {
		return fmt.Errorf("decoding oauth2 token meta data: %w", err)
	}
	return json.Unmarshal(token.MetaData, meta)
}

func (s *oauth2Authorizer) GetAccessToken(ctx context.Context) (string, error) {
	token, err := s.getAvailableToken(ctx)
	if err != nil {
		return "", errors.Wrap(err, "get oauth2 token")
	}
	return token.AccessToken, nil
}

func decryptData(cipher crypto.CryptoCipher, data []byte, dst interface{}) error {
	rawData, err := cipher.Decrypt(data)
	if err != nil {
		return errors.Wrap(err, "decrypt credential connection data")
	}
	err = json.Unmarshal(rawData, dst)
	if err != nil {
		return errors.Wrap(err, "unmarshal credential connection data")
	}
	return nil
}

type OAuthCredentialUpdater struct {
	DB    model.Operator
	Cache *cache.Cache
}

func (o oauthAuthorizer) GetAccessToken(ctx context.Context) (string, error) {
	return "", errors.New("not implemented deliberately")
}

func (o oauthAuthorizer) DecodeTokenMetaData(ctx context.Context, meta interface{}) error {
	return errors.New("not implemented deliberately")
}

var _ OAuthCredentialMetaUpdater = oauthAuthorizer{}

type OAuthCredentialMetaUpdater interface {
	UpdateCredentialMeta(ctx context.Context, fn func() (meta any, err error)) (err error)
}

func UpdateCredentialMetaLockName(credentialID string) string {
	return "oauth-refresh-" + credentialID
}

func (o oauthAuthorizer) UpdateCredentialMeta(ctx context.Context, fn func() (meta any, runErr error)) (err error) {
	err = o.credentialUpdater.Cache.WaitLockToRun(ctx, UpdateCredentialMetaLockName(o.credential.ID), 1*time.Minute, func() (err error) {
		meta, err := fn()
		if err != nil {
			err = fmt.Errorf("running meta generating callback: %w", err)
			return
		}
		marshaled, err := json.Marshal(meta)
		if err != nil {
			err = fmt.Errorf("marshaling meta into JSON: %w", err)
			return
		}
		encrypted, err := o.cipher.Encrypt(marshaled)
		if err != nil {
			err = fmt.Errorf("encrypting data: %w", err)
			return
		}
		err = o.credentialUpdater.DB.UpdateCredentialDataByID(ctx, o.credential.ID, encrypted)
		if err != nil {
			err = fmt.Errorf("updating credential by id: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("waiting for lock and run: %w", err)
		return
	}

	return
}
