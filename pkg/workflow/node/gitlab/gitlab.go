package gitlab

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/oauth"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/node/common"

	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var (
	className    = "ultrafox/gitlab"
	adapterClass = common.AdapterClass(className)
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	registerAdapter()

	workflow.RegistryNodeMeta(&IssueTrigger{})
	workflow.RegistryNodeMeta(&NoteTrigger{})
	workflow.RegistryNodeMeta(&MergeRequestTrigger{})
	workflow.RegistryNodeMeta(&PushTrigger{})
	workflow.RegistryNodeMeta(&TagTrigger{})
	workflow.RegistryNodeMeta(&ReleaseTrigger{})
	workflow.RegistryNodeMeta(&JobTrigger{})
	workflow.RegistryNodeMeta(&PipelineTrigger{})
	workflow.RegistryNodeMeta(&MemberTrigger{}) // Hidden, todo(sword): gitlab webhook api don't support.

	workflow.RegistryNodeMeta(&AddProjectIssueLabel{})
	workflow.RegistryNodeMeta(&ListGroupIssue{})
	workflow.RegistryNodeMeta(&IssueLink{})
	workflow.RegistryNodeMeta(&CommentProjectIssue{})
	workflow.RegistryNodeMeta(&CloseProjectIssue{})
	workflow.RegistryNodeMeta(&CreateProjectIssue{})
	workflow.RegistryNodeMeta(&UpdateProjectIssue{})
	workflow.RegistryNodeMeta(&CreateEpic{})
	workflow.RegistryNodeMeta(&CreateGroupLabel{})
	workflow.RegistryNodeMeta(&UpdateEpic{})
	workflow.RegistryNodeMeta(&CreateGroupMilestone{})
	workflow.RegistryNodeMeta(&ListGroupEpic{})
	workflow.RegistryNodeMeta(&ListProject{})
	workflow.RegistryNodeMeta(&ListUser{})
	workflow.RegistryNodeMeta(&ListIssue{})
	workflow.RegistryNodeMeta(&GetProjectMergeRequest{})
	workflow.RegistryNodeMeta(&GetProjectIssue{})
	workflow.RegistryNodeMeta(&UpdateProjectMergeRequest{})
	workflow.RegistryNodeMeta(&ListProjectMergeRequests{})
	workflow.RegistryNodeMeta(&CommentProjectMergeRequest{})
	workflow.RegistryNodeMeta(&ListGroupMergeRequests{})
	workflow.RegistryNodeMeta(&ListGroup{})
	workflow.RegistryNodeMeta(&ListGroupMilestone{})
	workflow.RegistryNodeMeta(&ListProjectMilestone{})
	workflow.RegistryNodeMeta(&RunPipeline{})
	workflow.RegistryNodeMeta(&ListGroupMembers{})
	workflow.RegistryNodeMeta(&ListMergeRequestsRelatedToIssue{})
	workflow.RegistryNodeMeta(&UpdateRelease{})
}

func registerAdapter() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	adapterMeta.RegisterCredentialTemplate(adapter.AccessTokenCredentialType, `{
	"metaData": {
		"server": "{{ .server }}"
	},
	"accessToken": "{{ .accessToken }}"
}`)

	adapterMeta.RegisterCredentialTestingFunc(testCredential)
}

func testCredential(ctx context.Context, credentialType model.CredentialType, inputFields model.InputFields) (err error) {
	if credentialType != model.CredentialTypeAccessToken {
		// Validation is only required for the accessToken credential, so ensure that it is successfully validated.
		return
	}

	accessToken, ok := inputFields.GetString2("accessToken", true)
	if !ok {
		err = fmt.Errorf("accessToken is required")
		return
	}
	server, ok := inputFields.GetString2("server", true)
	if !ok {
		err = fmt.Errorf("server is required")
		return
	}

	var client *gitlab.Client
	client, err = gitlab.NewClient(accessToken, gitlab.WithBaseURL(server), gitlab.WithHTTPClient(http.DefaultClient))
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	_, resp, err := client.Version.GetVersion(gitlab.WithContext(ctx))
	if err != nil {
		err = fmt.Errorf("getting gitlab version: %w", err)
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	}
	return
}

type BaseGitlabNode struct {
	client *gitlab.Client
}

func (g *BaseGitlabNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) error {
	if dependencies.PassportVendorLookup == nil {
		return errors.New("dependencies.PassportVendorLookup is nil")
	}

	client, err := newClient(ctx, dependencies.Authorizer, dependencies.PassportVendorLookup)
	if err != nil {
		return err
	}
	g.client = client
	return nil
}

func (g *BaseGitlabNode) GetClient() *gitlab.Client {
	return g.client
}

type gitlabAuthMeta struct {
	Server string `json:"server"`
}

type GitlabOAuthMeta struct {
	Vendor      model.PassportVendorName `json:"vendor"`
	RedirectURL string                   `json:"redirect_url"`

	oauth.GitlabAccessToken
}

func newClient(ctx context.Context, authorizer auth.Authorizer, passportVendorLookup map[model.PassportVendorName]model.PassportVendor) (client *gitlab.Client, err error) {
	if authorizer == nil {
		return nil, errors.New("authorizer is nil")
	}

	credentialType := authorizer.CredentialType()
	switch credentialType {
	case model.CredentialTypeOAuth2:
		err = errors.New("support for legacy OAuth2 has been dropped, please use the new OAuth credential instead")
		return
	case model.CredentialTypeOAuth, model.CredentialTypeAccessToken:
		// relax
	default:
		err = fmt.Errorf("unexpected credential type %s", credentialType)
		return
	}

	if credentialType == model.CredentialTypeAccessToken {
		var accessToken string
		accessToken, err = authorizer.GetAccessToken(ctx)
		if err != nil {
			err = fmt.Errorf("getting access token: %w", err)
			return
		}
		meta := &gitlabAuthMeta{}
		err = authorizer.DecodeMeta(meta)
		if err != nil {
			err = fmt.Errorf("decoding auth meta data: %w", err)
			return
		}
		client, err = gitlab.NewClient(accessToken, gitlab.WithBaseURL(meta.Server))

		return
	}

	var oAuthMeta GitlabOAuthMeta
	err = authorizer.DecodeMeta(&oAuthMeta)
	if err != nil {
		err = fmt.Errorf("decoding auth meta data: %w", err)
		return
	}

	passportVendor, ok := passportVendorLookup[oAuthMeta.Vendor]
	if !ok {
		err = fmt.Errorf("vendor %q does not show up in passportVendorLookup", oAuthMeta.Vendor)
		return
	}

	if oAuthMeta.ExpiresAt().Add(-5 * time.Minute).After(time.Now()) {
		client, err = gitlab.NewOAuthClient(oAuthMeta.AccessToken, gitlab.WithBaseURL(passportVendor.BaseURL))
		return
	}

	updater := authorizer.(auth.OAuthCredentialMetaUpdater)

	err = updater.UpdateCredentialMeta(ctx, func() (meta any, err error) {
		// expiring or expired token, we need to refresh it.
		token, err := oauth.GitlabRefreshAccessToken(ctx, oauth.GitlabRefreshAccessTokenOpt{
			BaseURL:      passportVendor.BaseURL,
			ClientID:     passportVendor.ClientID,
			ClientSecret: passportVendor.ClientSecret,
			RefreshToken: oAuthMeta.RefreshToken,
			RedirectURL:  oAuthMeta.RedirectURL,
		})
		if err != nil {
			err = fmt.Errorf("refreshing user's access token: %w", err)
			return
		}
		oAuthMeta.GitlabAccessToken = token
		meta = oAuthMeta

		return
	})
	if err != nil {
		err = fmt.Errorf("updating OAuth credential meta: %w", err)
		return
	}

	client, err = gitlab.NewOAuthClient(oAuthMeta.AccessToken, gitlab.WithBaseURL(passportVendor.BaseURL))
	return
}
