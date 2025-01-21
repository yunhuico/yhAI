package jira

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// IssueSearch searches issues by JQL
//
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-post
func (c *Client) IssueSearch(ctx context.Context, opt IssueSearchOpt) (resp IssueSearchResp, err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodPost,
		Path:   "/rest/api/3/search",
		Body:   opt,
		Dest:   &resp,
	})

	return
}

type IssueSearchOpt struct {
	// A JQL expression.
	// https://support.atlassian.com/jira-service-management-cloud/docs/use-advanced-search-with-jira-query-language-jql/
	JQL string `json:"jql,omitempty"`
	// The index of the first item to return in the page of results (page offset).
	// The base index is 0
	StartAt int `json:"startAt"`
	// The maximum number of items to return per page.
	MaxResults   int      `json:"maxResults,omitempty"`
	FieldsByKeys bool     `json:"fieldsByKeys,omitempty"`
	Fields       []string `json:"fields,omitempty"`
	Expand       []string `json:"expand,omitempty"`
}

type IssueSearchResp struct {
	RespPagination

	Expand string  `json:"expand,omitempty"`
	Issues []Issue `json:"issues,omitempty"`
}

type ProjectSearchOpt struct {
	// The index of the first item to return in the page of results (page offset).
	// The base index is 0
	StartAt int
	// The maximum number of items to return per page.
	// Default is 50.
	MaxResults int
	// Filter the results using a literal string.
	// Projects with a matching key or name are returned (case insensitive).
	Query string
}

type ProjectSearchResp struct {
	RespPagination

	Values []Project `json:"values,omitempty"`
}

type Project struct {
	Expand         string     `json:"expand"`
	Self           string     `json:"self"`
	ID             string     `json:"id"`
	Key            string     `json:"key"`
	Name           string     `json:"name"`
	AvatarUrls     AvatarUrls `json:"avatarUrls"`
	ProjectTypeKey string     `json:"projectTypeKey"`
	Simplified     bool       `json:"simplified"`
	Style          string     `json:"style"`
	IsPrivate      bool       `json:"isPrivate"`
	EntityID       string     `json:"entityId"`
	UUID           string     `json:"uuid"`
}

// ProjectSearch searches project by its name or key
//
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-projects/#api-rest-api-3-project-search-get
func (c *Client) ProjectSearch(ctx context.Context, opt ProjectSearchOpt) (resp ProjectSearchResp, err error) {
	query := make(url.Values)
	if opt.Query != "" {
		query.Set("query", opt.Query)
	}
	if opt.StartAt > 0 {
		query.Set("startAt", strconv.Itoa(opt.StartAt))
	}
	if opt.MaxResults > 0 {
		query.Set("maxResults", strconv.Itoa(opt.MaxResults))
	}

	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   "/rest/api/3/project/search",
		Body:   nil,
		Query:  query,
		Dest:   &resp,
	})

	return
}

type InstanceInfo struct {
	BaseURL        string `json:"baseUrl"`
	BuildDate      string `json:"buildDate"`
	BuildNumber    int    `json:"buildNumber"`
	DeploymentType string `json:"deploymentType"`
	HealthChecks   []struct {
		Description string `json:"description"`
		Name        string `json:"name"`
		Passed      bool   `json:"passed"`
	} `json:"healthChecks"`
	ScmInfo        string `json:"scmInfo"`
	ServerTime     string `json:"serverTime"`
	ServerTitle    string `json:"serverTitle"`
	Version        string `json:"version"`
	VersionNumbers []int  `json:"versionNumbers"`
}
