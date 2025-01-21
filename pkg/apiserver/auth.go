package apiserver

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed" // import beta user approval email template
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	nanoID "github.com/matoous/go-nanoid/v2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/oauth"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/httpbase"

	"github.com/gin-gonic/gin"
	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

//go:embed beta_user_approval_email.gohtml
var rawBetaUserApprovalTemplate string

var betaUserApprovalTemplate *template.Template

func init() {
	var err error

	betaUserApprovalTemplate, err = template.New("").Parse(rawBetaUserApprovalTemplate)
	if err != nil {
		err = fmt.Errorf("parsing rawBetaUserApprovalTemplate: %w", err)
		panic(err)
	}
}

const (
	ctxSessionKey = "session"
	// sessionCookieKey is where session cookie lies on browsers
	sessionCookieKey = "fox_sess"
	// sessionMaxAgeSeconds how long will a session cookie live on browsers
	sessionMaxAgeSeconds = 14 * 24 * 60 * 60
	// oAuthStateCookieKey keeps OAuth state on browsers
	oAuthStateCookieKey = "fox_oauth_state"
	// oAuthRedirectURLCookieKey keeps OAuth redirect URL on browsers
	oAuthRedirectURLCookieKey = "fox_oauth_redirect_url"
	// oAuthResourceOrgIDKey keeps resource orgID on browsers
	oAuthResourceOrgIDKey = "fox_oauth_resource_org_id"
	// loginContinueURLCookieKey keeps login continue URL on browsers
	loginContinueURLCookieKey = "fox_continue_url"
	// beforeLoginStateMaxAgeSeconds how long OAuth state and before-login states lives on browsers
	beforeLoginStateMaxAgeSeconds = 10 * 60
)

// getSession extracts session from Gin's context.
// Caller must ensure that context has session attached.
// Refer to APIHandler.AuthMiddleware.
func getSession(c *gin.Context) model.Session {
	value, exists := c.Get(ctxSessionKey)
	if !exists {
		panic("no session attached on the context")
	}

	return value.(model.Session)
}

// userIdentifier is used in httpbase.Middleware
func userIdentifier(c *gin.Context) httpbase.UserIdentity {
	value, exists := c.Get(ctxSessionKey)
	if !exists {
		return httpbase.UserIdentity{}
	}

	session := value.(model.Session)
	return httpbase.UserIdentity{
		ID: session.UserID,
	}
}

func (h *APIHandler) takeValidSessionFromCookie(ctx context.Context, c *gin.Context) (session model.Session, err error) {
	sessionID, err := c.Cookie(sessionCookieKey)
	if err != nil {
		err = fmt.Errorf("parsing session cookie: %w", err)
		return
	}

	if sessionID == "" {
		err = errors.New("session id is empty")
		return
	}

	session, err = h.db.GetSessionByID(ctx, sessionID)
	if err != nil {
		err = fmt.Errorf("querying session by ID %q: %w", sessionID, err)
		return
	}

	if session.Status != model.SessionStatusOK {
		err = fmt.Errorf("session status is %q, want %q", session.Status, model.SessionStatusOK)
		return
	}
	if session.ExpiredAt.Before(time.Now()) {
		err = fmt.Errorf("session has expired since %s", session.ExpiredAt.Format(time.RFC3339))
		return
	}

	return
}

// bindObjectOwner looks into request's orgId query string
// and produces corresponding object ownerRef.
// if orgID is empty, it returns ownerRef of current user themselves.
func bindObjectOwner(c *gin.Context) (objectOwner model.OwnerRef, err error) {
	rawOrgID := c.Query("orgId")
	if rawOrgID == "" {
		objectOwner = model.OwnerRef{
			OwnerType: model.OwnerTypeUser,
			OwnerID:   getSession(c).UserID,
		}

		return
	}

	orgID, err := strconv.Atoi(rawOrgID)
	if err != nil {
		err = fmt.Errorf("binding orgID: %w", err)
		return
	}
	if orgID <= 0 {
		err = fmt.Errorf("invalid orgID, got %d", orgID)
		return
	}

	objectOwner = model.OwnerRef{
		OwnerType: model.OwnerTypeOrganization,
		OwnerID:   orgID,
	}

	return
}

// AuthMiddleware ensures request has a valid session
// and attaches user-related info for downstream.
func (h *APIHandler) AuthMiddleware(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)

	var (
		err error
		ctx = c.Request.Context()
	)
	defer func() {
		if err != nil {
			// clear the session cookie
			c.SetCookie(sessionCookieKey, "", -1, "", "", true, true)
			_ = c.Error(err)
			_ = c.Error(errUnauthorized)
			c.Abort()
		}
	}()

	session, err := h.takeValidSessionFromCookie(ctx, c)
	if err != nil {
		return
	}

	// all is well
	c.Set(ctxSessionKey, session)
	c.Next()
}

// LoginStatus returns current login status and user info.
// It does not need AuthMiddleware.
// @Summary returns current login status and user info
// @Description returns current login status and user info.
// @Accept json
// @Produce json
// @Success 200 {object} apiserver.R{data=response.LoginStatusResp}
// @Router /api/v1/auth/status [get]
func (h *APIHandler) LoginStatus(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)

	c.SetSameSite(http.SameSiteLaxMode)
	defer func() {
		if err != nil {
			// clear the session cookie
			c.SetCookie(sessionCookieKey, "", -1, "", "", true, true)
			// discard any error
			err = nil
			OK(c, response.LoginStatusResp{})
		}
	}()

	session, err := h.takeValidSessionFromCookie(ctx, c)
	if err != nil {
		return
	}
	user, err := h.db.GetUserByID(ctx, session.UserID)
	if err != nil {
		err = fmt.Errorf("querying user by id %d: %w", session.UserID, err)
		return
	}

	OK(c, response.LoginStatusResp{
		SignedIn: true,
		User:     &user,
	})
}

// Logout signs out current user.
// It does not need AuthMiddleware.
// @Summary signs out current user
// @Description signs out current user.
// @Accept json
// @Produce json
// @Success 302
// @Failure 200 {object} apiserver.R
// @Router /api/v1/auth/logout [post]
func (h *APIHandler) Logout(c *gin.Context) {
	var (
		err error
		ctx = c.Request.Context()
	)

	c.SetSameSite(http.SameSiteLaxMode)
	defer func() {
		// clear the session cookie
		c.SetCookie(sessionCookieKey, "", -1, "", "", true, true)

		if err != nil {
			_ = c.Error(err)
			_ = c.Error(errInternal)
			c.Abort()
		}
	}()

	sessionID, _ := c.Cookie(sessionCookieKey)
	if sessionID != "" {
		err = h.db.DeleteSessionByID(ctx, sessionID)
		if err != nil {
			err = fmt.Errorf("delete session by id %q: %w", sessionID, err)
			return
		}
	}

	c.Redirect(http.StatusFound, "/")
}

// LoginViaJihulab redirects user to Jihulab OAuth login page
// It does not need AuthMiddleware.
// @Summary redirects user to Jihulab OAuth login page
// @Description redirects user to Jihulab OAuth login page.
// @Param   continueUrl formData string false "URL to go when the login is succeeded"
// @Success 302
// @Failure 200 {object} apiserver.R
// @Router /api/v1/auth/login/jihulab [post]
func (h *APIHandler) LoginViaJihulab(c *gin.Context) {
	var (
		err         error
		continueURL = c.PostForm("continueUrl") // let's make our frontend college's life easier.
	)

	c.SetSameSite(http.SameSiteLaxMode)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			_ = c.Error(errInternal)
			c.Abort()
		}
	}()

	passportVendor, exists := h.passportVendorLookup[model.PassportVendorJihulab]
	if !exists {
		err = fmt.Errorf("passport vendor %q is disabled", model.PassportVendorJihulab)
		return
	}
	var root string

	if continueURL != "" {
		root, err = h.serverHost.IsInboundURL(continueURL)
		if err != nil {
			err = fmt.Errorf("checking is continueURL inbound: %w", err)
			return
		}
	}
	if root == "" {
		root = h.serverHost.API()
	}
	if continueURL == "" {
		c.SetCookie(loginContinueURLCookieKey, "", -1, "", "", true, true)
	} else {
		c.SetCookie(loginContinueURLCookieKey, continueURL, beforeLoginStateMaxAgeSeconds, "", "", true, true)
	}

	state, err := nanoID.New()
	if err != nil {
		err = fmt.Errorf("generating state: %w", err)
		return
	}
	authCodeURL := oauth.GitlabAuthURL(oauth.GitlabAuthURLOpt{
		BaseURL:     "https://jihulab.com",
		ClientID:    passportVendor.ClientID,
		RedirectURL: root + "/api/v1/auth/callback/jihulab",
		State:       state,
		Scope:       []string{"read_user"},
	})

	c.SetCookie(oAuthStateCookieKey, state, beforeLoginStateMaxAgeSeconds, "", "", true, true)
	c.Redirect(http.StatusFound, authCodeURL)
}

var insiderEmailSuffix = []string{
	"@jihulab.com",
	"@gitlab.cn",
}

func isUserInsider(email string) bool {
	for _, emailSuffix := range insiderEmailSuffix {
		if strings.HasSuffix(email, emailSuffix) {
			return true
		}
	}
	return false
}

// isLDAPDummyEmailAddress When a user use LDAP and does not provide an email address to Gitlab,
// their email is a dummy and looks like temp-email-for-oauth-[username]@gitlab.localhost
// We must reject such mail addresses.
// Ref: https://gitlab.com/gitlab-org/gitlab/-/issues/209047
func isLDAPDummyEmailAddress(email string) bool {
	return strings.HasSuffix(email, "@gitlab.localhost") || strings.HasPrefix(email, "temp-email-for-oauth-")
}

// InsiderMiddleware ensures current use is an insider.
// It relies on AuthMiddleware.
func (h *APIHandler) InsiderMiddleware(c *gin.Context) {
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

	user, err := h.db.GetUserByID(ctx, getSession(c).UserID)
	if err != nil {
		_ = c.Error(fmt.Errorf("querying user: %w", err))
		err = errUnauthorized
		return
	}

	if !isUserInsider(user.Email) {
		_ = c.Error(fmt.Errorf("email %q of user %d does not meet insider criteria", user.Email, user.ID))
		err = errUnauthorized
		return
	}

	// All is well. Proceed.
	c.Next()
}

// LoginCallbackJihulab handles Jihulab's OAuth2 login callback
// and signs the user in.
// It does not need AuthMiddleware.
func (h *APIHandler) LoginCallbackJihulab(c *gin.Context) {
	var (
		err               error
		ctx               = c.Request.Context()
		code              = c.Query("code")
		state             = c.Query("state")
		stateInCookie, _  = c.Cookie(oAuthStateCookieKey)
		continueURL, _    = c.Cookie(loginContinueURLCookieKey)
		exposeErr         bool
		alreadyRedirected bool
	)

	// always clear state
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(oAuthStateCookieKey, "", -1, "", "", true, true)
	c.SetCookie(loginContinueURLCookieKey, "", -1, "", "", true, true)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			if !exposeErr {
				_ = c.Error(errUnauthorized)
			}

			c.Abort()
			return
		}

		if !alreadyRedirected {
			c.Redirect(http.StatusFound, continueURL)
		}
	}()

	if stateInCookie == "" {
		err = errors.New("no state in cookie")
		return
	}
	if state != stateInCookie {
		err = fmt.Errorf("state %q != stateInCookie %q", state, stateInCookie)
		return
	}
	if code == "" {
		err = errors.New("code is empty")
		return
	}

	passportVendor, exists := h.passportVendorLookup[model.PassportVendorJihulab]
	if !exists {
		err = fmt.Errorf("passport vendor %q is disabled", model.PassportVendorJihulab)
		return
	}

	var root string

	if continueURL != "" {
		root, err = h.serverHost.IsInboundURL(continueURL)
		if err != nil {
			err = fmt.Errorf("checking is continueURL inbound: %w", err)
			return
		}
	} else {
		continueURL = "/"
	}

	if root == "" {
		root = h.serverHost.API()
	}

	token, err := oauth.GitlabExchangeAccessToken(ctx, oauth.GitlabExchangeAccessTokenOpt{
		BaseURL:      "https://jihulab.com",
		ClientID:     passportVendor.ClientID,
		ClientSecret: passportVendor.ClientSecret,
		Code:         code,
		RedirectURL:  root + "/api/v1/auth/callback/jihulab",
	})
	if err != nil {
		err = fmt.Errorf("exchanging access token using OAuth code: %w", err)
		return
	}

	client, err := gitlab.NewOAuthClient(token.AccessToken, gitlab.WithBaseURL("https://jihulab.com"))
	if err != nil {
		err = fmt.Errorf("init Gitlab client: %w", err)
		return
	}
	jihulabUser, _, err := client.Users.CurrentUser()
	if err != nil {
		err = fmt.Errorf("lookup current user on Jihulab: %w", err)
		return
	}
	if jihulabUser.ID == 0 {
		err = fmt.Errorf("the id of user %q is 0", jihulabUser.Username)
		return
	}
	if jihulabUser.ConfirmedAt == nil {
		err = fmt.Errorf("user %q(id: %d) is not confirmed on Jihulab", jihulabUser.Username, jihulabUser.ID)
		exposeErr = true
		return
	}

	const gitlabUserStateActive = "active"
	if jihulabUser.State != gitlabUserStateActive {
		err = fmt.Errorf("the state of user %q(id: %d) is %q, want %q", jihulabUser.Username, jihulabUser.ID, jihulabUser.State, gitlabUserStateActive)
		exposeErr = true
		return
	}

	if isLDAPDummyEmailAddress(jihulabUser.Email) {
		err = errors.New(`it seems that you are using LDAP to log in GitLab. To use our service, you must set your email address properly and delete the email address "temp-email-for-oauth-yourname@gitlab.localhost" at https://jihulab.com/-/profile/emails`)
		exposeErr = true
		return
	}

	extUID := strconv.Itoa(jihulabUser.ID)
	passport, err := h.db.GetPassportWithUserByExtUID(ctx, model.PassportVendorJihulab, extUID)
	if err == nil {
		switch passport.User.Status {
		case model.UserStatusOK:
			// existed allowed user, log them in
			var session model.Session
			session, err = h.logUserIn(ctx, passport.UserID)
			if err != nil {
				err = fmt.Errorf("logging existed user in: %w", err)
				return
			}
			c.SetCookie(sessionCookieKey, session.ID, sessionMaxAgeSeconds, "", "", true, true)

			return
		case model.UserStatusPendingBetaInvitation:
			// redirect pending-invitation users to the beta sign-up collecting sheet
			c.Redirect(http.StatusFound, h.betaConfig.InvitationSignUpSheetURLWithEmail(jihulabUser.Email))
			alreadyRedirected = true
			return
		default:
			err = fmt.Errorf("unexpected user status %q of user %d", passport.User.Status, passport.UserID)
			return
		}

	}
	if !errors.Is(err, sql.ErrNoRows) {
		// other types of errors
		err = fmt.Errorf("querying passport: %w", err)
		return
	}

	// new user, register them and log them in if applicable
	isUserPreallowed := isUserInsider(jihulabUser.Email)

	var userStatus model.UserStatus
	if isUserPreallowed {
		userStatus = model.UserStatusOK
	} else {
		userStatus = model.UserStatusPendingBetaInvitation
	}

	newUser := model.User{
		Status:    userStatus,
		Email:     jihulabUser.Email,
		Name:      jihulabUser.Name,
		AvatarURL: jihulabUser.AvatarURL,
	}

	err = h.db.RunInTx(ctx, func(tx model.Operator) (err error) {
		err = tx.InsertUser(ctx, &newUser)
		if err != nil {
			err = fmt.Errorf("inserting user: %w", err)
			return
		}

		err = tx.InsertPassport(ctx, &model.Passport{
			UserID: newUser.ID,
			Vendor: model.PassportVendorJihulab,
			ExtUID: extUID,
		})
		if err != nil {
			err = fmt.Errorf("inserting passport: %w", err)
			return
		}

		return
	})
	if err != nil {
		err = fmt.Errorf("registering Jihulab user %q(id: %d): %w", jihulabUser.Username, jihulabUser.ID, err)
		return
	}

	// log preallowed user in
	if isUserPreallowed {
		var session model.Session
		session, err = h.logUserIn(ctx, newUser.ID)
		if err != nil {
			err = fmt.Errorf("logging new user in: %w", err)
			return
		}

		c.SetCookie(sessionCookieKey, session.ID, sessionMaxAgeSeconds, "", "", true, true)

		return
	}

	// redirect pending-invitation users to the beta sign-up collecting sheet
	c.Redirect(http.StatusFound, h.betaConfig.InvitationSignUpSheetURLWithEmail(jihulabUser.Email))
	alreadyRedirected = true
}

func (h *APIHandler) logUserIn(ctx context.Context, userID int) (session model.Session, err error) {
	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		err = fmt.Errorf("querying user: %w", err)
		return
	}
	if user.Status != model.UserStatusOK {
		err = fmt.Errorf("status of user %q(id: %d) is %q, want %q", user.Name, user.ID, user.Status, model.UserStatusOK)
		return
	}

	session = model.Session{
		UserID:    userID,
		Status:    model.SessionStatusOK,
		ExpiredAt: time.Now().Add(sessionMaxAgeSeconds * time.Second),
	}
	err = h.db.InsertSession(ctx, &session)
	if err != nil {
		err = fmt.Errorf("inserting session: %w", err)
		return
	}

	return
}

type betaUserApprovalTemplateContext struct {
	Username string
	SiteURL  string
}

// BetaApproveUser beta only: approve user into private beta
func (h *APIHandler) BetaApproveUser(c *gin.Context) {
	type Req struct {
		// table name at vika.cn
		Table string `json:"table"`
		// user provided email
		Email string `json:"email" binding:"required,max=200"`
		// is this an approval request
		Approved bool `json:"approved"`
	}

	var (
		err error
		req Req
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
		}

		// respond the webhook asap.
		// The webhook does not need to know the error
		OK(c, nil)
	}()

	bearerToken := strings.TrimPrefix(c.Request.Header.Get("Authorization"), "Bearer ")
	if bearerToken != h.betaConfig.APIBearerToken {
		err = errors.New("bearer token mismatches")
		return
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}
	if !req.Approved {
		// relax
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	go func() {
		var badLuck error

		defer func() {
			cancel()
			if badLuck != nil {
				h.logger.Warn("problem when approving the beta user", zap.Error(badLuck), zap.String("table", req.Table), zap.String("email", req.Email))
			}
		}()

		user, badLuck := h.db.GetUserByEmail(ctx, req.Email)
		if badLuck != nil {
			badLuck = fmt.Errorf("querying user by email %q: %w", req.Email, badLuck)
			return
		}
		if user.Status != model.UserStatusPendingBetaInvitation {
			badLuck = fmt.Errorf("status of user %d is %q, want %q", user.ID, user.Status, model.UserStatusPendingBetaInvitation)
			return
		}

		badLuck = h.db.UpdateUserStatus(ctx, user.ID, model.UserStatusOK, model.UserStatusPendingBetaInvitation)
		if badLuck != nil {
			badLuck = fmt.Errorf("UpdateUserStatus: %w", badLuck)
			return
		}

		h.logger.Info("beta user approved", zap.String("table", req.Table), zap.String("email", req.Email))

		var buf bytes.Buffer
		badLuck = betaUserApprovalTemplate.Execute(&buf, betaUserApprovalTemplateContext{
			Username: user.Name,
			SiteURL:  h.serverHost.API(),
		})
		if badLuck != nil {
			badLuck = fmt.Errorf("executing email template: %w", badLuck)
			return
		}

		badLuck = h.mailSender.SendMail(ctx, smtp.Mail{
			To:       []string{user.Email},
			Subject:  fmt.Sprintf("%s ，欢迎使用 UltraFox ！", user.Name),
			Body:     buf.Bytes(),
			HTMLBody: true,
		})
		if badLuck != nil {
			badLuck = fmt.Errorf("sending approval notification email: %w", badLuck)
			return
		}
	}()
}
