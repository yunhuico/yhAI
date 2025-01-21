package licensedot

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type CreateLicense struct {
	BaseNode

	Name               string `json:"name"`
	Email              string `json:"email"`
	StartsAt           string `json:"startsAt"`
	ExpiresAt          string `json:"expiresAt"`
	UsersCount         int    `json:"usersCount,omitempty"`
	PreviousUsersCount int    `json:"previousUsersCount,omitempty"`
	TrueupQuantity     int    `json:"trueupQuantity,omitempty"`
	Trial              int    `json:"trial,omitempty"`
	PlanCode           string `json:"planCode"`
	Notes              string `json:"notes"`
	Company            string `json:"company"`
}

func (c *CreateLicense) Run(ctx *workflow.NodeContext) (any, error) {
	err := c.valid()
	if err != nil {
		return nil, fmt.Errorf("license param valid error: %w", err)
	}

	userCount := strconv.Itoa(c.UsersCount)
	preCount := strconv.Itoa(c.PreviousUsersCount)
	trial := strconv.Itoa(c.Trial)
	trueupQuantity := strconv.Itoa(c.TrueupQuantity)

	license := url.Values{
		wrapWithLicense("skip_trial_validation"):   {"true"},
		wrapWithLicense("name"):                    {c.Name},
		wrapWithLicense("company"):                 {c.Company},
		wrapWithLicense("previous_users_count"):    {preCount},
		wrapWithLicense("email"):                   {c.Email},
		wrapWithLicense("notes"):                   {c.Notes},
		wrapWithLicense("starts_at"):               {c.StartsAt},
		wrapWithLicense("expires_at"):              {c.ExpiresAt},
		wrapWithLicense("zuora_subscription_id"):   {""},
		wrapWithLicense("zuora_subscription_name"): {""},
		wrapWithLicense("plan_code"):               {c.PlanCode},
		wrapWithLicense("trial"):                   {trial},
		wrapWithLicense("trueup_quantity"):         {trueupQuantity},
		wrapWithLicense("users_count"):             {userCount},
	}
	data, err := c.client.createLicense(ctx.Context(), license)
	if err != nil {
		return nil, fmt.Errorf("create license error: %w", err)
	}
	result := map[string]any{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("decode create license result error: %w", err)
	}
	return result, nil
}

func (c *CreateLicense) valid() error {
	if c.Name == "" {
		return errors.New("name can not be blank")
	}
	if c.Email == "" {
		return errors.New("email can not be blank")
	}
	_, err := mail.ParseAddress(c.Email)
	if err != nil {
		return err
	}
	if c.PlanCode == "" {
		return errors.New("plan code can not be blank")
	}
	return nil
}

func (c *CreateLicense) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/licensedot#createLicense")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(CreateLicense)
		},
		InputForm: spec.InputSchema,
	}
}

func wrapWithLicense(name string) string {
	return fmt.Sprintf("license[%s]", name)
}
