package apiserver

import (
	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
)

// ServerMeta returns the server meta information.
// @Accept json
// @Produce json
// @Success 302
// @Success 200 {object} apiserver.R{data=response.ServerMetaResp}
// @Failure 200 {object} apiserver.R
// @Router /api/v1/server/meta [get]
func (h *APIHandler) ServerMeta(c *gin.Context) {
	OK(c, response.ServerMetaResp{
		DocsHost:          h.serverHost.Docs(),
		OAuth2CallbackURL: h.serverHost.APIFullURL("/api/v1/credentials/oauth2/callback"),
	})
}
