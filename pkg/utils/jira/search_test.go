package jira

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_IssueSearch(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		resp, err := client.IssueSearch(ctx, IssueSearchOpt{
			JQL:        "order by created desc",
			MaxResults: 10,
		})
		assert.NoError(err)

		assert.True(len(resp.Issues) > 0)
	})
}

func TestClient_ProjectSearch(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		resp, err := client.ProjectSearch(ctx, ProjectSearchOpt{
			Query: "",
		})
		assert.NoError(err)
		assert.True(len(resp.Values) > 0)

		resp, err = client.ProjectSearch(ctx, ProjectSearchOpt{
			Query: "NON-EXISTED",
		})
		assert.NoError(err)
		assert.True(len(resp.Values) == 0)
	})
}
