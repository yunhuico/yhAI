package jira

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_GetCurrentUser(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		user, err := client.GetCurrentUser(ctx)
		assert.NoError(err)
		assert.NotEmpty(user.Self)
		assert.NotEmpty(user.TimeZone)
	})
}

func TestClient_ListAssignableUsersForProject(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		users, err := client.FindAssignableUsers(ctx, FindAssignableUsersParam{
			ProjectKeyOrID: "10000",
			Query:          "",
		})
		assert.NoError(err)
		assert.True(len(users) > 1)

		users, err = client.FindAssignableUsers(ctx, FindAssignableUsersParam{
			ProjectKeyOrID: "10000",
			Query:          "nanmu42",
		})
		assert.NoError(err)
		assert.True(len(users) == 1)
		assert.Equal("nanmu42", users[0].DisplayName)
	})
}
