package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// jqlAbsoluteTimeLayout yyyy/MM/dd HH:mm
// refers to
// https://support.atlassian.com/jira-service-management-cloud/docs/advanced-search-reference-jql-fields/#Advancedsearchingfieldsreference-CreatedCreatedDatecreatedDateCreated
const jqlAbsoluteTimeLayout = "2006/01/02 15:04"

type Issue struct {
	Expand    string          `json:"expand,omitempty"`
	ID        string          `json:"id,omitempty"`
	Self      string          `json:"self,omitempty"`
	Key       string          `json:"key,omitempty"`
	RawFields json.RawMessage `json:"fields,omitempty"`

	Fields IssueFields `json:"-"`
}

func (i *Issue) UnmarshalJSON(data []byte) (err error) {
	type wrapped Issue

	var raw wrapped
	err = json.Unmarshal(data, &raw)
	if err != nil {
		err = fmt.Errorf("unmarshaling on raw type: %w", err)
		return
	}

	err = json.Unmarshal(raw.RawFields, &raw.Fields)
	if err != nil {
		err = fmt.Errorf("unmarshaling on fields: %w", err)
		return
	}

	*i = Issue(raw)
	return
}

type IssueFields struct {
	CreatedAt DateTime  `json:"created,omitempty"`
	UpdatedAt DateTime  `json:"updated,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	IssueType IssueType `json:"issueType"`
}

const dateTimeLayout = "2006-01-02T15:04:05.000-0700"

type DateTime time.Time

func (t DateTime) Time() time.Time {
	return time.Time(t)
}

func (t *DateTime) UnmarshalJSON(data []byte) (err error) {

	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return
	}

	raw, err := time.Parse(`"`+dateTimeLayout+`"`, string(data))
	*t = DateTime(raw)
	return
}

func (t DateTime) MarshalJSON() ([]byte, error) {
	raw := time.Time(t)
	if y := raw.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		//
		// We are not using RFC3339 but let's go the Roman's way when in Rome.
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(dateTimeLayout)+2)
	b = append(b, '"')
	b = raw.AppendFormat(b, dateTimeLayout)
	b = append(b, '"')
	return b, nil
}

// ListRecentlyCreatedIssues lists issues created after param "after",
// the issues are ordered by its creation time ascending.
//
// The creation time of the latest issue is reported by the output batchLatest,
// which can be used as the param "after" the next time calling this method.
//
// When there's no new issues, batchLatest is set to after.
// This method returns at most around 1000 issues and a suitable batchLatest.
//
// Two pitfalls exist:
// 1. If there are more than 1000 issues created during one minute,
// issues at and after #1001 may not be accessible.
// 2. If there are more than one issues created at exactly "after" to millisecond precision,
// there may be data duplication.
func (c *Client) ListRecentlyCreatedIssues(ctx context.Context, after time.Time, project string) (issues []Issue, batchLatest time.Time, err error) {
	const (
		issuesPerCall      = 100
		batchSizeThreshold = 1000
	)

	userInfo, err := c.GetCurrentUser(ctx)
	if err != nil {
		err = fmt.Errorf("retriving user timezone: %w", err)
		return
	}

	location, err := time.LoadLocation(userInfo.TimeZone)
	if err != nil {
		err = fmt.Errorf("loading user timezone %q: %w", userInfo.TimeZone, err)
		return
	}

	var (
		formattedAfter = after.In(location).Format(jqlAbsoluteTimeLayout)
		jqlQuery       = fmt.Sprintf(`created > "%s" order by created asc`, formattedAfter)
		offset         int
	)
	if project != "" {
		jqlQuery = fmt.Sprintf(`project = "%s" and `, project) + jqlQuery
	}

	for {
		var resp IssueSearchResp
		resp, err = c.IssueSearch(ctx, IssueSearchOpt{
			JQL:        jqlQuery,
			StartAt:    offset,
			MaxResults: issuesPerCall,
		})
		if err != nil {
			err = fmt.Errorf("calling issue search at offset %d: %w", offset, err)
			return
		}
		if len(resp.Issues) == 0 {
			break
		}

		var n int
		for _, item := range resp.Issues {
			// filter out processed issues based on time at millisecond precision
			if after.Before(item.Fields.CreatedAt.Time()) {
				break
			}

			n++
		}

		issues = append(issues, resp.Issues[n:]...)
		if len(issues) > batchSizeThreshold {
			break
		}

		offset += len(resp.Issues)
	}

	length := len(issues)
	if length == 0 {
		batchLatest = after
	} else {
		batchLatest = issues[length-1].Fields.CreatedAt.Time()
	}

	return
}

// ListRecentlyUpdatedIssues lists issues updated after param "after",
// the issues are ordered by its updating time ascending.
//
// The updating time of the latest issue is reported by the output batchLatest,
// which can be used as the param "after" the next time calling this method.
//
// When there's no new issues, batchLatest is set to after.
// This method returns at most around 1000 issues and a suitable batchLatest.
//
// Two pitfalls exist:
// 1. If there are more than 1000 issues updated during one minute,
// issues at and after #1001 may not be accessible.
// 2. If there are more than one issues updated at exactly "after" to millisecond precision,
// there may be data duplication.
func (c *Client) ListRecentlyUpdatedIssues(ctx context.Context, after time.Time, project string) (issues []Issue, batchLatest time.Time, err error) {
	const (
		issuesPerCall      = 100
		batchSizeThreshold = 1000
	)

	userInfo, err := c.GetCurrentUser(ctx)
	if err != nil {
		err = fmt.Errorf("retriving user timezone: %w", err)
		return
	}

	location, err := time.LoadLocation(userInfo.TimeZone)
	if err != nil {
		err = fmt.Errorf("loading user timezone %q: %w", userInfo.TimeZone, err)
		return
	}

	var (
		formattedAfter = after.In(location).Format(jqlAbsoluteTimeLayout)
		jqlQuery       = fmt.Sprintf(`updated > "%s" order by updated asc`, formattedAfter)
		offset         int
	)
	if project != "" {
		jqlQuery = fmt.Sprintf(`project = "%s" and `, project) + jqlQuery
	}

	for {
		var resp IssueSearchResp
		resp, err = c.IssueSearch(ctx, IssueSearchOpt{
			JQL:        jqlQuery,
			StartAt:    offset,
			MaxResults: issuesPerCall,
		})
		if err != nil {
			err = fmt.Errorf("calling issue search at offset %d: %w", offset, err)
			return
		}
		if len(resp.Issues) == 0 {
			break
		}

		var n int
		for _, item := range resp.Issues {
			// filter out processed issues based on time at millisecond precision
			itemTime := item.Fields.UpdatedAt.Time()
			if after.Before(itemTime) {
				break
			}

			n++
		}

		for _, item := range resp.Issues[n:] {
			// filter out newly created issues
			if item.Fields.CreatedAt.Time().Equal(item.Fields.UpdatedAt.Time()) {
				continue
			}

			issues = append(issues, item)
		}

		batchLatest = resp.Issues[len(resp.Issues)-1].Fields.UpdatedAt.Time()

		if len(issues) > batchSizeThreshold {
			break
		}

		offset += len(resp.Issues)
	}

	if batchLatest.IsZero() && len(issues) == 0 {
		batchLatest = after
	}

	return
}

// GetIssueMetadataByProjectID gets issue metadata by project id.
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-createmeta-get
func (c *Client) GetIssueMetadataByProjectID(ctx context.Context, projectID string) (issueTypes []IssueType, err error) {
	var resp IssueMetaResp

	query := make(url.Values)
	query.Set("expand", "projects.issuetypes.fields")
	if projectID != "" {
		query.Set("projectIds", projectID)
	}

	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   "/rest/api/3/issue/createmeta",
		Body:   nil,
		Query:  query,
		Dest:   &resp,
	})
	if err != nil {
		return
	}
	if len(resp.Projects) == 0 {
		err = errors.New("no IssueMetaProjects available")
		return
	}
	issueTypes = resp.Projects[0].IssueTypes

	return
}

type IssueMetaResp struct {
	// When filtered by projected, there's only one item
	Projects []IssueMetaProjects `json:"projects"`
}

type IssueMetaProjects struct {
	// issuetypes
	Expand string `json:"expand"`
	// https://nanmu42.atlassian.net/rest/api/3/project/10000
	Self string `json:"self"`
	// 10000
	ID string `json:"id"`
	// UL
	Key string `json:"key"`
	// Ultrafox
	Name string `json:"name"`
	// one item for one issue type
	IssueTypes []IssueType `json:"issuetypes"`
}

type IssueType struct {
	// https://nanmu42.atlassian.net/rest/api/3/issuetype/10004
	Self string `json:"self"`
	// 10004
	ID string `json:"id"`
	// Bugs track problems or errors.
	Description string `json:"description"`
	// https://nanmu42.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10303?size=medium
	IconURL string `json:"iconUrl"`
	// Bug
	Name string `json:"name"`
	// Bug
	UntranslatedName string `json:"untranslatedName"`
	Subtask          bool   `json:"subtask"`
	// fields
	Expand string          `json:"expand"`
	Fields IssueTypeFields `json:"fields,omitempty"`
}

// IssueTypeFields key: field name, value: field meta
type IssueTypeFields map[string]IssueField

func (i IssueTypeFields) FindKeyByFieldName(name ...string) string {
	for k, v := range i {
		for _, want := range name {
			if v.Name == want {
				return k
			}
		}
	}

	return ""
}

func (i IssueTypeFields) KeyExists(key string) bool {
	_, ok := i[key]
	return ok
}

type IssueField struct {
	Required bool             `json:"required"`
	Schema   IssueFieldSchema `json:"schema"`
	// Linked Issues
	Name string `json:"name"`
	// issuelinks
	Key             string `json:"key"`
	HasDefaultValue bool   `json:"hasDefaultValue"`
	// add, copy, set
	Operations    []string                  `json:"operations,omitempty"`
	AllowedValues []IssueFieldAllowedValues `json:"allowedValues,omitempty"`
}

type IssueFieldSchema struct {
	// string, array, date, ...
	Type string `json:"type"`
	// issuelinks, for array
	Items string `json:"items,omitempty"`
	// com.atlassian.jira.plugin.system.customfieldtypes:datepicker
	Custom string `json:"custom,omitempty"`
	// issuelinks
	System string `json:"system"`
}

type IssueFieldAllowedValues struct {
	// https://nanmu42.atlassian.net/rest/api/3/customFieldOption/10019
	Self string `json:"self"`
	// 10019
	ID string `json:"id"`
	// e.g. Impediment
	Value any `json:"value,omitempty"`
	// Epics track large pieces of work.
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
	// Epic
	Name           string `json:"name,omitempty"`
	Subtask        bool   `json:"subtask,omitempty"`
	HierarchyLevel int    `json:"hierarchyLevel,omitempty"`
}

type GetValidTransitionOfIssueResp struct {
	Expand      string            `json:"expand"`
	Transitions []IssueTransition `json:"transitions"`
}

type StatusCategory struct {
	Self      string `json:"self"`
	ID        int    `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

type WorkflowStatus struct {
	Self           string         `json:"self"`
	Description    string         `json:"description"`
	IconURL        string         `json:"iconUrl"`
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	StatusCategory StatusCategory `json:"statusCategory"`
}

type IssueTransition struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	To            WorkflowStatus `json:"to"`
	HasScreen     bool           `json:"hasScreen"`
	IsGlobal      bool           `json:"isGlobal"`
	IsInitial     bool           `json:"isInitial"`
	IsAvailable   bool           `json:"isAvailable"`
	IsConditional bool           `json:"isConditional"`
	IsLooped      bool           `json:"isLooped"`
}

// GetValidTransitionOfIssue returns either all transitions or a transition that can be performed by the user on an issue,
// based on the issue's status.
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-issueidorkey-notify-post
func (c *Client) GetValidTransitionOfIssue(ctx context.Context, issueIDOrKey string) (transitions []IssueTransition, err error) {
	var resp GetValidTransitionOfIssueResp

	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueIDOrKey),
		Dest:   &resp,
	})
	transitions = resp.Transitions

	return
}

type CreateIssueReq struct {
	// key: field key, value: field object suitable for the field type
	Fields     map[string]any        `json:"fields"`
	Transition *IssueTransitionInput `json:"transition,omitempty"`
}

type UpdateIssueReq struct {
	// key: field key, value: field object suitable for the field type
	Fields map[string]any `json:"fields"`
}

type IssueTransitionInput struct {
	ID string `json:"id"`
}

type CreateIssueResp struct {
	// issue id
	ID string `json:"id"`
	// issue key
	Key string `json:"key"`
	// issue URL
	Self string `json:"self"`
}

func (c *Client) GetIssue(ctx context.Context, issueIDOrKey string) (issue Issue, err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/rest/api/3/issue/%s", issueIDOrKey),
		Dest:   &issue,
	})

	return
}

// CreateIssue Creates an issue or, where the option to create subtasks is enabled in Jira, a subtask.
// A transition may be applied, to move the issue or subtask to a workflow step other than the default start step,
// and issue properties set.
//
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-post
func (c *Client) CreateIssue(ctx context.Context, issue CreateIssueReq) (resp CreateIssueResp, err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodPost,
		Path:   "/rest/api/3/issue",
		Body:   &issue,
		Query:  nil,
		Dest:   &resp,
	})

	return
}

func (c *Client) UpdateIssue(ctx context.Context, issueIDOrKey string, updateFields UpdateIssueReq) (err error) {
	err = c.call(ctx, callOpt{
		Method: http.MethodPut,
		Path:   "/rest/api/3/issue/" + issueIDOrKey,
		Body:   &updateFields,
	})

	return
}

// TransitionIssue Performs an issue transition and,
// if the transition has a screen, updates the fields from the transition screen.
//
// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-issueidorkey-transitions-get
func (c *Client) TransitionIssue(ctx context.Context, issueIDOrKey string, transitionID string) (err error) {
	req := map[string]any{
		"transition": map[string]string{
			"id": transitionID,
		},
	}

	err = c.call(ctx, callOpt{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueIDOrKey),
		Body:   req,
	})

	return
}

// DocText handles JSON format of Jira Doc text
type DocText string

func (c DocText) MarshalJSON() ([]byte, error) {
	v := map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []map[string]any{
			{
				"type": "paragraph",
				"content": []map[string]any{
					{
						"text": string(c),
						"type": "text",
					},
				},
			},
		},
	}

	return json.Marshal(v)
}
