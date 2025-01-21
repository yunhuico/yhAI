package pagerduty

import (
	"context"
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type basePagerDutyNode struct {
	client *pagerduty.Client
}

func (b *basePagerDutyNode) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) (err error) {
	authToken, err := dependencies.Authorizer.GetAccessToken(ctx)
	if err != nil {
		err = fmt.Errorf("Authorizer.GetAccessToken: %w", err)
		return
	}
	b.client = pagerduty.NewClient(authToken)
	return
}
