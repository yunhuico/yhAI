package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cast"
	"github.com/xanzy/go-gitlab"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

var (
	_ trigger.TriggerProvider = (*IssueTrigger)(nil)
	_ trigger.TriggerProvider = (*NoteTrigger)(nil)
	_ trigger.TriggerProvider = (*MergeRequestTrigger)(nil)
	_ trigger.TriggerProvider = (*PushTrigger)(nil)
	_ trigger.TriggerProvider = (*TagTrigger)(nil)
	_ trigger.TriggerProvider = (*ReleaseTrigger)(nil)
	_ trigger.TriggerProvider = (*JobTrigger)(nil)
	_ trigger.TriggerProvider = (*PipelineTrigger)(nil)
	_ trigger.TriggerProvider = (*MemberTrigger)(nil)

	_ trigger.SampleProvider = (*IssueTrigger)(nil)
	_ trigger.SampleProvider = (*NoteTrigger)(nil)
	_ trigger.SampleProvider = (*MergeRequestTrigger)(nil)
	_ trigger.SampleProvider = (*PushTrigger)(nil)
	_ trigger.SampleProvider = (*TagTrigger)(nil)
	_ trigger.SampleProvider = (*ReleaseTrigger)(nil)
	_ trigger.SampleProvider = (*JobTrigger)(nil)
	_ trigger.SampleProvider = (*PipelineTrigger)(nil)
	_ trigger.SampleProvider = (*MemberTrigger)(nil)

	_ workflow.PreFilterProvider = (*IssueTrigger)(nil)
	_ workflow.PreFilterProvider = (*NoteTrigger)(nil)
	_ workflow.PreFilterProvider = (*MergeRequestTrigger)(nil)
	_ workflow.PreFilterProvider = (*ReleaseTrigger)(nil)
	_ workflow.PreFilterProvider = (*JobTrigger)(nil)
	_ workflow.PreFilterProvider = (*PipelineTrigger)(nil)
	_ workflow.PreFilterProvider = (*MemberTrigger)(nil)
)

type conditionalEvent interface {
	// if not ok, indicates the event data is invalid.
	getEvent() (event string, ok bool)
}

// WebhookTime Gitlab is inconsistent on time format of webhook and API.
// It uses 2006-01-02T15:04:05Z07:00(RFC3339) in API and 2022-11-09 14:11:26 +0800 / 2022-11-09 07:24:10 UTC in webhook.
//
// This is a long-existed issue with many complains:
// https://gitlab.com/gitlab-org/gitlab/-/issues/19567
//
// WebhookTime must only be used in webhook triggers and any other usages(like in APIs) are discouraged.
type WebhookTime time.Time

func (t WebhookTime) Time() time.Time {
	return time.Time(t)
}

func (t *WebhookTime) UnmarshalJSON(data []byte) (err error) {
	const (
		numberLayout = "2006-01-02 15:04:05 -0700"
		letterLayout = "2006-01-02 15:04:05 MST"
	)

	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return
	}

	var (
		raw   time.Time
		input = string(data)
	)

	// Jihulab uses RFC3339 sometimes
	raw, err = time.Parse(`"`+time.RFC3339+`"`, input)
	if err == nil {
		*t = WebhookTime(raw)
		return
	}
	// Jihulab uses numberLayout at another times
	raw, err = time.Parse(`"`+numberLayout+`"`, input)
	if err == nil {
		*t = WebhookTime(raw)
		return
	}
	// Gitlab uses letterLayout
	raw, err = time.Parse(`"`+letterLayout+`"`, input)
	if err != nil {
		return
	}

	*t = WebhookTime(raw)
	return
}

func (t WebhookTime) MarshalJSON() ([]byte, error) {
	const numberLayout = "2006-01-02 15:04:05 -0700"

	raw := time.Time(t)
	if y := raw.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		//
		// We are not using RFC3339 but let's go the Roman's way when in Rome.
		return nil, errors.New("WebhookTime.MarshalJSON: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(numberLayout)+2)
	b = append(b, '"')
	b = raw.AppendFormat(b, numberLayout)
	b = append(b, '"')
	return b, nil
}

type Trigger workflow.HTTPRequest

type IssueTrigger struct {
	Trigger
}

func (g *IssueTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return g.Trigger.preFilterHTTP(configObj, data, &IssueEvent{})
}

func (g *Trigger) preFilterHTTP(configObj any, data []byte, eventObj conditionalEvent) (shouldAbort bool, err error) {
	err = json.Unmarshal(data, g)
	if err != nil {
		err = fmt.Errorf("json unmarshal data to trigger: %w", err)
		return
	}

	triggerConfig, ok := configObj.(*TriggerConfig)
	if !ok {
		err = errors.New("invalid trigger config")
		return
	}
	if triggerConfig.Event == "" || triggerConfig.Event == "all" {
		return
	}

	_, err = g.unmarshalBody(eventObj)
	if err != nil {
		err = fmt.Errorf("unmarshal to issue event: %w", err)
		return
	}

	event, ok := eventObj.getEvent()
	if !ok || event != triggerConfig.Event {
		shouldAbort = true
		return
	}

	return
}

func (g *IssueTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&IssueEvent{})
}

func (g *IssueTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "issues_events")
}

type NoteTrigger struct {
	Trigger
}

func (g *NoteTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return g.Trigger.preFilterHTTP(configObj, data, &NoteEvent{})
}

func (g *NoteTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&NoteEvent{})
}

var (
	openedState = "opened"
)

func (g *NoteTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	var issues []*gitlab.Issue
	if triggerConfig.isGroupScope() {
		var resources []any
		// collect 20 issues from 20 projects.
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 20, func(ctx context.Context, projectID int) (any, error) {
			newIssues, _, err := client.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 3,
				},
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project issues: %w", err)
			}
			return newIssues, nil
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			issues = append(issues, resource.(*gitlab.Issue))
		}
	} else {
		// try to get notes from the latest 5 issues.
		issues, _, err = client.Issues.ListProjectIssues(triggerConfig.ProjectID, &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 5,
			},
			State: &openedState,
		})
	}
	if err != nil {
		err = fmt.Errorf("list issues: %w", err)
		return
	}

	projects := map[int]*Project{}
	for _, issue := range issues {
		var notes []*gitlab.Note
		notes, _, err = client.Notes.ListIssueNotes(issue.ProjectID, issue.IID, &gitlab.ListIssueNotesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
		if err != nil {
			continue
		}
		err = collectProject(c.Context(), client, projects, issue.ProjectID)
		if err != nil {
			return
		}

		for _, note := range notes {
			// filter the system note
			if note.System {
				continue
			}

			result = append(result, &NoteEvent{
				ObjectKind: "issue",
				Project:    projects[issue.ProjectID],
				User:       toEventUser(user),
				ObjectAttributes: &Note{
					Id:           note.ID,
					AuthorId:     note.Author.ID,
					Note:         note.Body,
					NoteableId:   note.NoteableID,
					NoteableType: note.NoteableType,
					ProjectId:    issue.ProjectID,
					System:       note.System,
					CreatedAt:    (*WebhookTime)(note.CreatedAt),
					UpdatedAt:    (*WebhookTime)(note.UpdatedAt),
				},
			})

			if len(result) >= 3 {
				break
			}
		}
	}

	return
}

func (g *IssueTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerIssue"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(IssueTrigger)
		},
	}
}

func (g *IssueTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	var issues []*gitlab.Issue
	if triggerConfig.isGroupScope() {
		issues, _, err = client.Issues.ListGroupIssues(triggerConfig.GroupID, &gitlab.ListGroupIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
	} else {
		issues, _, err = client.Issues.ListProjectIssues(triggerConfig.ProjectID, &gitlab.ListProjectIssuesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
	}
	if err != nil {
		err = fmt.Errorf("list issues: %w", err)
		return
	}

	result = make([]trigger.SampleData, len(issues))
	projects := map[int]*Project{}
	for i, issue := range issues {
		if err = collectProject(c.Context(), client, projects, issue.ProjectID); err != nil {
			return
		}

		result[i] = IssueEvent{
			ObjectKind:       "issue",
			Project:          projects[issue.ProjectID],
			User:             toEventUser(user),
			ObjectAttributes: toEventIssue(issue),
			Labels:           toEventLabels(issue.LabelDetails),
			Assignees:        toEventAssignees(issue.Assignees),
			State:            openedState,
		}
	}

	return
}

func (g *NoteTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "note_events")
}

type MergeRequestTrigger struct {
	Trigger
}

func (g *MergeRequestTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return g.preFilterHTTP(configObj, data, &MergeRequestEvent{})
}

func (g *MergeRequestTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&MergeRequestEvent{})
}

func (g *MergeRequestTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	var mrs []*gitlab.MergeRequest
	if triggerConfig.isGroupScope() {
		mrs, _, err = client.MergeRequests.ListGroupMergeRequests(triggerConfig.GroupID, &gitlab.ListGroupMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
	} else {
		mrs, _, err = client.MergeRequests.ListProjectMergeRequests(triggerConfig.ProjectID, &gitlab.ListProjectMergeRequestsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
	}
	if err != nil {
		err = fmt.Errorf("list merge requests: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	projects := map[int]*Project{}
	result = make([]trigger.SampleData, len(mrs))
	for i, mr := range mrs {
		if err = collectProject(c.Context(), client, projects, mr.ProjectID); err != nil {
			return
		}

		result[i] = MergeRequestEvent{
			ObjectKind:       "merge_request",
			User:             toEventUser(user),
			Project:          projects[mr.ProjectID],
			ObjectAttributes: toMergeRequest(mr),
			Labels:           toLabelsFromStrings(mr.Labels),
			Assignees:        toEventAssignees2(mr.Assignees),
		}
	}

	return
}

func (g *MergeRequestTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "merge_requests_events")
}

type PushTrigger struct {
	Trigger
}

func (g *PushTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&PushEvent{})
}

func (g *PushTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	var commits []*gitlab.Commit
	if triggerConfig.isGroupScope() {
		var resources []any
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 3, func(ctx context.Context, projectID int) (any, error) {
			newCommits, _, err := client.Commits.ListCommits(projectID, &gitlab.ListCommitsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 3,
				},
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project commits: %w", err)
			}
			for _, commit := range newCommits {
				commit.ProjectID = projectID // gitlab api bug, return empty project_id.
			}
			return newCommits, err
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			commits = append(commits, resource.(*gitlab.Commit))
		}
	} else {
		commits, _, err = client.Commits.ListCommits(triggerConfig.ProjectID, &gitlab.ListCommitsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
		for _, commit := range commits {
			commit.ProjectID = triggerConfig.ProjectID // gitlab api bug, return empty project_id.
		}
	}
	if err != nil {
		err = fmt.Errorf("list commits: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	projects := map[int]*Project{}
	result = make([]trigger.SampleData, len(commits))
	for i, commit := range commits {
		if err = collectProject(c.Context(), client, projects, commit.ProjectID); err != nil {
			return
		}

		result[i] = PushEvent{
			ObjectKind:   "push",
			Before:       "",
			After:        "",
			Ref:          "main",
			CheckoutSha:  commit.ID,
			UserId:       user.ID,
			UserName:     user.Username,
			UserUsername: user.Name,
			UserEmail:    user.Email,
			UserAvatar:   user.AvatarURL,
			ProjectId:    commit.ProjectID,
			Project:      projects[commit.ProjectID],
			Commits: []Commit{
				{
					Id:        commit.ID,
					Message:   commit.Message,
					Title:     commit.Title,
					Timestamp: getTimestamp(commit.CreatedAt),
					Url:       commit.WebURL,
					Author: struct {
						Name  string `json:"name"`
						Email string `json:"email"`
					}{
						Name:  commit.AuthorName,
						Email: commit.AuthorEmail,
					},
				},
			},
			TotalCommitsCount: 1,
		}
	}

	return
}

func (g *PushTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "push_events")
}

type TagTrigger struct {
	Trigger
}

func (g *TagTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&TagEvent{})
}

func (g *TagTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	var tags []*gitlab.Tag
	if triggerConfig.isGroupScope() {
		var resources []any
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 3, func(ctx context.Context, projectID int) (any, error) {
			tags, _, err := client.Tags.ListTags(projectID, &gitlab.ListTagsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 3,
				},
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project tags: %w", err)
			}
			var newTags []*gitlab.Tag
			for _, tag := range tags {
				if tag.Commit != nil {
					tag.Commit.ProjectID = projectID
					newTags = append(newTags, tag)
				}
			}
			return newTags, nil
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			tags = append(tags, resource.(*gitlab.Tag))
		}
	} else {
		tags, _, err = client.Tags.ListTags(triggerConfig.ProjectID, &gitlab.ListTagsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
		for _, tag := range tags {
			if tag.Commit != nil {
				tag.Commit.ProjectID = triggerConfig.ProjectID
			}
		}
	}
	if err != nil {
		err = fmt.Errorf("list issues: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	projects := map[int]*Project{}
	result = make([]trigger.SampleData, 0, len(tags))
	for _, tag := range tags {
		if tag.Commit == nil {
			continue
		}

		if err = collectProject(c.Context(), client, projects, tag.Commit.ProjectID); err != nil {
			return
		}

		result = append(result, TagEvent{
			ObjectKind:   "tag_push",
			CheckoutSHA:  tag.Target,
			Ref:          "ref/tags/" + tag.Name,
			UserID:       user.ID,
			UserName:     user.Name,
			UserUsername: user.Username,
			UserAvatar:   user.AvatarURL,
			UserEmail:    user.Email,
			ProjectID:    tag.Commit.ProjectID,
			Message:      tag.Message,
			Project:      projects[tag.Commit.ProjectID],
		})
	}

	return
}

func (g *TagTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "tag_push_events")
}

type ReleaseTrigger struct {
	Trigger
}

func (g *ReleaseTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return g.preFilterHTTP(configObj, data, &ReleaseEvent{})
}

func (g *ReleaseTrigger) Run(c *workflow.NodeContext) (any, error) {
	return g.unmarshalBody(&ReleaseEvent{})
}

func (g *ReleaseTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("issue trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	var releases []*gitlab.Release
	if triggerConfig.isGroupScope() {
		var resources []any
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 3, func(ctx context.Context, projectID int) (any, error) {
			newReleases, _, err := client.Releases.ListReleases(projectID, &gitlab.ListReleasesOptions{
				Page:    1,
				PerPage: 3,
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project commits: %w", err)
			}
			for _, release := range newReleases {
				release.Commit.ProjectID = projectID
			}
			return newReleases, nil
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			releases = append(releases, resource.(*gitlab.Release)) // gitlab api or sdk bug
		}
	} else {
		releases, _, err = client.Releases.ListReleases(triggerConfig.ProjectID, &gitlab.ListReleasesOptions{
			Page:    1,
			PerPage: 3,
		}, gitlab.WithContext(c.Context()))
		for _, release := range releases {
			release.Commit.ProjectID = triggerConfig.ProjectID // gitlab api or sdk bug
		}
		if err != nil {
			err = fmt.Errorf("list releases: %w", err)
			return
		}
	}

	projects := map[int]*Project{}
	result = make([]trigger.SampleData, len(releases))
	for i, release := range releases {
		if err = collectProject(c.Context(), client, projects, release.Commit.ProjectID); err != nil {
			return
		}

		assets := &Assets{}
		commit := &Commit{}
		_ = objToObj(release.Assets, assets)
		_ = objToObj(release.Commit, commit)
		result[i] = ReleaseEvent{
			ObjectKind:  "release",
			CreatedAt:   (*WebhookTime)(release.CreatedAt),
			Description: release.Description,
			Name:        release.Name,
			ReleasedAt:  release.ReleasedAt.Format("2006-01-02 15:04:05 -0700"),
			Tag:         release.TagName,
			Project:     projects[release.Commit.ProjectID],
			Action:      "create",
			Assets:      assets,
			Commit:      commit,
		}
	}

	return
}

func (g *ReleaseTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return g.create(c, "releases_events")
}

func (g *NoteTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerNote"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(NoteTrigger)
		},
	}
}

func (g *MergeRequestTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerMergeRequest"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(MergeRequestTrigger)
		},
	}
}

func (g *PushTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerPush"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(PushTrigger)
		},
	}
}

func (g *TagTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerTag"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(TagTrigger)
		},
	}
}

func (g *ReleaseTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerRelease"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ReleaseTrigger)
		},
	}
}

type JobTrigger struct {
	Trigger
}

func (j JobTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerJob"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(JobTrigger)
		},
	}
}

func (j JobTrigger) Run(c *workflow.NodeContext) (any, error) {
	return j.unmarshalBody(&jobEvent{})
}

func (j JobTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("job trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	var jobs []*gitlab.Job
	if triggerConfig.isGroupScope() {
		var resources []any
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 3, func(ctx context.Context, projectID int) (any, error) {
			newJobs, _, err := client.Jobs.ListProjectJobs(projectID, &gitlab.ListJobsOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 3,
				},
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project pipelines: %w", err)
			}
			return newJobs, nil
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			jobs = append(jobs, resource.(*gitlab.Job))
		}
	} else {
		jobs, _, err = client.Jobs.ListProjectJobs(triggerConfig.ProjectID, &gitlab.ListJobsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		})
	}
	if err != nil {
		err = fmt.Errorf("list jobs: %w", err)
		return
	}
	result = make([]trigger.SampleData, len(jobs))
	for i, job := range jobs {
		commit := &Commit{}
		_ = objToObj(job.Commit, commit)

		result[i] = jobEvent{
			ObjectKind:         "build",
			Ref:                job.Ref,
			Tag:                job.Tag,
			BeforeSha:          "",
			Sha:                "",
			BuildId:            job.ID,
			BuildName:          job.Name,
			BuildStage:         job.Stage,
			BuildStatus:        job.Status,
			BuildCreatedAt:     (*WebhookTime)(job.CreatedAt),
			BuildStartedAt:     (*WebhookTime)(job.StartedAt),
			BuildFinishedAt:    (*WebhookTime)(job.FinishedAt),
			BuildDuration:      job.Duration,
			BuildAllowFailure:  job.AllowFailure,
			BuildFailureReason: "",
			RetriesCount:       0,
			PipelineId:         job.Pipeline.ID,
			ProjectId:          job.Project.ID,
			ProjectName:        job.Project.Name,
			User:               toEventUser(job.User),
			Commit:             commit,
		}
	}
	return
}

func (j JobTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return j.preFilterHTTP(configObj, data, &jobEvent{})
}

func (j JobTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return j.create(c, "job_events")
}

type PipelineTrigger struct {
	Trigger
}

func (p PipelineTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerPipeline"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(PipelineTrigger)
		},
	}
}

func (p PipelineTrigger) Run(c *workflow.NodeContext) (any, error) {
	return p.unmarshalBody(&pipelineEvent{})
}

func (p PipelineTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	if c.GetAuthorizer() == nil {
		err = errors.New("job trigger authorizer empty")
		return
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		err = fmt.Errorf("new gitlab client: %w", err)
		return
	}

	user, _, err := client.Users.CurrentUser(gitlab.WithContext(c.Context()))
	if err != nil {
		err = fmt.Errorf("get current user: %w", err)
		return
	}

	var pipelines []*gitlab.PipelineInfo
	if triggerConfig.isGroupScope() {
		var resources []any
		err = collectGroupResource(c.Context(), client, triggerConfig.GroupID, &resources, 3, func(ctx context.Context, projectID int) (any, error) {
			newPipelines, _, err := client.Pipelines.ListProjectPipelines(projectID, &gitlab.ListProjectPipelinesOptions{
				ListOptions: gitlab.ListOptions{
					Page:    1,
					PerPage: 3,
				},
			}, gitlab.WithContext(ctx))
			if err != nil {
				return nil, fmt.Errorf("get project pipelines: %w", err)
			}
			return newPipelines, err
		})
		if err != nil {
			return
		}
		for _, resource := range resources {
			pipelines = append(pipelines, resource.(*gitlab.PipelineInfo))
		}
	} else {
		pipelines, _, err = client.Pipelines.ListProjectPipelines(triggerConfig.ProjectID, &gitlab.ListProjectPipelinesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 3,
			},
		}, gitlab.WithContext(c.Context()))
	}

	if err != nil {
		err = fmt.Errorf("list pipelines: %w", err)
		return
	}

	projects := map[int]*Project{}
	result = make([]trigger.SampleData, len(pipelines))
	for i, pipeline := range pipelines {
		if err = collectProject(c.Context(), client, projects, pipeline.ProjectID); err != nil {
			return
		}

		var jobs []*gitlab.Job
		jobs, _, err = client.Jobs.ListPipelineJobs(pipeline.ProjectID, pipeline.ID, nil)
		if err != nil {
			err = fmt.Errorf("list pipelines jobs: %w", err)
			return
		}
		builds := []Build{}
		for _, job := range jobs {
			builds = append(builds, Build{
				ID:           job.ID,
				Stage:        job.Stage,
				Name:         job.Name,
				Status:       job.Status,
				CreatedAt:    (*WebhookTime)(job.CreatedAt),
				StartedAt:    (*WebhookTime)(job.StartedAt),
				FinishedAt:   (*WebhookTime)(job.FinishedAt),
				When:         "manual || on_success || always",
				Manual:       false,
				AllowFailure: false,
			})
		}

		result[i] = pipelineEvent{
			ObjectKind: "pipeline",
			ObjectAttributes: Pipeline{
				Id:         pipeline.ID,
				Iid:        0,
				Ref:        pipeline.Ref,
				Tag:        false,
				Sha:        pipeline.SHA,
				BeforeSha:  "",
				Source:     pipeline.Source,
				Status:     pipeline.Status,
				Stages:     nil,
				CreatedAt:  (*WebhookTime)(pipeline.CreatedAt),
				FinishedAt: (*WebhookTime)(pipeline.UpdatedAt),
			},
			User:    toEventUser(user),
			Project: projects[pipeline.ProjectID],
			Builds:  builds,
		}
	}
	return
}

func collectProject(ctx context.Context, client *gitlab.Client, projects map[int]*Project, projectID int) error {
	if _, ok := projects[projectID]; !ok {
		project, _, err := client.Projects.GetProject(projectID, &gitlab.GetProjectOptions{}, gitlab.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("get project %d: %w", projectID, err)
		}
		projects[projectID] = toEventProject(project)
	}
	return nil
}

func collectGroupResource(ctx context.Context,
	client *gitlab.Client,
	groupID int,
	result *[]any,
	count int,
	// data should be a slice
	getProjectResourceAPI func(ctx context.Context, projectID int) (data any, err error)) error {

	projects, _, err := client.Groups.ListGroupProjects(groupID, &gitlab.ListGroupProjectsOptions{
		IncludeSubGroups: &boolTrue,
	}, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("list group projects: %w", err)
	}

	for _, project := range projects {
		data, err := getProjectResourceAPI(ctx, project.ID)
		if err != nil {
			return fmt.Errorf("list project resources: %w", err)
		}
		resources, _ := trans.ToAnySlice(data)
		for _, resource := range resources {
			if len(*result) >= count {
				break
			}
			*result = append(*result, resource)
		}
	}
	return nil
}

func (p PipelineTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return p.preFilterHTTP(configObj, data, &pipelineEvent{})
}

func (p PipelineTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	return p.create(c, "pipeline_events")
}

func (g *Trigger) unmarshalBody(object any) (any, error) {
	err := json.Unmarshal(g.Body, object)
	if err != nil {
		return nil, fmt.Errorf("unmarshal body error: %v", err)
	}
	return object, nil
}

type TriggerConfig struct {
	Scope     string `json:"scope"`
	ProjectID int    `json:"projectId"`
	GroupID   int    `json:"groupId"`
	Event     string `json:"event"`
}

func (c TriggerConfig) isGroupScope() bool {
	return c.Scope == "group"
}

var (
	boolTrue  = true
	boolFalse = false
)

// according to https://jihulab.com/ultrafox/ultrafox/-/issues/471,
// all events except push_event should specify PushEvents=false explicitly.
var projectEvents = map[string]func(opt *gitlab.AddProjectHookOptions){
	"issues_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.IssuesEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"merge_requests_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.MergeRequestsEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"push_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.PushEvents = &boolTrue
	},
	"releases_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.ReleasesEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"tag_push_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.TagPushEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"note_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.NoteEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"job_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.JobEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"pipeline_events": func(opt *gitlab.AddProjectHookOptions) {
		opt.PipelineEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
}

var groupEvents = map[string]func(opt *gitlab.AddGroupHookOptions){
	"issues_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.IssuesEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"merge_requests_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.MergeRequestsEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"push_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.PushEvents = &boolTrue
	},
	"releases_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.ReleasesEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"tag_push_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.TagPushEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"note_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.NoteEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"job_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.JobEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
	"pipeline_events": func(opt *gitlab.AddGroupHookOptions) {
		opt.PipelineEvents = &boolTrue
		opt.PushEvents = &boolFalse
	},
}

func (g *Trigger) create(c trigger.WebhookContext, event string) (map[string]any, error) {
	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		return nil, err
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	var (
		response *gitlab.Response
		rawHook  any
		hookID   int
	)
	if triggerConfig.isGroupScope() {
		opt := &gitlab.AddGroupHookOptions{}
		groupEvents[event](opt)
		url := c.GetWebhookURL()
		opt.URL = &url

		var hook *gitlab.GroupHook
		hook, response, err = client.Groups.AddGroupHook(triggerConfig.GroupID, opt, gitlab.WithContext(c.Context()))
		if hook != nil {
			hookID = hook.ID
		}
		rawHook = hook
	} else {
		opt := &gitlab.AddProjectHookOptions{}
		projectEvents[event](opt)
		url := c.GetWebhookURL()
		opt.URL = &url

		var hook *gitlab.ProjectHook
		hook, response, err = client.Projects.AddProjectHook(triggerConfig.ProjectID, opt, gitlab.WithContext(c.Context()))
		if hook != nil {
			hookID = hook.ID
		}
		rawHook = hook
	}

	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("gitlab server response %s", response.Status)
	}
	if rawHook == nil {
		return nil, err
	}
	return map[string]any{
		"webhookID": hookID,
		"raw":       rawHook,
	}, nil
}

func (g *Trigger) Exists(c trigger.WebhookContext) (bool, error) {
	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		return false, err
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return false, fmt.Errorf("invalid trigger config")
	}

	data := c.GetTriggerData()
	webhookID := getInt(data, "webhookID")
	if webhookID <= 0 {
		return false, nil
	}

	var (
		hook     any
		response *gitlab.Response
	)
	if triggerConfig.isGroupScope() {
		hook, response, err = client.Groups.GetGroupHook(triggerConfig.GroupID, webhookID, gitlab.WithContext(c.Context()))
	} else {
		hook, response, err = client.Projects.GetProjectHook(triggerConfig.ProjectID, webhookID, gitlab.WithContext(c.Context()))
	}
	if response.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("%w: %s", trigger.ErrTokenUnauthorized, err)
	}
	if isHookNotFound(response) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("gitlab server response %s", response.Status)
	}
	if hook == nil {
		return false, err
	}
	return true, err
}

func (g *Trigger) Delete(c trigger.WebhookContext) error {
	client, err := newClient(c.Context(), c.GetAuthorizer(), c.GetPassportVendorLookup())
	if err != nil {
		return err
	}

	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return fmt.Errorf("invalid trigger config")
	}

	triggerData := c.GetTriggerData()
	webhookID := getInt(triggerData, "webhookID")
	if webhookID <= 0 {
		return nil
	}

	var response *gitlab.Response
	if triggerConfig.isGroupScope() {
		response, err = client.Groups.DeleteGroupHook(triggerConfig.GroupID, webhookID, gitlab.WithContext(c.Context()))
	} else {
		response, err = client.Projects.DeleteProjectHook(triggerConfig.ProjectID, webhookID, gitlab.WithContext(c.Context()))
	}
	if response.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: %s", trigger.ErrTokenUnauthorized, err)
	}
	if isHookNotFound(response) {
		log.Warn("hook already deleted", log.Int("webhookID", webhookID))
		return nil
	}
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("gitlab server response %s", response.Status)
	}
	return nil
}

func (g *Trigger) GetConfigObject() any {
	return &TriggerConfig{}
}

func isHookNotFound(response *gitlab.Response) bool {
	return response != nil && response.StatusCode == http.StatusNotFound
}

func (g *Trigger) FieldsDeleted() []string {
	return []string{
		"webhookID",
		"raw",
	}
}

func getInt(params map[string]any, key string) int {
	value, ok := params[key]
	if !ok {
		return 0
	}
	return cast.ToInt(value)
}

type PushEvent struct {
	ObjectKind        string   `json:"object_kind"`
	Before            string   `json:"before"`
	After             string   `json:"after"`
	Ref               string   `json:"ref"`
	CheckoutSha       string   `json:"checkout_sha"`
	UserId            int      `json:"user_id"`
	UserName          string   `json:"user_name"`
	UserUsername      string   `json:"user_username"`
	UserEmail         string   `json:"user_email"`
	UserAvatar        string   `json:"user_avatar"`
	ProjectId         int      `json:"project_id"`
	Project           *Project `json:"project"`
	Commits           []Commit `json:"commits"`
	TotalCommitsCount int      `json:"total_commits_count"`
}

func (p PushEvent) GetID() string {
	return p.CheckoutSha
}

func (p PushEvent) GetVersion() string {
	return p.CheckoutSha
}

type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	AvatarURL         string `json:"avatar_url"`
	GitSSHURL         string `json:"git_ssh_url"`
	GitHTTPURL        string `json:"git_http_url"`
	Namespace         string `json:"namespace"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
	Homepage          string `json:"homepage"`
	URL               string `json:"url"`
	SSHURL            string `json:"ssh_url"`
	HTTPURL           string `json:"http_url"`
	WebURL            string `json:"web_url"`
}

type IssueEvent struct {
	ObjectKind       string            `json:"object_kind"`
	User             *gitlab.EventUser `json:"user"`
	Project          *Project          `json:"project"`
	ObjectAttributes *Issue            `json:"object_attributes"`
	Labels           []Label           `json:"labels"`
	Changes          struct {
	} `json:"changes"`
	State     string              `json:"state"`
	Assignees []*gitlab.EventUser `json:"assignees"`
}

func (i IssueEvent) getEvent() (event string, ok bool) {
	if i.ObjectAttributes == nil {
		ok = false
		return
	}
	event, ok = i.ObjectAttributes.Action, true
	return
}

func (i IssueEvent) GetID() string {
	return strconv.Itoa(i.ObjectAttributes.ID)
}

func (i IssueEvent) GetVersion() string {
	if i.ObjectAttributes.UpdatedAt != nil {
		return i.ObjectAttributes.UpdatedAt.Time().Format(time.RFC3339)
	}
	return i.ObjectAttributes.CreatedAt.Time().Format(time.RFC3339)
}

type TagEvent struct {
	ObjectKind   string   `json:"object_kind"`
	CheckoutSHA  string   `json:"checkout_sha"`
	Ref          string   `json:"ref"`
	UserID       int      `json:"user_id"`
	UserName     string   `json:"user_name"`
	UserUsername string   `json:"user_username"`
	UserAvatar   string   `json:"user_avatar"`
	UserEmail    string   `json:"user_email"`
	ProjectID    int      `json:"project_id"`
	Message      string   `json:"message"`
	Project      *Project `json:"project"`
}

func (t TagEvent) GetID() string {
	return t.Ref
}

func (t TagEvent) GetVersion() string {
	return t.Ref
}

type Label struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Color       string `json:"color"`
	ProjectID   int    `json:"project_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Template    bool   `json:"template"`
	Description string `json:"description"`
	Type        string `json:"type"`
	GroupID     int    `json:"group_id"`
}

type Issue struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	AssigneeID  int          `json:"assignee_id"`
	AuthorID    int          `json:"author_id"`
	ProjectID   int          `json:"project_id"`
	CreatedAt   *WebhookTime `json:"created_at"`
	UpdatedAt   *WebhookTime `json:"updated_at"`
	Position    int          `json:"position"`
	BranchName  string       `json:"branch_name"`
	Description string       `json:"description"`
	MilestoneID int          `json:"milestone_id"`
	State       string       `json:"state"`
	IID         int          `json:"iid"`
	URL         string       `json:"url"`
	Action      string       `json:"action"`
}

type NoteEvent struct {
	ObjectKind       string            `json:"object_kind"`
	User             *gitlab.EventUser `json:"user"`
	Project          *Project          `json:"project"`
	ObjectAttributes *Note             `json:"object_attributes"`
}

func (n NoteEvent) getEvent() (event string, ok bool) {
	if n.ObjectAttributes == nil {
		ok = false
		return
	}
	event, ok = n.ObjectAttributes.NoteableType, true
	return
}

func (n NoteEvent) GetID() string {
	return strconv.Itoa(n.ObjectAttributes.Id)
}

func (n NoteEvent) GetVersion() string {
	if n.ObjectAttributes.UpdatedAt != nil {
		return n.ObjectAttributes.UpdatedAt.Time().Format(time.RFC3339)
	}
	return n.ObjectAttributes.CreatedAt.Time().Format(time.RFC3339)
}

type Note struct {
	Id           int          `json:"id"`
	AuthorId     int          `json:"author_id"`
	Note         string       `json:"note"`
	NoteableId   int          `json:"noteable_id"`
	NoteableType string       `json:"noteable_type"`
	Description  string       `json:"description"`
	Url          string       `json:"url"`
	ProjectId    int          `json:"project_id"`
	System       bool         `json:"system"`
	UpdatedAt    *WebhookTime `json:"updated_at"`
	CreatedAt    *WebhookTime `json:"created_at"`
}

type MergeRequest struct {
	ID                     int          `json:"id"`
	TargetBranch           string       `json:"target_branch"`
	SourceBranch           string       `json:"source_branch"`
	SourceProjectID        int          `json:"source_project_id"`
	AuthorID               int          `json:"author_id"`
	AssigneeID             int          `json:"assignee_id"`
	AssigneeIDs            []int        `json:"assignee_ids"`
	Title                  string       `json:"title"`
	CreatedAt              *WebhookTime `json:"created_at"`
	UpdatedAt              *WebhookTime `json:"updated_at"`
	MilestoneID            int          `json:"milestone_id"`
	State                  string       `json:"state"`
	MergeStatus            string       `json:"merge_status"`
	TargetProjectID        int          `json:"target_project_id"`
	IID                    int          `json:"iid"`
	Description            string       `json:"description"`
	MergeError             string       `json:"merge_error"`
	MergeWhenBuildSucceeds bool         `json:"merge_when_build_succeeds"`
	MergeUserID            int          `json:"merge_user_id"`
	MergeCommitSHA         string       `json:"merge_commit_sha"`
	ApprovalsBeforeMerge   string       `json:"approvals_before_merge"`
	TimeEstimate           int          `json:"time_estimate"`
	WorkInProgress         bool         `json:"work_in_progress"`
	URL                    string       `json:"url"`
	Action                 string       `json:"action"`
}

type MergeRequestEvent struct {
	ObjectKind       string              `json:"object_kind"`
	User             *gitlab.EventUser   `json:"user"`
	Project          *Project            `json:"project"`
	ObjectAttributes *MergeRequest       `json:"object_attributes"`
	Labels           []Label             `json:"labels"`
	Assignees        []*gitlab.EventUser `json:"assignees"`
}

func (m MergeRequestEvent) getEvent() (event string, ok bool) {
	if m.ObjectAttributes == nil {
		ok = false
		return
	}
	ok = true
	event = m.ObjectAttributes.Action
	return
}

func (m MergeRequestEvent) GetID() string {
	return strconv.Itoa(m.ObjectAttributes.ID)
}

func (m MergeRequestEvent) GetVersion() string {
	if m.ObjectAttributes.UpdatedAt != nil {
		return m.ObjectAttributes.UpdatedAt.Time().Format(time.RFC3339)
	}
	return m.ObjectAttributes.CreatedAt.Time().Format(time.RFC3339)
}

type ReleaseEvent struct {
	CreatedAt   *WebhookTime `json:"created_at"`
	Description string       `json:"description"`
	Name        string       `json:"name"`
	ReleasedAt  string       `json:"released_at"`
	Tag         string       `json:"tag"`
	ObjectKind  string       `json:"object_kind"`
	Project     *Project     `json:"project"`
	Url         string       `json:"url"`
	Action      string       `json:"action"`
	Assets      *Assets      `json:"assets"`
	Commit      *Commit      `json:"commit"`
}

func (r ReleaseEvent) getEvent() (event string, ok bool) {
	ok = true
	event = r.Action
	return
}

func (r ReleaseEvent) GetID() string {
	return r.Tag
}

func (r ReleaseEvent) GetVersion() string {
	return r.Tag
}

type Commit struct {
	Id        any       `json:"id"` // in jobEvent is int, otherwise is string.
	Message   string    `json:"message"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"` // uses RFC3339 instead of WebhookTime, I know, it's confusing
	Url       string    `json:"url"`
	Author    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

type Assets struct {
	Count   int           `json:"count"`
	Links   []interface{} `json:"links"`
	Sources []struct {
		Format string `json:"format"`
		Url    string `json:"url"`
	} `json:"sources"`
}

func toLabelsFromStrings(ss []string) (labels []Label) {
	for _, s := range ss {
		labels = append(labels, Label{
			Title: s,
		})
	}
	return
}

func toMergeRequest(mr *gitlab.MergeRequest) *MergeRequest {
	return &MergeRequest{
		ID:                     mr.ID,
		TargetBranch:           mr.TargetBranch,
		SourceBranch:           mr.SourceBranch,
		SourceProjectID:        mr.SourceProjectID,
		AuthorID:               mr.Author.ID,
		AssigneeID:             getAssigneeID(mr.Assignee),
		AssigneeIDs:            getAssigneeIDs(mr.Assignees),
		Title:                  mr.Title,
		CreatedAt:              (*WebhookTime)(mr.CreatedAt),
		UpdatedAt:              (*WebhookTime)(mr.UpdatedAt),
		MilestoneID:            getMilestoneID(mr.Milestone),
		State:                  mr.State,
		MergeStatus:            mr.MergeStatus,
		TargetProjectID:        mr.TargetProjectID,
		IID:                    mr.IID,
		Description:            mr.Description,
		MergeError:             mr.MergeError,
		MergeWhenBuildSucceeds: mr.MergeWhenPipelineSucceeds,
		MergeUserID:            getUserID(mr.MergedBy),
		MergeCommitSHA:         mr.MergeCommitSHA,
		WorkInProgress:         mr.WorkInProgress,
		URL:                    mr.WebURL,
		Action:                 "open | close | reopen | update | merge",
	}
}

func getUserID(user *gitlab.BasicUser) int {
	if user == nil {
		return 0
	}
	return user.ID
}

func getAssigneeIDs(assignees []*gitlab.BasicUser) (result []int) {
	for _, assignee := range assignees {
		result = append(result, assignee.ID)
	}
	return
}

func getAssigneeID(assignee *gitlab.BasicUser) int {
	if assignee == nil {
		return 0
	}
	return assignee.ID
}

func toEventProject(project *gitlab.Project) *Project {
	if project == nil {
		return nil
	}
	return &Project{
		ID:                project.ID,
		Name:              project.Name,
		Description:       project.Description,
		AvatarURL:         project.AvatarURL,
		GitSSHURL:         project.SSHURLToRepo,
		GitHTTPURL:        project.HTTPURLToRepo,
		Namespace:         project.Namespace.Name,
		PathWithNamespace: project.PathWithNamespace,
		DefaultBranch:     project.DefaultBranch,
		Homepage:          project.WebURL,
		URL:               project.WebURL,
		SSHURL:            project.SSHURLToRepo,
		HTTPURL:           project.HTTPURLToRepo,
		WebURL:            project.WebURL,
	}
}

func toEventAssignees(assignees []*gitlab.IssueAssignee) []*gitlab.EventUser {
	if len(assignees) == 0 {
		return nil
	}
	result := make([]*gitlab.EventUser, len(assignees))
	for i, assignee := range assignees {
		result[i] = toEventAssignee(assignee)
	}
	return result
}

func toEventAssignees2(assignees []*gitlab.BasicUser) []*gitlab.EventUser {
	if len(assignees) == 0 {
		return nil
	}
	result := make([]*gitlab.EventUser, len(assignees))
	for i, assignee := range assignees {
		result[i] = &gitlab.EventUser{
			ID:        assignee.ID,
			Name:      assignee.Name,
			Username:  assignee.Username,
			AvatarURL: assignee.AvatarURL,
		}
	}
	return result
}

func toEventAssignee(assignee *gitlab.IssueAssignee) *gitlab.EventUser {
	if assignee == nil {
		return nil
	}
	return &gitlab.EventUser{
		ID:        assignee.ID,
		Name:      assignee.Name,
		Username:  assignee.Username,
		AvatarURL: assignee.AvatarURL,
	}
}

func toEventLabels(details []*gitlab.LabelDetails) []Label {
	if len(details) == 0 {
		return nil
	}
	result := make([]Label, len(details))
	for i, detail := range details {
		result[i] = Label{
			ID:          detail.ID,
			Title:       detail.Name,
			Color:       detail.Color,
			Description: detail.Description,
		}
	}
	return result
}

func toEventIssue(issue *gitlab.Issue) *Issue {
	return &Issue{
		ID:          issue.ID,
		Title:       issue.Title,
		AssigneeID:  getIssueAssigneeID(issue),
		AuthorID:    issue.Author.ID,
		ProjectID:   issue.ProjectID,
		CreatedAt:   (*WebhookTime)(issue.CreatedAt),
		UpdatedAt:   (*WebhookTime)(issue.UpdatedAt),
		Description: issue.Description,
		MilestoneID: getMilestoneID(issue.Milestone),
		IID:         issue.IID,
		URL:         issue.WebURL,
		State:       issue.State,
		Action:      "open || close || update || reopen",
	}
}

func getMilestoneID(m *gitlab.Milestone) int {
	if m == nil {
		return 0
	}
	return m.ID
}

func getIssueAssigneeID(issue *gitlab.Issue) int {
	if issue.Assignee == nil {
		return 0
	}
	return issue.Assignee.ID
}

func toEventUser(user *gitlab.User) *gitlab.EventUser {
	return &gitlab.EventUser{
		ID:        user.ID,
		Name:      user.Name,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Email:     user.Email,
	}
}

func objToObj(src any, dist any) (err error) {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("marshal src: %w", err)
	}
	err = json.Unmarshal(b, dist)
	if err != nil {
		return fmt.Errorf("unmarshal dist: %w", err)
	}
	return nil
}

func getTimestamp(at *time.Time) time.Time {
	if at != nil {
		return *at
	}
	return time.Now()
}

var _ trigger.SampleData = (*jobEvent)(nil)
var _ conditionalEvent = (*jobEvent)(nil)

type jobEvent struct {
	ObjectKind         string            `json:"object_kind"`
	Ref                string            `json:"ref"`
	Tag                bool              `json:"tag"`
	BeforeSha          string            `json:"before_sha"`
	Sha                string            `json:"sha"`
	BuildId            int               `json:"build_id"`
	BuildName          string            `json:"build_name"`
	BuildStage         string            `json:"build_stage"`
	BuildStatus        string            `json:"build_status"`
	BuildCreatedAt     *WebhookTime      `json:"build_created_at"`
	BuildStartedAt     *WebhookTime      `json:"build_started_at"`
	BuildFinishedAt    *WebhookTime      `json:"build_finished_at"`
	BuildDuration      any               `json:"build_duration"`
	BuildAllowFailure  bool              `json:"build_allow_failure"`
	BuildFailureReason string            `json:"build_failure_reason"`
	RetriesCount       int               `json:"retries_count"`
	PipelineId         int               `json:"pipeline_id"`
	ProjectId          int               `json:"project_id"`
	ProjectName        string            `json:"project_name"`
	User               *gitlab.EventUser `json:"user"`
	Commit             *Commit           `json:"commit"`
}

func (j jobEvent) getEvent() (event string, ok bool) {
	return j.BuildStatus, true
}

func (j jobEvent) GetID() string {
	return j.Sha
}

func (j jobEvent) GetVersion() string {
	return j.Sha
}

var _ trigger.SampleData = (*pipelineEvent)(nil)
var _ conditionalEvent = (*pipelineEvent)(nil)

type Pipeline struct {
	Id         int          `json:"id"`
	Iid        int          `json:"iid"`
	Ref        string       `json:"ref"`
	Tag        bool         `json:"tag"`
	Sha        string       `json:"sha"`
	BeforeSha  string       `json:"before_sha"`
	Source     string       `json:"source"`
	Status     string       `json:"status"`
	Stages     []string     `json:"stages"`
	CreatedAt  *WebhookTime `json:"created_at"`
	FinishedAt *WebhookTime `json:"finished_at"`
	Duration   int          `json:"duration"`
	Variables  []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
}

type pipelineEvent struct {
	ObjectKind       string            `json:"object_kind"`
	ObjectAttributes Pipeline          `json:"object_attributes"`
	MergeRequest     MergeRequest      `json:"merge_request"`
	User             *gitlab.EventUser `json:"user"`
	Project          *Project          `json:"project"`
	Commit           Commit            `json:"commit"`
	Builds           []Build           `json:"builds"`
}

type Build struct {
	ID           int          `json:"id"`
	Stage        string       `json:"stage"`
	Name         string       `json:"name"`
	Status       string       `json:"status"`
	CreatedAt    *WebhookTime `json:"created_at"`
	StartedAt    *WebhookTime `json:"started_at"`
	FinishedAt   *WebhookTime `json:"finished_at"`
	When         string       `json:"when"`
	Manual       bool         `json:"manual"`
	AllowFailure bool         `json:"allow_failure"`
}

func (p pipelineEvent) getEvent() (event string, ok bool) {
	return p.ObjectAttributes.Status, true
}

func (p pipelineEvent) GetID() string {
	return strconv.Itoa(p.ObjectAttributes.Id)
}

func (p pipelineEvent) GetVersion() string {
	return strconv.Itoa(p.ObjectAttributes.Id)
}

type MemberTrigger struct {
	Trigger
}

func (m MemberTrigger) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("triggerMember"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(MemberTrigger)
		},
	}
}

func (m MemberTrigger) Run(c *workflow.NodeContext) (any, error) {
	return m.unmarshalBody(&memberEvent{})
}

func (m MemberTrigger) PreFilter(configObj any, data []byte) (shouldAbort bool, err error) {
	return m.preFilterHTTP(configObj, data, &memberEvent{})
}

func (m MemberTrigger) GetSampleList(c *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	return nil, fmt.Errorf("implement me")
}

// Create TODO(sword): member webhook can't create by api.
// https://docs.gitlab.com/ee/api/groups.html#add-group-hook
func (m MemberTrigger) Create(c trigger.WebhookContext) (map[string]any, error) {
	triggerConfig, ok := c.GetConfigObject().(*TriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid trigger config")
	}

	triggerConfig.Scope = "group"
	return m.create(c, "member_events")
}

type memberEvent struct {
	CreatedAt    *WebhookTime `json:"created_at"`
	UpdatedAt    *WebhookTime `json:"updated_at"`
	GroupName    string       `json:"group_name"`
	GroupPath    string       `json:"group_path"`
	GroupID      int          `json:"group_id"`
	UserUsername string       `json:"user_username"`
	UserName     string       `json:"user_name"`
	UserEmail    string       `json:"user_email"`
	UserID       int          `json:"user_id"`
	GroupAccess  string       `json:"group_access"`
	GroupPlan    interface{}  `json:"group_plan"`
	ExpiresAt    *WebhookTime `json:"expires_at"`
	EventName    string       `json:"event_name"`
}

func (m memberEvent) getEvent() (event string, ok bool) {
	return m.EventName, true
}
