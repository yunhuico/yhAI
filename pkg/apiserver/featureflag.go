package apiserver

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"
)

func TestFeatureFlag(c *gin.Context) {
	var (
		err error
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	req := &payload.TestFeatureFlagReq{}
	err = c.ShouldBindJSON(&req)
	if err != nil {
		err = fmt.Errorf("binding JSON: %w", err)
		return
	}

	enabled := featureflag.IsEnabled(c.Request.Context(), featureflag.FeatureName(req.Name), req.Context)
	OK(c, &response.TestFeatureFlagResp{Enabled: enabled})
}
