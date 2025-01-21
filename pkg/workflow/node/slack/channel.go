package slack

import (
	"fmt"

	"github.com/slack-go/slack"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type ListChannels struct {
	BaseSlackNode `json:"-"`

	workflow.ListPagination
}

func (l ListChannels) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/slack#listChannel")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListChannels)
		},
		InputForm: spec.InputSchema,
	}
}

// MaxRequestTimes each request get maximum 1000 channels, so 10 times can get 10000.
// 10000 channels maybe enough.
// TODO(sword): cache this?
const MaxRequestTimes = 10

func (l ListChannels) run(c *workflow.NodeContext) (channels []slack.Channel, nextCursor string, err error) {
	currentRequestTimes := 0
	for {
		if currentRequestTimes == MaxRequestTimes {
			break
		}
		params := &slack.GetConversationsParameters{
			Cursor:          nextCursor,
			ExcludeArchived: true,
			Limit:           1000,
		}

		currentRequestTimes++
		channels, nextCursor, err = l.client.GetConversationsContext(c.Context(), params)
		if err != nil {
			err = fmt.Errorf("slack get conversations: %w", err)
			break
		}

		if nextCursor != "" {
			break
		}
	}
	return
}

func (l ListChannels) Run(c *workflow.NodeContext) (any, error) {
	channels, _, err := l.run(c)
	return channels, err
}

func (l ListChannels) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	channels, nextCursor, err := l.run(c)
	result.NextCursor = nextCursor
	if nextCursor == "" {
		result.NoMore = true
	}
	for _, channel := range channels {
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: channel.Name,
			Value: channel.ID,
		})
	}
	return
}
