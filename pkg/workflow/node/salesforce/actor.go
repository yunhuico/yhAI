package salesforce

import (
	"encoding/json"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

type QueryAccountByID struct {
	BaseNode
	AccountID string `json:"accountId"`
}

type QueryResult struct {
	TotalSize int  `json:"totalSize"`
	Done      bool `json:"done"`
	Records   []any
}

type QueryContactByID struct {
	BaseNode
	ContactID string `json:"contactId"`
}

func (q *QueryContactByID) Run(c *workflow.NodeContext) (any, error) {
	info, err := q.client.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("get user meta info error: %w", err)
	}

	reqUrl := info.sobjectURL() + "Contact/" + q.ContactID
	data, err := q.client.get(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("get contact attribute error: %w", err)
	}
	res := map[string]any{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, fmt.Errorf("decode data error:%w", err)
	}
	return res, nil
}

func (q *QueryContactByID) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/salesforce#queryContact")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(QueryContactByID)
		},
		InputForm: spec.InputSchema,
	}
}

func (q *QueryAccountByID) Run(c *workflow.NodeContext) (any, error) {
	info, err := q.client.GetUserInfo()
	if err != nil {
		return nil, fmt.Errorf("get user meta info error: %w", err)
	}
	reqUrl := info.sobjectURL() + "Account/" + q.AccountID
	data, err := q.client.get(reqUrl)
	if err != nil {
		return nil, fmt.Errorf("get account attribute error: %w", err)
	}

	result := map[string]any{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("decode query result error: %w", err)
	}
	return result, nil
}

func (q *QueryAccountByID) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/salesforce#queryAccount")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(QueryAccountByID)
		},
		InputForm: spec.InputSchema,
	}
}

type ListContact struct {
	BaseNode `json:"-"`

	workflow.ListPagination
}

func (c *ListContact) Run(ctx *workflow.NodeContext) (any, error) {
	return c.run(ctx)
}

func (c *ListContact) run(ctx *workflow.NodeContext) (*queryResponse, error) {
	// The SOQL FIELDS function must have a LIMIT of at most 200
	if c.PerPage < 0 {
		return nil, errors.New("per page cannot be negative")
	}
	if c.PerPage == 0 || c.PerPage > 200 {
		c.PerPage = 200
	}
	var (
		like *likeSql
		err  error
	)
	if c.Search != "" {
		like, err = WhereLike("Name", c.Search)
		if err != nil {
			return nil, fmt.Errorf("build like sql: %w", err)
		}
	}
	resp, err := c.client.getSObjectCollection("contact", like, c.PerPage, (c.Page-1)*c.PerPage)
	if err != nil {
		return nil, fmt.Errorf("get contact collection error: %w", err)
	}
	return resp, nil
}

func (c *ListContact) QueryFieldResultList(ctx *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	resp, err := c.run(ctx)
	if err != nil {
		return
	}
	for _, record := range resp.Records {
		name := record["Name"].(string)
		value := record["Id"].(string)
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: name,
			Value: value,
		})
	}
	return
}

func (c *ListContact) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/salesforce#listContact")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListContact)
		},
		InputForm: spec.InputSchema,
	}
}

type ListAccount struct {
	BaseNode `json:"-"`

	workflow.ListPagination
}

func (c *ListAccount) Run(ctx *workflow.NodeContext) (any, error) {
	return c.run(ctx)
}

func (c *ListAccount) run(ctx *workflow.NodeContext) (*queryResponse, error) {
	// The SOQL FIELDS function must have a LIMIT of at most 200
	if c.PerPage < 0 {
		return nil, errors.New("per page cannot be negative")
	}
	if c.PerPage == 0 || c.PerPage > 200 {
		c.PerPage = 200
	}
	var (
		like *likeSql
		err  error
	)
	if c.Search != "" {
		like, err = WhereLike("Name", c.Search)
		if err != nil {
			return nil, fmt.Errorf("build like sql: %w", err)
		}
	}
	resp, err := c.client.getSObjectCollection("account", like, c.PerPage, (c.Page-1)*c.PerPage)
	if err != nil {
		return nil, fmt.Errorf("get account collection error: %w", err)
	}
	return resp, nil
}

func (c *ListAccount) QueryFieldResultList(ctx *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	resp, err := c.run(ctx)
	if err != nil {
		return
	}
	for _, record := range resp.Records {
		name := record["Name"].(string)
		value := record["Id"].(string)
		result.Items = append(result.Items, workflow.QueryFieldItem{
			Label: name,
			Value: value,
		})
	}
	return
}

func (c *ListAccount) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/salesforce#listAccount")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(ListAccount)
		},
		InputForm: spec.InputSchema,
	}
}
