package jira

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_ListAllStatuses(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		statuses, err := client.ListAllStatuses(ctx, "UL")
		assert.NoError(err)
		assert.NotEmpty(statuses)

		got, err := json.Marshal(statuses)
		assert.NoError(err)

		const want = `[{"self":"https://nanmu42.atlassian.net/rest/api/3/status/10000","description":"","iconUrl":"https://nanmu42.atlassian.net/","name":"To Do","untranslatedName":"To Do","id":"10000","statusCategory":{"self":"https://nanmu42.atlassian.net/rest/api/3/statuscategory/2","id":2,"key":"new","colorName":"blue-gray","name":"To Do"}},{"self":"https://nanmu42.atlassian.net/rest/api/3/status/10001","description":"","iconUrl":"https://nanmu42.atlassian.net/","name":"In Progress","untranslatedName":"In Progress","id":"10001","statusCategory":{"self":"https://nanmu42.atlassian.net/rest/api/3/statuscategory/4","id":4,"key":"indeterminate","colorName":"yellow","name":"In Progress"}},{"self":"https://nanmu42.atlassian.net/rest/api/3/status/10002","description":"","iconUrl":"https://nanmu42.atlassian.net/","name":"Done","untranslatedName":"Done","id":"10002","statusCategory":{"self":"https://nanmu42.atlassian.net/rest/api/3/statuscategory/3","id":3,"key":"done","colorName":"green","name":"Done"}}]`
		assert.JSONEq(want, string(got))
	})
}
