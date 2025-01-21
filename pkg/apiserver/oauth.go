package apiserver

import (
	"database/sql"
	_ "embed" // embed OAuth callback page template
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"

	"github.com/xanzy/go-gitlab"

	"github.com/gin-gonic/gin"
	nanoID "github.com/matoous/go-nanoid/v2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/oauth"
	gitlabAdapter "jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/gitlab"
)

func (h *APIHandler) GitlabCredentialAuthorization(gitlabVendor model.PassportVendorName) func(c *gin.Context) {
	return func(c *gin.Context) {
		var (
			err error
			ctx = c.Request.Context()
		)
		defer func() {
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}
		}()
		c.SetSameSite(http.SameSiteLaxMode)

		passportVendor, exists := h.passportVendorLookup[gitlabVendor]
		if !exists {
			err = fmt.Errorf("passport vendor %q is not enabled", gitlabVendor)
			return
		}

		owner, err := bindObjectOwner(c)
		if err != nil {
			err = fmt.Errorf("binding object owner: %w", err)
			return
		}
		err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, owner, permission.CredentialWrite)
		if err != nil {
			err = fmt.Errorf("ensuring permissions: %w", err)
			return
		}

		root, badReferer := h.serverHost.IsInboundURL(c.Request.Referer())
		if badReferer != nil {
			root = h.serverHost.API()
		}
		redirectURL := fmt.Sprintf("%s/api/v1/credentials/oauth/callback/%s", root, strings.ToLower(string(passportVendor.Name)))

		state, err := nanoID.New()
		if err != nil {
			err = fmt.Errorf("generating state: %w", err)
			return
		}
		authCodeURL := oauth.GitlabAuthURL(oauth.GitlabAuthURLOpt{
			BaseURL:     passportVendor.BaseURL,
			ClientID:    passportVendor.ClientID,
			RedirectURL: redirectURL,
			State:       state,
			Scope:       []string{"api"},
		})

		c.SetCookie(oAuthStateCookieKey, state, beforeLoginStateMaxAgeSeconds, "", "", true, true)
		c.SetCookie(oAuthRedirectURLCookieKey, redirectURL, beforeLoginStateMaxAgeSeconds, "", "", true, true)
		if owner.OwnerType == model.OwnerTypeOrganization {
			c.SetCookie(oAuthResourceOrgIDKey, strconv.Itoa(owner.OwnerID), beforeLoginStateMaxAgeSeconds, "", "", true, true)
		}
		c.Redirect(http.StatusFound, authCodeURL)
	}
}

func (h *APIHandler) GitlabCredentialOAuthCallback(gitlabVendor model.PassportVendorName) func(c *gin.Context) {
	return func(c *gin.Context) {
		var (
			err              error
			ctx              = c.Request.Context()
			code             = c.Query("code")
			state            = c.Query("state")
			stateInCookie, _ = c.Cookie(oAuthStateCookieKey)
			redirectURL, _   = c.Cookie(oAuthRedirectURLCookieKey)
			orgID, _         = c.Cookie(oAuthResourceOrgIDKey)
			pagePayload      oAuthCallbackPayload
		)

		// always clear state
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(oAuthStateCookieKey, "", -1, "", "", true, true)
		c.SetCookie(oAuthRedirectURLCookieKey, "", -1, "", "", true, true)
		c.SetCookie(oAuthResourceOrgIDKey, "", -1, "", "", true, true)
		defer func() {
			if err != nil {
				_ = c.Error(err)
				c.Abort()
			}

			// Notify the frontend despite there's an error or not
			_ = oAuthCallbackPageTemplate.Execute(c.Writer, pagePayload)
		}()

		passportVendor, exists := h.passportVendorLookup[gitlabVendor]
		if !exists {
			err = fmt.Errorf("passport vendor %q is not enabled", gitlabVendor)
			return
		}

		if stateInCookie == "" {
			err = errors.New("no state in cookie")
			return
		}
		if state != stateInCookie {
			err = fmt.Errorf("state %q != stateInCookie %q", state, stateInCookie)
			return
		}
		if code == "" {
			// When a user denies the request, the callback URL looks like:
			// http://localhost:8080/api/v1/credentials/oauth/callback/jihulab?error=access_denied&error_description=The+resource+owner+or+authorization+server+denied+the+request.&state=V0po3n9IJYQD3n9jT_F4p
			err = fmt.Errorf("code is empty: %s", c.Query("error"))
			return
		}
		if redirectURL == "" {
			err = errors.New("redirect URL is empty")
			return
		}

		owner, err := bindObjectOwner(c)
		if err != nil {
			err = fmt.Errorf("binding object owner: %w", err)
			return
		}
		if orgID != "" {
			owner.OwnerID, err = strconv.Atoi(orgID)
			if err != nil {
				err = fmt.Errorf("parsing orgID %q: %w", orgID, err)
				return
			}
			owner.OwnerType = model.OwnerTypeOrganization
		}

		err = h.enforcer.EnsurePermissions(ctx, getSession(c).UserID, owner, permission.CredentialWrite)
		if err != nil {
			err = fmt.Errorf("ensuring permissions: %w", err)
			return
		}

		token, err := oauth.GitlabExchangeAccessToken(ctx, oauth.GitlabExchangeAccessTokenOpt{
			BaseURL:      passportVendor.BaseURL,
			ClientID:     passportVendor.ClientID,
			ClientSecret: passportVendor.ClientSecret,
			Code:         code,
			RedirectURL:  redirectURL,
		})
		if err != nil {
			err = fmt.Errorf("exchanging access token: %w", err)
			return
		}

		client, err := gitlab.NewOAuthClient(token.AccessToken, gitlab.WithBaseURL(passportVendor.BaseURL))
		if err != nil {
			err = fmt.Errorf("init Gitlab client: %w", err)
			return
		}
		vendorUser, _, err := client.Users.CurrentUser()
		if err != nil {
			err = fmt.Errorf("querying current user on Gitlab: %w", err)
			return
		}
		if vendorUser.ID == 0 {
			err = fmt.Errorf("the id of user %q is 0", vendorUser.Username)
			return
		}

		searchKey := fmt.Sprintf("%s-%d", passportVendor.Name, vendorUser.ID)

		userCredential, err := h.db.GetCredentialByOwnerAndSearchKey(ctx, owner, searchKey)
		if err == nil {
			// existed, update the data field
			meta := gitlabAdapter.GitlabOAuthMeta{
				Vendor:            passportVendor.Name,
				RedirectURL:       redirectURL,
				GitlabAccessToken: token,
			}

			var data []byte
			data, err = json.Marshal(meta)
			if err != nil {
				err = fmt.Errorf("marshaling meta into JSON: %w", err)
				return
			}
			data, err = h.cipher.Encrypt(data)
			if err != nil {
				err = fmt.Errorf("encrypting data: %w", err)
				return
			}
			err = h.db.UpdateCredentialDataByID(ctx, userCredential.ID, data)
			if err != nil {
				err = fmt.Errorf("updating new OAuth tokens into existed credential %s: %w", userCredential.ID, err)
				return
			}

			pagePayload = oAuthCallbackPayload{oAuthCallbackMessage{
				Completed:    true,
				CredentialID: userCredential.ID,
			}}

			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			err = fmt.Errorf("querying for existed Gitlab OAuth credential: %w", err)
			return
		}

		// A brand-new credential
		meta := gitlabAdapter.GitlabOAuthMeta{
			Vendor:            passportVendor.Name,
			RedirectURL:       redirectURL,
			GitlabAccessToken: token,
		}
		data, err := json.Marshal(meta)
		if err != nil {
			err = fmt.Errorf("marshaling meta into JSON: %w", err)
			return
		}
		data, err = h.cipher.Encrypt(data)
		if err != nil {
			err = fmt.Errorf("encrypting data: %w", err)
			return
		}

		userCredential = model.Credential{
			OwnerRef: owner,
			EditableCredential: model.EditableCredential{
				Name:         fmt.Sprintf("%s(@%s)", passportVendor.Name, vendorUser.Username),
				AdapterClass: "ultrafox/gitlab",
				Type:         model.CredentialTypeOAuth,
			},
			Status:    model.CredentialStatusAvailable,
			SearchKey: searchKey,
			Data:      data,
		}
		err = h.db.InsertCredential(ctx, &userCredential)
		if err != nil {
			err = fmt.Errorf("inserting new OAuth credential: %w", err)
			return
		}

		pagePayload = oAuthCallbackPayload{oAuthCallbackMessage{
			Completed:    true,
			CredentialID: userCredential.ID,
		}}
	}
}

//go:embed oauth_callback.gohtml
var oAuthCallbackPageTemplateRaw string
var oAuthCallbackPageTemplate *template.Template

func init() {
	var err error

	oAuthCallbackPageTemplate, err = template.New("").Funcs(map[string]any{
		"jsonify": func(v any) template.JS {
			marshaled, _ := json.Marshal(v)
			return template.JS(marshaled)
		},
	}).Parse(oAuthCallbackPageTemplateRaw)
	if err != nil {
		err = fmt.Errorf("building oAuthCallbackPageTemplate: %w", err)
		panic(err)
	}
}

type oAuthCallbackMessage struct {
	// Did the user complete the OAuth process?
	Completed bool `json:"completed"`
	// The ID of the related credential, e.g. qy87mh66xeo6kvr6
	CredentialID string `json:"credentialId,omitempty"`
}

type oAuthCallbackPayload struct {
	Msg oAuthCallbackMessage
}
