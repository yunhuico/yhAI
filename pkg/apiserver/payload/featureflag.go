package payload

import "jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"

type TestFeatureFlagReq struct {
	Name    string                  `json:"name"`
	Context featureflag.ContextData `json:"context"`
}
