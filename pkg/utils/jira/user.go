package jira

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

type User struct {
	// API URL to the user
	Self string `json:"self"`
	// something like 123456:5c29cada-9411-4d47-8e53-196e48cd5786
	AccountID    string     `json:"accountId"`
	EmailAddress string     `json:"emailAddress"`
	AvatarUrls   AvatarUrls `json:"avatarUrls"`
	DisplayName  string     `json:"displayName"`
	Active       bool       `json:"active"`
	// e.g. Asia/Shanghai
	TimeZone string `json:"timeZone"`
	// e.g. en_US
	Locale string `json:"locale"`
}

type AvatarUrls struct {
	Dim16 string `json:"16x16"`
	Dim24 string `json:"24x24"`
	Dim32 string `json:"32x32"`
	Dim48 string `json:"48x48"`
}

func (c *Client) GetCurrentUser(ctx context.Context) (user User, err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   "/rest/api/3/myself",
		Body:   nil,
		Dest:   &user,
	})

	return
}

type FindAssignableUsersParam struct {
	// Required
	ProjectKeyOrID string
	// A query string that is matched against user attributes,
	// such as displayName, and emailAddress, to find relevant users.
	// The string can match the prefix of the attribute's value
	Query string
	// The index of the first item to return in the page of results (page offset).
	// The base index is 0
	StartAt int `json:"startAt"`
	// The maximum number of items to return per page.
	MaxResults int `json:"maxResults,omitempty"`
}

func (c *Client) FindAssignableUsers(ctx context.Context, param FindAssignableUsersParam) (users []User, err error) {
	query := make(url.Values)
	query.Set("project", param.ProjectKeyOrID)
	if param.Query != "" {
		query.Set("query", param.Query)
	}
	if param.StartAt > 0 {
		query.Set("startAt", strconv.Itoa(param.StartAt))
	}
	if param.MaxResults > 0 {
		query.Set("maxResults", strconv.Itoa(param.MaxResults))
	}

	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   "/rest/api/3/user/assignable/search",
		Query:  query,
		Dest:   &users,
	})

	return
}
