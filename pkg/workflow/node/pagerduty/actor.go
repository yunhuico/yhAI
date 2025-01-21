package pagerduty

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)
	adapter.RegisterCredentialTestingFunc(testCredential)

	workflow.RegistryNodeMeta(&ListCurrentOnCallUser{})
	workflow.RegistryNodeMeta(&CreateIncident{})
	workflow.RegistryNodeMeta(&ListSchedules{})
	workflow.RegistryNodeMeta(&ListUsers{})
	workflow.RegistryNodeMeta(&ListServices{})
}

func testCredential(ctx context.Context, credentialType model.CredentialType, fields model.InputFields) (err error) {
	client := pagerduty.NewClient(fields.GetString("accessToken"))
	_, err = client.ListAbilitiesWithContext(ctx)
	if err != nil {
		// ignore error, the error is very confusing.
		// error: HTTP response with status code 401 does not contain Content-Type: application/json
		err = errors.New("the access token is invalid")
		return
	}
	return
}

type CreateIncident struct {
	basePagerDutyNode

	AccountEmail string `json:"accountEmail"`
	ServiceID    string `json:"serviceId"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
}

func (i *CreateIncident) UltrafoxNode() workflow.NodeMeta {
	const class = "ultrafox/pagerduty#createIncident"

	spec := adapter.MustLookupSpec(class)
	return workflow.NodeMeta{
		Class: class,
		New: func() workflow.Node {
			return new(CreateIncident)
		},
		InputForm: spec.InputSchema,
	}
}

func (i *CreateIncident) Run(c *workflow.NodeContext) (any, error) {
	incident, err := i.client.CreateIncidentWithContext(c.Context(), i.AccountEmail, &pagerduty.CreateIncidentOptions{
		Title: i.Title,
		Service: &pagerduty.APIReference{
			ID:   i.ServiceID,
			Type: "service_reference",
		},
		Body: &pagerduty.APIDetails{
			Type:    "incident_body",
			Details: i.Detail,
		},
	})
	if err != nil {
		err = fmt.Errorf("creating incident: %w", err)
		return nil, err
	}

	return incident, nil
}

type ListCurrentOnCallUser struct {
	basePagerDutyNode

	ScheduleID string `json:"scheduleId"`
}

func (l *ListCurrentOnCallUser) UltrafoxNode() workflow.NodeMeta {
	const class = "ultrafox/pagerduty#listCurrentOnCallUser"

	spec := adapter.MustLookupSpec(class)
	return workflow.NodeMeta{
		Class: class,
		New: func() workflow.Node {
			return new(ListCurrentOnCallUser)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListCurrentOnCallUser) Run(c *workflow.NodeContext) (any, error) {
	now := time.Now()
	users, err := l.client.ListOnCallUsersWithContext(c.Context(), l.ScheduleID, pagerduty.ListOnCallUsersOptions{
		Since: now.Format(time.RFC3339),
		Until: now.Add(time.Second).Format(time.RFC3339),
	})
	if err != nil {
		err = fmt.Errorf("listing on-call users: %w", err)
		return nil, err
	}
	if len(users) == 0 {
		err = errors.New("no user is on call")
		return nil, err
	}

	user := users[0]
	return map[string]string{
		"name":  user.Name,
		"email": user.Email,
	}, nil
}

type ListSchedules struct {
	basePagerDutyNode `json:"-"`

	workflow.ListPagination
}

func (l *ListSchedules) UltrafoxNode() workflow.NodeMeta {
	const class = "ultrafox/pagerduty#listSchedules"

	spec := adapter.MustLookupSpec(class)
	return workflow.NodeMeta{
		Class: class,
		New: func() workflow.Node {
			return new(ListSchedules)
		},
		InputForm: spec.InputSchema,
	}
}

type Schedule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Self        string `json:"self"`
	HTMLURL     string `json:"htmlUrl"`
	TimeZone    string `json:"timezone"`
	Description string `json:"description"`
}

func (l *ListSchedules) Run(c *workflow.NodeContext) (result any, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	schedules, err := l.client.ListSchedulesWithContext(c.Context(), pagerduty.ListSchedulesOptions{
		Limit:  uint(perPage),
		Offset: uint(perPage * page),
		Total:  false,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("calling pagerduty API: %w", err)
		return
	}

	sanitized := make([]Schedule, len(schedules.Schedules))
	for i, v := range schedules.Schedules {
		sanitized[i] = Schedule{
			ID:          v.ID,
			Summary:     v.Summary,
			Self:        v.Self,
			HTMLURL:     v.HTMLURL,
			Name:        v.Name,
			TimeZone:    v.TimeZone,
			Description: v.Description,
		}
	}

	result = sanitized
	return
}

func (l *ListSchedules) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	schedules, err := l.client.ListSchedulesWithContext(c.Context(), pagerduty.ListSchedulesOptions{
		Limit:  uint(perPage),
		Offset: uint(perPage * page),
		Total:  true,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("listing pagerduty schedules: %w", err)
		return
	}

	result.NoMore = !schedules.More
	result.Items = make([]workflow.QueryFieldItem, len(schedules.Schedules))

	for i, v := range schedules.Schedules {
		result.Items[i] = workflow.QueryFieldItem{
			Label: v.Name,
			Value: v.ID,
		}
	}

	return
}

type ListUsers struct {
	basePagerDutyNode `json:"-"`

	workflow.ListPagination
}

func (l *ListUsers) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	users, err := l.client.ListUsersWithContext(c.Context(), pagerduty.ListUsersOptions{
		Limit:  uint(perPage),
		Offset: uint(perPage * page),
		Total:  true,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("listing Pagerduty users: %w", err)
		return
	}

	result.NoMore = !users.More
	result.Items = make([]workflow.QueryFieldItem, len(users.Users))
	for i, v := range users.Users {
		result.Items[i] = workflow.QueryFieldItem{
			Label: v.Name,
			Value: v.Email,
		}
	}

	return
}

func (l *ListUsers) UltrafoxNode() workflow.NodeMeta {
	const class = "ultrafox/pagerduty#listUsers"

	spec := adapter.MustLookupSpec(class)
	return workflow.NodeMeta{
		Class: class,
		New: func() workflow.Node {
			return new(ListUsers)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListUsers) Run(c *workflow.NodeContext) (result any, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	users, err := l.client.ListUsersWithContext(c.Context(), pagerduty.ListUsersOptions{
		Limit:  uint(perPage),
		Offset: uint(page * perPage),
		Total:  true,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("listing Pagerduty users: %w", err)
		return
	}

	result = users.Users
	return
}

type ListServices struct {
	basePagerDutyNode `json:"-"`

	workflow.ListPagination
}

func (l *ListServices) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	resp, err := l.client.ListServicesWithContext(c.Context(), pagerduty.ListServiceOptions{
		Limit:  uint(perPage),
		Offset: uint(page * perPage),
		Total:  true,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("listing Pagerduty services: %w", err)
		return
	}

	result.NoMore = !resp.More
	result.Items = make([]workflow.QueryFieldItem, len(resp.Services))
	for i, v := range resp.Services {
		result.Items[i] = workflow.QueryFieldItem{
			Label: v.Name,
			Value: v.ID,
		}
	}

	return
}

func (l *ListServices) UltrafoxNode() workflow.NodeMeta {
	const class = "ultrafox/pagerduty#listServices"

	spec := adapter.MustLookupSpec(class)
	return workflow.NodeMeta{
		Class: class,
		New: func() workflow.Node {
			return new(ListServices)
		},
		InputForm: spec.InputSchema,
	}
}

func (l *ListServices) Run(c *workflow.NodeContext) (result any, err error) {
	perPage := cappedNum(l.PerPage, 0, 100)
	page := cappedNum(l.Page, 0, 100)

	resp, err := l.client.ListServicesWithContext(c.Context(), pagerduty.ListServiceOptions{
		Limit:  uint(perPage),
		Offset: uint(page * perPage),
		Total:  false,
		Query:  l.Search,
	})
	if err != nil {
		err = fmt.Errorf("listing Pagerduty services: %w", err)
		return
	}

	result = resp.Services
	return
}

func cappedNum(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}
