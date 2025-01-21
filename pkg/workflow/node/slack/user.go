package slack

import (
	"fmt"

	"github.com/slack-go/slack"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type ListUsers struct {
	BaseSlackNode `json:"-"`
	workflow.ListPagination
}

func (l ListUsers) Run(c *workflow.NodeContext) (any, error) {
	return l.run(c)
}

func (l ListUsers) run(c *workflow.NodeContext) ([]slack.User, error) {
	users, err := l.client.GetUsersContext(c.Context())
	if err != nil {
		err = fmt.Errorf("slack get user in conversations: %w", err)
		return nil, err
	}
	return users, nil
}

func (l ListUsers) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#listUser")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListUsers)
		},
		InputForm: spec.InputSchema,
	}
}

func (l ListUsers) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	users, err := l.run(c)
	if err != nil {
		return
	}
	result.NoMore = true
	for _, user := range users {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: user.Name,
			Value: user.ID,
		})
	}
	return
}
