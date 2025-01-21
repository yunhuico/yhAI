package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIssue_UnmarshalJSON(t *testing.T) {
	const corpse = `{"expand":"operations,versionedRepresentations,editmeta,changelog,customfield_10010.requestTypePractice,renderedFields","id":"10004","self":"https://nanmu42.atlassian.net/rest/api/3/issue/10004","key":"UL-5","fields":{"statuscategorychangedate":"2023-01-10T09:41:44.708+0800","issuetype":{"self":"https://nanmu42.atlassian.net/rest/api/3/issuetype/10001","id":"10001","description":"Tasks track small, distinct pieces of work.","iconUrl":"https://nanmu42.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium","name":"Task","subtask":false,"avatarId":10318,"entityId":"deff9e6f-00ba-4225-ae99-b328888e9568","hierarchyLevel":0},"timespent":null,"customfield_10030":null,"customfield_10031":null,"project":{"self":"https://nanmu42.atlassian.net/rest/api/3/project/10000","id":"10000","key":"UL","name":"Ultrafox","projectTypeKey":"software","simplified":true,"avatarUrls":{"48x48":"https://nanmu42.atlassian.net/rest/api/3/universal_avatar/view/type/project/avatar/10403","24x24":"https://nanmu42.atlassian.net/rest/api/3/universal_avatar/view/type/project/avatar/10403?size=small","16x16":"https://nanmu42.atlassian.net/rest/api/3/universal_avatar/view/type/project/avatar/10403?size=xsmall","32x32":"https://nanmu42.atlassian.net/rest/api/3/universal_avatar/view/type/project/avatar/10403?size=medium"}},"fixVersions":[],"aggregatetimespent":null,"resolution":null,"customfield_10027":null,"customfield_10028":null,"customfield_10029":null,"resolutiondate":null,"workratio":-1,"watches":{"self":"https://nanmu42.atlassian.net/rest/api/3/issue/UL-5/watchers","watchCount":1,"isWatching":false},"lastViewed":null,"created":"2023-01-10T09:41:43.359+0800","customfield_10020":null,"customfield_10021":null,"customfield_10022":null,"priority":{"self":"https://nanmu42.atlassian.net/rest/api/3/priority/3","iconUrl":"https://nanmu42.atlassian.net/images/icons/priorities/medium.svg","name":"Medium","id":"3"},"customfield_10023":null,"customfield_10024":null,"customfield_10025":null,"labels":[],"customfield_10026":null,"customfield_10016":null,"customfield_10017":null,"customfield_10018":{"hasEpicLinkFieldDependency":false,"showField":false,"nonEditableReason":{"reason":"PLUGIN_LICENSE_ERROR","message":"The Parent Link is only available to Jira Premium users."}},"customfield_10019":"0|i0001b:","timeestimate":null,"aggregatetimeoriginalestimate":null,"versions":[],"issuelinks":[],"assignee":null,"updated":"2023-01-10T09:41:43.359+0800","status":{"self":"https://nanmu42.atlassian.net/rest/api/3/status/10000","description":"","iconUrl":"https://nanmu42.atlassian.net/","name":"To Do","id":"10000","statusCategory":{"self":"https://nanmu42.atlassian.net/rest/api/3/statuscategory/2","id":2,"key":"new","colorName":"blue-gray","name":"To Do"}},"components":[],"timeoriginalestimate":null,"description":{"version":1,"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"this is a description"}]}]},"customfield_10010":null,"customfield_10014":null,"customfield_10015":null,"customfield_10005":null,"customfield_10006":null,"customfield_10007":null,"security":null,"customfield_10008":null,"aggregatetimeestimate":null,"customfield_10009":null,"summary":"new issue at 2023年01月10日09:41:32","creator":{"self":"https://nanmu42.atlassian.net/rest/api/3/user?accountId=63bbc70c49a31f95b87319c3","accountId":"63bbc70c49a31f95b87319c3","avatarUrls":{"48x48":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","24x24":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","16x16":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","32x32":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png"},"displayName":"LI Zhennan","active":true,"timeZone":"Asia/Shanghai","accountType":"atlassian"},"subtasks":[],"reporter":{"self":"https://nanmu42.atlassian.net/rest/api/3/user?accountId=63bbc70c49a31f95b87319c3","accountId":"63bbc70c49a31f95b87319c3","avatarUrls":{"48x48":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","24x24":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","16x16":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png","32x32":"https://secure.gravatar.com/avatar/eabb1a6da084a055db8036aa14e1f5bb?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FLZ-3.png"},"displayName":"LI Zhennan","active":true,"timeZone":"Asia/Shanghai","accountType":"atlassian"},"aggregateprogress":{"progress":0,"total":0},"customfield_10001":null,"customfield_10002":null,"customfield_10003":null,"customfield_10004":null,"environment":null,"duedate":null,"progress":{"progress":0,"total":0},"votes":{"self":"https://nanmu42.atlassian.net/rest/api/3/issue/UL-5/votes","votes":0,"hasVoted":false}}}`

	var (
		assert = require.New(t)
		issue  Issue
		err    error
	)
	err = json.Unmarshal([]byte(corpse), &issue)
	assert.NoError(err)
	assert.Equal(int64(1673314903359), issue.Fields.UpdatedAt.Time().UnixMilli())
	assert.Equal(int64(1673314903359), issue.Fields.CreatedAt.Time().UnixMilli())

	marshaled, err := json.Marshal(issue)
	assert.NoError(err)
	assert.JSONEq(corpse, string(marshaled))
}

func TestClient_ListRecentlyCreatedIssues(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		var (
			assert = require.New(t)
			a      = time.UnixMilli(1670550124123)
			b      = time.UnixMilli(1673055731598)
		)

		issues, latest, err := client.ListRecentlyCreatedIssues(ctx, a, "")
		assert.NoError(err)
		t.Logf("there are %d issues after %s, latest %s", len(issues), a, latest)
		assert.True(latest.After(a))
		assert.True(len(issues) > 0)

		issues, latest, err = client.ListRecentlyCreatedIssues(ctx, b, "")
		assert.NoError(err)
		t.Logf("there are %d issues after %s, latest %s", len(issues), b, latest)
		assert.True(latest.After(b))
		assert.True(len(issues) > 0)
	})
}

func TestClient_ListRecentlyUpdatedIssues(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		var (
			assert = require.New(t)
			a      = time.UnixMilli(1670550124123)
			b      = time.UnixMilli(1673055731598)
		)

		issues, latest, err := client.ListRecentlyUpdatedIssues(ctx, a, "")
		assert.NoError(err)
		t.Logf("there are %d issues after %s, latest %s", len(issues), a, latest)
		assert.True(latest.After(a))
		assert.True(len(issues) > 0)

		issues, latest, err = client.ListRecentlyUpdatedIssues(ctx, b, "")
		assert.NoError(err)
		t.Logf("there are %d issues after %s, latest %s", len(issues), b, latest)
		assert.True(latest.After(b))
		assert.True(len(issues) > 0)
	})
}

func TestClient_GetIssueMetadataByProjectID(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		types, err := client.GetIssueMetadataByProjectID(ctx, "10000")
		assert.NoError(err)
		assert.True(len(types) > 0)
		marshal, err := json.Marshal(types)
		assert.NoError(err)
		fmt.Printf("%s\n", marshal)
	})
}

func TestClient_GetWorkflowStatus(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		status, err := client.GetValidTransitionOfIssue(ctx, "UL-13")
		assert.NoError(err)
		assert.Len(status, 3)
	})
}

func TestClient_CreateUpdateIssue(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		const projectID = "10000"

		assert := require.New(t)

		types, err := client.GetIssueMetadataByProjectID(ctx, projectID)
		assert.NoError(err)
		assert.True(len(types) > 0)

		issue := CreateIssueReq{
			Fields: map[string]any{
				"issuetype": map[string]string{"id": types[0].ID},
				"summary":   "A issue creation test - " + time.Now().UTC().Format(time.RFC3339),
				"project":   map[string]string{"id": projectID},
			},
			Transition: &IssueTransitionInput{
				ID: "21", // In Progress
			},
		}

		resp, err := client.CreateIssue(ctx, issue)
		assert.NoError(err)
		assert.NotEmpty(resp.ID)
		assert.NotEmpty(resp.Key)
		assert.NotEmpty(resp.Self)

		update := UpdateIssueReq{
			Fields: map[string]any{
				"summary":     "A issue creation/update test - " + time.Now().UTC().Format(time.RFC3339),
				"description": DocText("This is line 1\nThis is line 2"),

				// Using the same value during creation for compatibility check

				"issuetype": map[string]string{"id": types[0].ID},
				"project":   map[string]string{"id": projectID},
			},
		}

		err = client.UpdateIssue(ctx, resp.ID, update)
		assert.NoError(err)

		err = client.TransitionIssue(ctx, resp.Key, "31") // Done
		assert.NoError(err)
	})
}

func TestDocText_MarshalJSON(t *testing.T) {
	assert := require.New(t)

	const want = `{
      "content": [
        {
          "content": [
            {
              "text": "Order entry fails when selecting supplier.",
              "type": "text"
            }
          ],
          "type": "paragraph"
        }
      ],
      "type": "doc",
      "version": 1
    }`

	got, err := json.Marshal(DocText("Order entry fails when selecting supplier."))
	assert.NoError(err)
	assert.JSONEq(want, string(got))
}

func TestClient_GetIssue(t *testing.T) {
	Suite.Run(t, func(t *testing.T, client *Client, ctx context.Context) {
		assert := require.New(t)

		issue, err := client.GetIssue(ctx, "UL-24")
		assert.NoError(err)

		assert.NotEmpty(issue.Fields.Summary)
		assert.NotEmpty(issue.Fields.CreatedAt.Time())
		assert.NotEmpty(issue.Fields.IssueType.ID)
		assert.NotEmpty(issue.Fields.IssueType.Name)
		assert.NotEmpty(issue.Fields.IssueType.Self)
	})
}
