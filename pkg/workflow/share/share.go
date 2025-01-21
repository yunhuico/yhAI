package share

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

// Annotations represents the metadata of the workflow,
// Author represents the owner name,
// workflow should either belong to user or organization
// CreateAt represents the exported time
type Annotations struct {
	Author    string    `json:"author" yaml:"author,omitempty"`
	AvatarURL string    `json:"avatarUrl" yaml:"avatarUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
}

type ExportedWorkflow struct {
	model.WorkflowWithNodes `yaml:",inline"`
	Annotations             Annotations `json:"annotations" yaml:"annotations"`
}

type AuthorStore interface {
	GetUserByID(ctx context.Context, id int) (user model.User, err error)
	GetOrganizationByID(ctx context.Context, id int) (organization model.Organization, err error)
}

func Export(ctx context.Context, workflow *model.WorkflowWithNodes, db AuthorStore) (workflowYaml string, err error) {
	authorName, authorAvator, err := GetAuthorNameAvatar(ctx, workflow.OwnerType, workflow.OwnerID, db)
	if err != nil {
		err = fmt.Errorf("querying user name or organization name: %w", err)
		return
	}

	// delete id and owner information
	workflow.ID = ""
	workflow.OwnerType = ""
	workflow.OwnerID = 0

	// delete credential information and generate new uid
	for i := range workflow.Nodes {
		workflow.Nodes[i].CredentialID = ""
		workflow.Nodes[i].WorkflowID = ""
		workflow.Nodes[i].TestingStatus = ""
	}
	old2NewIDMap, err := createIDMap(workflow)
	if err != nil {
		return
	}

	// replace transition in workflow
	err = replaceNodesID(workflow, old2NewIDMap)
	if err != nil {
		err = fmt.Errorf("replacing nodes id: %w", err)
		return
	}

	exportedWorkflow := ExportedWorkflow{
		WorkflowWithNodes: *workflow,
		Annotations: Annotations{
			Author:    authorName,
			AvatarURL: authorAvator,
			CreatedAt: time.Now(),
		},
	}

	yamlBytes, err := yaml.Marshal(exportedWorkflow)
	if err != nil {
		err = fmt.Errorf("encoding workflow yaml: %w", err)
		return "", err
	}

	// replace old node id in various input with new id
	workflowYaml = replaceNodesExpr(string(yamlBytes), old2NewIDMap)

	return workflowYaml, nil
}

func SanitizeImport(workflow *model.WorkflowWithNodes) (err error) {
	// set current time
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()
	// remove related workflowID and credentialID
	for i := range workflow.Nodes {
		workflow.Nodes[i].CredentialID = ""
		workflow.Nodes[i].WorkflowID = ""
		workflow.Nodes[i].TestingStatus = model.NodeTestingDefaultStatus
	}

	yamlBytes, err := yaml.Marshal(workflow)
	if err != nil {
		err = fmt.Errorf("unmarshaling workflow yaml: %w", err)
		return
	}

	old2NewIDMap, err := createIDMap(workflow)
	if err != nil {
		err = fmt.Errorf("creating id mapping: %w", err)
		return
	}

	workflowYaml := replaceNodesExpr(string(yamlBytes), old2NewIDMap)

	err = yaml.Unmarshal([]byte(workflowYaml), workflow)
	if err != nil {
		err = fmt.Errorf("unmarshaling workflow yaml: %w", err)
		return
	}

	workflow.ID, err = utils.NanoID()
	if err != nil {
		err = fmt.Errorf("generating nano id for workflow: %w", err)
		return
	}

	err = replaceNodesID(workflow, old2NewIDMap)
	if err != nil {
		err = fmt.Errorf("replacing nodes id: %w", err)
		return
	}

	return
}

func createIDMap(workflow *model.WorkflowWithNodes) (map[string]string, error) {
	var err error
	old2NewIDMap := make(map[string]string)

	for _, node := range workflow.Nodes {
		old2NewIDMap[node.ID], err = utils.NanoID()
		if err != nil {
			return nil, fmt.Errorf("generating id for node: %w", err)
		}
	}

	return old2NewIDMap, err
}

// regexNodeValidIdentifier checks whether the node in input field is valid
// N {{}}
// N {{ }}
// N {{ .Node.id.output.id}}
// Y {{.Node.id.output.id}}
// Y {{ .Node.id.output.labels.[].id }}
// Y {{ .Node.node1.output.assignees[].emails[]? }}
var regexNodeValidIdentifier = regexp.MustCompile(`{{ *\.Node\.(\w+)\.[\w.\[\]\?]+ *}}`)

func replaceNodesExpr(workflowYaml string, old2NewIDMap map[string]string) string {
	rules := make([]string, 0, 2*len(old2NewIDMap))
	for k, v := range old2NewIDMap {
		rules = append(rules, k, v)
	}

	replacer := strings.NewReplacer(rules...)

	return regexNodeValidIdentifier.ReplaceAllStringFunc(workflowYaml, func(got string) string {
		return replacer.Replace(got)
	})
}

func replaceNodesID(workflow *model.WorkflowWithNodes, old2NewIDMap map[string]string) error {
	workflow.StartNodeID = old2NewIDMap[workflow.StartNodeID]

	for i := range workflow.Nodes {
		node := &workflow.Nodes[i]

		node.ID = old2NewIDMap[node.ID]
		node.Transition = old2NewIDMap[node.Transition]

		switch node.Class {
		case validate.ForeachClass:
			var foreachNode validate.LoopFromListNode
			err := trans.MapToStruct(node.Data.InputFields, &foreachNode)
			if err != nil {
				return fmt.Errorf("foreach input field error: %s", err)
			}
			foreachNode.Transition = old2NewIDMap[foreachNode.Transition]

			inputField, err := trans.StructToMap(foreachNode)
			if err != nil {
				return fmt.Errorf("trans foreach struct to input field error: %s", err)
			}
			node.Data.InputFields = inputField

		case validate.SwitchClass:
			var switchNode validate.SwitchLogicNode
			err := trans.MapToStruct(node.Data.InputFields, &switchNode)
			if err != nil {
				return fmt.Errorf("switch input field error: %s", err)
			}
			for j, path := range switchNode.Paths {
				switchNode.Paths[j].Transition = old2NewIDMap[path.Transition]
			}

			inputField, err := trans.StructToMap(switchNode)
			if err != nil {
				return fmt.Errorf("trans switch struct to input field error: %s", err)
			}
			node.Data.InputFields = inputField
		}
	}

	return nil
}

func GetAuthorNameAvatar(ctx context.Context, ownerType model.OwnerType, ownerID int, db AuthorStore) (authorName, avatarURL string, err error) {
	switch ownerType {
	case model.OwnerTypeUser:
		var user model.User
		user, err = db.GetUserByID(ctx, ownerID)
		if err != nil {
			err = fmt.Errorf("querying user %d: %w", ownerID, err)
			return
		}
		authorName, avatarURL = user.Name, user.AvatarURL
	case model.OwnerTypeOrganization:
		var org model.Organization
		org, err = db.GetOrganizationByID(ctx, ownerID)
		if err != nil {
			err = fmt.Errorf("querying org %d: %w", ownerID, err)
			return
		}
		authorName, avatarURL = org.Name, org.AvatarURL
	default:
		err = fmt.Errorf("unexpected ownerType %q", authorName)
	}
	return
}
