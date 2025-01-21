package payload

import (
	"fmt"
	"net/url"
	"strings"
)

type QueryFieldSelectReq struct {
	Class        string         `json:"class"`
	WorkflowID   string         `json:"workflowId"`
	CredentialID string         `json:"credentialId"`
	Page         int            `json:"page"`
	PerPage      int            `json:"perPage"`
	Search       string         `json:"search"`
	NextCursor   string         `json:"nextCursor"`
	InputFields  map[string]any `json:"inputFields"`
}

// Normalize dont validate the request in Normalize().
func (r *QueryFieldSelectReq) Normalize() (err error) {
	class, transMap, err := r.extractClassAndTransMap()
	if err != nil {
		return
	}
	r.Class = class

	// transform the inputFields, for example:
	// {"projectId": 1, "projectId2": 2}
	// define transMap as {"projectId2": "projectId"}
	// then in the field search business logic, will use {"projectId": 2} as context.
	for originKey, distKey := range transMap {
		r.InputFields[distKey] = r.InputFields[originKey]
		if originKey != distKey {
			delete(r.InputFields, originKey)
		}
	}

	if r.InputFields == nil {
		r.InputFields = map[string]any{}
	}
	r.InputFields["page"] = r.Page
	r.InputFields["perPage"] = r.PerPage
	r.InputFields["search"] = r.Search
	r.InputFields["nextCursor"] = r.NextCursor

	return
}

func (r QueryFieldSelectReq) extractClassAndTransMap() (class string, transMap map[string]string, err error) {
	tmp := strings.Split(r.Class, "?")
	class = tmp[0]
	if len(tmp) == 1 {
		return
	}

	query, err := url.ParseQuery(tmp[1])
	if err != nil {
		err = fmt.Errorf("parse class as query: %w", err)
		return
	}

	transMap = make(map[string]string, len(query))
	for key := range query {
		v := query.Get(key)
		if v == "" {
			err = fmt.Errorf("%s should provide a value like '%s=foo'", key, key)
			return
		}
		transMap[key] = query.Get(key)
	}
	return
}
