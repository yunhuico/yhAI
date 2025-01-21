package jira

import (
	"context"
	"net/http"
)

func (c *Client) GetInstanceInfo(ctx context.Context) (resp InstanceInfo, err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   "/rest/api/3/serverInfo",
		Body:   nil,
		Dest:   &resp,
	})

	return
}
