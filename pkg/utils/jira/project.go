package jira

import (
	"context"
	"fmt"
	"net/http"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/set"
)

// ListAllStatuses Get all statuses for project, all status from different issue type and merged together.
//
// Ref: https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-projects/#api-rest-api-3-project-projectidorkey-statuses-get
func (c *Client) ListAllStatuses(ctx context.Context, projectKeyOrID string) (statuses []ProjectStatus, err error) {
	var resp ListAllStatusesResp

	err = c.call(ctx, callOpt{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/rest/api/3/project/%s/statuses", projectKeyOrID),
		Dest:   &resp,
	})
	if err != nil {
		return
	}

	seen := make(set.Set[string])
	for _, issueType := range resp {
		for _, status := range issueType.Statuses {
			if seen.Has(status.ID) {
				continue
			}
			seen.Add(status.ID)

			statuses = append(statuses, status)
		}
	}

	return
}

type ListAllStatusesResp []struct {
	Self     string          `json:"self"`
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Subtask  bool            `json:"subtask"`
	Statuses []ProjectStatus `json:"statuses"`
}

type ProjectStatus struct {
	Self             string         `json:"self"`
	Description      string         `json:"description"`
	IconURL          string         `json:"iconUrl"`
	Name             string         `json:"name"`
	UntranslatedName string         `json:"untranslatedName"`
	ID               string         `json:"id"`
	StatusCategory   StatusCategory `json:"statusCategory"`
}
