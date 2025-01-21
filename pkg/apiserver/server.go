package apiserver

import (
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/cache"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/work"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/serverhost"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/auth/crypto"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "jihulab.com/jihulab/ultrafox/ultrafox/docs/api_docs" // swagger docs
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/nanmu42/gzip"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
)

// ServerOpt options to start a new server
type ServerOpt struct {
	Port                int
	Logger              log.Logger
	DB                  *model.DB
	Cache               *cache.Cache
	TriggerRegistry     *trigger.Registry
	AuditResponse       bool
	EnableSwagger       bool
	EnableUI            bool
	FrontendAssetsPath  string
	PassportVendors     model.PassportVendors
	Cipher              crypto.CryptoCipher
	ServerHost          *serverhost.ServerHost
	OfficialCredentials model.OfficialCredentials
	MailSender          *smtp.Sender
	WorkProducer        *work.Producer
	BetaConfig          BetaConfig
	SwaggerConfig       SwaggerConfig
}

type SwaggerConfig struct {
	// whether to use HTTP basic auth to protect the swagger endpoints
	BasicAuth bool `comment:"whether to use HTTP basic auth to protect the swagger endpoints"`
	// username and password pairs. Required if BasicAuth is true.
	Accounts map[string]string `comment:"username and password pairs. Required if BasicAuth is true."`
}

// NewServer fires a new server
func NewServer(opt ServerOpt) (server *httpbase.GracefulServer, err error) {
	middleware := &httpbase.Middleware{
		Logger:         opt.Logger,
		AuditResponse:  opt.AuditResponse,
		UserIdentifier: userIdentifier,
	}
	router := newGin(middleware)

	if opt.EnableSwagger {
		err = registerSwaggerRouter(router, opt.SwaggerConfig)
		if err != nil {
			err = fmt.Errorf("enabling swagger endpoints: %w", err)
			return
		}
	}

	// health check
	router.GET("/healthz", func(c *gin.Context) {
		httpbase.SkipLogging(c)
		c.String(200, "OK")
	})

	if opt.EnableUI {
		router.NoRoute(httpbase.ServeFileWhenNotFound(opt.FrontendAssetsPath))
	}

	// robots.txt
	router.HEAD("/robots.txt", middleware.RobotsTXTHandler)
	router.GET("/robots.txt", middleware.RobotsTXTHandler)

	adapterManager := adapter.GetAdapterManager()
	adapterManager.FillDynamicFields(opt.ServerHost.APIFullURL("/api/v1/credentials/oauth2/callback"))
	// check each adapter has node implementation.
	checkAdaptersImplementation(adapterManager.GetMetas())

	// Web APIs
	webAPIController, err := newAPIHandler(APIHandlerOpt{
		DB:                        opt.DB,
		Cache:                     opt.Cache,
		Cipher:                    opt.Cipher,
		TriggerRegistry:           opt.TriggerRegistry,
		PassportVendors:           opt.PassportVendors,
		ServerHost:                opt.ServerHost,
		OfficialOAuth2Credentials: opt.OfficialCredentials,
		mailSender:                opt.MailSender,
		WorkProducer:              opt.WorkProducer,
		BetaConfig:                opt.BetaConfig,
	})
	if err != nil {
		err = fmt.Errorf("building webAPIController: %w", err)
		return
	}

	initAPIRouter(webAPIController, router)
	server = httpbase.NewGracefulServer(httpbase.GraceServerOpt{
		Logger: opt.Logger,
		Port:   opt.Port,
	}, router)

	return
}

// panics if an error is encountered.
func checkAdaptersImplementation(metas []*adapter.Meta) {
	for _, meta := range metas {
		for _, spec := range meta.Specs {
			class := spec.Class
			_, ok := workflow.GetNodeMeta(class)
			if !ok {
				panic(fmt.Errorf("spec %q don't has node implementation", class))
			}
		}
	}
}

func initAPIRouter(webAPIController *APIHandler, router *gin.Engine) {
	webAPI := router.Group("/api/v1")
	authWebAPI := router.Group("/api/v1", webAPIController.AuthMiddleware)
	// TODO(nanmu42): maybe put them in another controller? or not...?
	adminAPI := router.Group("/api/admin", webAPIController.AuthMiddleware, webAPIController.InsiderMiddleware)

	// meta
	webAPI.GET("/server/meta", webAPIController.ServerMeta)

	// login/logout
	authGroup := webAPI.Group("/auth")
	authGroup.GET("/status", webAPIController.LoginStatus)                 // current login status and user info
	authGroup.POST("/logout", webAPIController.Logout)                     // user signs out
	authGroup.PATCH("/beta/approveUser", webAPIController.BetaApproveUser) // beta only: approve user into private beta

	// OAuth Service OfficialCredential
	authGroup.POST("/login/jihulab", webAPIController.LoginViaJihulab)        // OAuth2 login via Jihulab
	authGroup.GET("/callback/jihulab", webAPIController.LoginCallbackJihulab) // OAuth2 login callback by Jihulab

	// workflows
	workflowGroup := authWebAPI.Group("/workflows")
	workflowGroup.GET("", webAPIController.ListWorkflow)                 // list workflows by page
	workflowGroup.POST("", webAPIController.CreateWorkflow)              // create workflow
	workflowGroup.PUT("/:id", webAPIController.UpdateWorkflow)           // update workflow
	workflowGroup.DELETE("/:id", webAPIController.DeleteWorkflow)        // delete workflow
	workflowGroup.GET("/log", webAPIController.ListAllWorkflowLog)       // list all workflow log by page
	workflowGroup.POST("/:id/enable", webAPIController.EnableWorkflow)   // enable workflow, set status to active
	workflowGroup.POST("/:id/disable", webAPIController.DisableWorkflow) // disable a workflow
	workflowGroup.POST("/:id/run", webAPIController.RunWorkflow)         // run a workflow by id
	workflowGroup.GET("/:id", webAPIController.GetWorkflow)              // get workflow detail
	workflowGroup.GET("/:id/extra", webAPIController.GetWorkflowExtra)   // get workflow extra information
	workflowGroup.GET("/:id/log", webAPIController.ListWorkflowLog)      // list workflow log by page
	workflowGroup.PUT("/apply", webAPIController.ApplyWorkflowYaml)      // apply a workflow yaml

	// nodes
	workflowGroup.POST("/:id/nodes", webAPIController.CreateNode)                               // create node
	workflowGroup.PUT("/:id/nodes/:nodeId", webAPIController.UpdateNode)                        // update node
	workflowGroup.PUT("/:id/nodes/:nodeId/transition", webAPIController.UpdateNodeTransition)   // update node transition
	workflowGroup.PUT("/:id/nodes/:nodeId/pathName", webAPIController.UpdateSwitchNodePathName) // update node transition
	workflowGroup.POST("/:id/nodes/:nodeId/run", webAPIController.RunNode)                      // run node
	workflowGroup.DELETE("/:id/nodes/:nodeId", webAPIController.DeleteNode)                     // delete node
	workflowGroup.GET("/:id/nodes/:nodeId/testPageData", webAPIController.GetNodeTestPageData)

	workflowGroup.GET("/:id/allNodeSamples", webAPIController.GetAllNodeSamples)                         // get all node samples
	workflowGroup.GET("/:id/nodes/:nodeId/samples", webAPIController.GetNodeSamples)                     // get node samples
	workflowGroup.POST("/:id/nodes/:nodeId/samples/loadMore", webAPIController.LoadMoreSamples)          // reload node samples
	workflowGroup.POST("/:id/nodes/:nodeId/samples/:sampleId/select", webAPIController.SelectNodeSample) // select a sample as testing context data
	workflowGroup.POST("/:id/nodes/:nodeId/skipTest", webAPIController.SkipTestNode)                     // generate a skip sample for node

	// share workflow
	shareGroup := workflowGroup.Group("/share")
	shareGroupNoAuth := webAPI.Group("/workflows/share")
	shareGroup.POST("/:id/export", webAPIController.ExportWorkflow)                     // export a workflow yaml
	shareGroup.POST("/import", webAPIController.ImportWorkflow)                         // import a workflow yaml
	shareGroup.POST("/:id/exportUrl", webAPIController.EnableOrCreateWorkflowShareLink) // export a workflow url
	shareGroup.PUT("/:id/exportUrl", webAPIController.ResetWorkflowShareLink)           // reset a workflow url
	shareGroup.GET("/:id/exportUrl", webAPIController.GetWorkflowShareLink)             // get a workflow sharing link
	shareGroup.DELETE("/:id/exportUrl", webAPIController.DisableWorkflowShareLink)      // disable a workflow sharing link

	// workflow import page
	shareGroupNoAuth.GET("/importUrl/:id/", webAPIController.BrowseWorkflowShareLink)     // first step of importing a workflow from sharing link
	shareGroup.POST("/importUrl/:id/accept", webAPIController.ImportWorkflowShareLink)    // import a workflow from a sharing link
	shareGroup.GET("/importUrl/:id/validate", webAPIController.ValidateWorkflowShareLink) // checks whether the workflow link is valid

	// Galaxy: workflow templates
	templateGroup := authWebAPI.Group("/templates")
	templateGroupNoAuth := webAPI.Group("/templates")
	templateGroupNoAuth.GET("/tags", webAPIController.GetTemplateTags)    // list template tags, categories, etc
	templateGroupNoAuth.POST("/search", webAPIController.SearchTemplates) // search templates for template list
	templateGroupNoAuth.GET("/:id", webAPIController.GetTemplate)         // get template detail by id
	templateGroup.POST("/:id/use", webAPIController.UseTemplate)          // use template by id
	templateGroup.GET("/byOwner", webAPIController.ListTemplatesByOwner)  // list user/org's templates, including all status
	templateGroup.POST("", webAPIController.PublishTemplate)              // publish a template, based on an existed workflow
	templateGroup.PATCH("/:id", webAPIController.UpdateTemplate)          // update a template by id, every field is required except workflowId
	templateGroup.DELETE("/:id", webAPIController.DeleteTemplate)         // delete a template by id

	// APIs to manage template tags
	templateAdmin := adminAPI.Group("/templates")
	templateAdmin.GET("/tags", webAPIController.AdminListTemplateTags)         // list all template tags with all fields
	templateAdmin.POST("/tags", webAPIController.AdminCreateTemplateTag)       // create a new template tag
	templateAdmin.PUT("/tags/:id", webAPIController.AdminUpdateTemplateTag)    // change the specified template tag
	templateAdmin.DELETE("/tags/:id", webAPIController.AdminDeleteTemplateTag) // delete the specified template tag, offering an optional supersedence.

	// confirms
	confirmGroup := authWebAPI.Group("/confirm")
	confirmGroup.GET("/:id", webAPIController.GetConfirm)              // fetch confirm by id
	confirmGroup.POST("/:id/decision", webAPIController.DecideConfirm) // decide on confirm

	// credentials
	credentialGroup := authWebAPI.Group("/credentials")
	credentialGroup.GET("", webAPIController.ListCredential)          // list all credentials
	credentialGroup.GET("/:id", webAPIController.GetCredential)       // get credential by id
	credentialGroup.POST("", webAPIController.CreateCredential)       // create credential
	credentialGroup.PUT("/:id", webAPIController.UpdateCredential)    // update credential
	credentialGroup.DELETE("/:id", webAPIController.DeleteCredential) // delete credential
	credentialGroup.GET("/:id/associatedWorkflows", webAPIController.ListAssociatedWorkflows)

	// to obtain accessToken from third-party.
	// DEPRECATED(nanmu42): use /api/v1/credentials/oauth/* instead
	webAPI.GET("/credentials/oauth2/callback", webAPIController.CredentialOAuth2Callback) // callback for OAuth
	authWebAPI.POST("/credentials/oauth2/authUrl", webAPIController.RequestAuthURL)       // request oauth2 auth url

	// Enable user authorization though Ultrafox-hosted OAuth APPs
	credentialOAuth := credentialGroup.Group("/oauth")
	credentialOAuthAuthorize := credentialOAuth.Group("/authorize")
	credentialOAuthCallback := credentialOAuth.Group("/callback")

	credentialOAuthAuthorize.GET("/gitlab", webAPIController.GitlabCredentialAuthorization(model.PassportVendorGitlab))
	credentialOAuthCallback.GET("/gitlab", webAPIController.GitlabCredentialOAuthCallback(model.PassportVendorGitlab))
	credentialOAuthAuthorize.GET("/jihulab", webAPIController.GitlabCredentialAuthorization(model.PassportVendorJihulab))
	credentialOAuthCallback.GET("/jihulab", webAPIController.GitlabCredentialOAuthCallback(model.PassportVendorJihulab))

	// adapters
	adapterGroup := authWebAPI.Group("/adapters")
	adapterGroupNoAuth := webAPI.Group("/adapters")
	adapterGroupNoAuth.GET("", webAPIController.GetAdapterList) // list all adapters
	adapterGroup.POST("/fieldSelect", webAPIController.QueryFieldSelect)

	// workflow execution history
	workflowInstGroup := authWebAPI.Group("/workflowInstances")
	workflowInstGroup.GET("/:id/detail", webAPIController.GetWorkflowInstanceDetail)
	workflowInstGroup.DELETE("/:id", webAPIController.DeleteWorkflowInstance)

	// organizations
	orgGroup := authWebAPI.Group("/orgs")
	orgGroupNoAuth := webAPI.Group("/orgs")
	orgGroup.GET("/joined", webAPIController.ListJoinedOrgs)              // list the orgs that current user has joined
	orgGroup.GET("/joined/role", webAPIController.ListJoinedOrgsAndRoles) // list the orgs that current user has joined and the user's role in the orgs
	orgGroup.POST("", webAPIController.CreateOrg)                         // user creates new org
	orgGroup.GET("/:id", webAPIController.GetOrgByID)                     // get org detail by its id, where the org must be visible to the user
	orgGroup.PUT("/:id", webAPIController.UpdateOrgByID)                  // update org attributes
	orgGroup.DELETE("/:id", webAPIController.DeleteOrgByID)               // delete an org

	// organization members
	orgGroup.GET("/:id/members", webAPIController.ListOrgMembers)                   // list org members by org id, where the org must be visible to the user
	orgGroup.POST("/:id/members/findByIds", webAPIController.ListOrgMembersByIDs)   // list org members by org id and user ids, where the org must be visible to the user
	orgGroup.GET("/:id/members/search", webAPIController.SearchOrgMembers)          // search org members by org id and username and emails, where the org must be visible to the user
	orgGroup.GET("/:id/inviteLinks", webAPIController.GetOrgInviteLinks)            // get invite links of the org for all roles
	orgGroup.PUT("/:id/inviteLinks", webAPIController.ResetOrgInviteLinks)          // reset invite links of the org for all roles
	orgGroup.DELETE("/:id/inviteLinks", webAPIController.DeleteOrgInviteLinks)      // delete all invite links of the org
	orgGroup.PUT("/:id/members/:userId/role", webAPIController.ChangeOrgMemberRole) // change a specified member's role in the org
	orgGroup.DELETE("/:id/members", webAPIController.RemoveMembersFromOrg)          // remove members from the org
	orgGroup.DELETE("/:id/members/me", webAPIController.LeaveOrg)                   // a member leaves org by themselves.

	// organization invite page
	orgGroupNoAuth.GET("/inviteLinks/:id", webAPIController.BrowseOrgInvite)     // a user browses an org invitation
	orgGroup.GET("/inviteLinks/:id/validation", webAPIController.ValidateInvite) // check if current user can join the organization via the invite link
	orgGroup.POST("/inviteLinks/:id/accept", webAPIController.AcceptOrgInvite)   // a user accept an org invitation

	// statistics
	webAPI.GET("/statistics", webAPIController.Statistics)

	// user tour
	tourGroup := authWebAPI.Group("/tour")
	tourGroup.GET("", webAPIController.GetUserTour)
	tourGroup.POST("", webAPIController.CreateUserTour)
	tourGroup.PUT("", webAPIController.UpdateUserTour)
	tourGroup.DELETE("/:id", webAPIController.DeleteUserTour)

	tourConfigGroup := authWebAPI.Group("/tourconfig")
	tourConfigGroup.GET("", webAPIController.GetTourConfigs)
	tourConfigGroup.POST("", webAPIController.CreateTourConfig)
	tourConfigGroup.PUT("/:id/config", webAPIController.UpdateTourConfig)
	tourConfigGroup.POST("/:id/step", webAPIController.CreateTourScopeStep)
	tourConfigGroup.PUT("/:id/step", webAPIController.UpdateTourStepConfigElementID)
	tourConfigGroup.DELETE("/:id/config", webAPIController.DeleteTourConfigByID)

	router.POST("/internal/test/featureFlag", TestFeatureFlag)
}

// newGin get you a glass of gin, flavored
func newGin(middleware *httpbase.Middleware) (g *gin.Engine) {
	const maxRequestBodySize = 256 << 10

	g = gin.New()

	g.ForwardedByClientIP = true
	g.HandleMethodNotAllowed = true
	g.NoMethod(middleware.MethodNotAllowedHandler)
	g.NoRoute(middleware.NotFoundHandler)

	g.Use(
		otelgin.Middleware("apiserver"),
		middleware.Recovery,
		gzip.DefaultHandler().Gin,
		middleware.LimitRequestBody(maxRequestBodySize),
		middleware.RequestLog,
		middleware.Error,
	)

	return
}

// @title UltraFox API Doc
// @version v1
// @contact.name ultrafox API Support
// @contact.url http://jihulab.com
// @contact.email support@jihulab.io
// @description API Doc
func registerSwaggerRouter(router *gin.Engine, swaggerConfig SwaggerConfig) (err error) {
	if !swaggerConfig.BasicAuth {
		router.GET("/swagger/v1/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("v1")))

		return
	}

	if len(swaggerConfig.Accounts) == 0 {
		err = errors.New("swagger basic auth: accounts is required for HTTP basic auth")
		return
	}

	router.GET("/swagger/v1/*any",
		gin.BasicAuthForRealm(swaggerConfig.Accounts, "Restricted endpoints. Please provide credential."),
		ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("v1")),
	)

	return
}
