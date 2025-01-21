package workflow

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/yaml"
)

func Apply(ctx context.Context, triggerRegistry *trigger.Registry, workflow *model.WorkflowWithNodes, tx model.Operator) (workflowID string, err error) {
	workflow.Status = model.WorkflowStatusDisabled // apply workflow default inactive status

	report := validate.ValidateWorkflow(*workflow, validate.WithImportWorkflowOpt())
	if report.ExistsFatal() {
		return "", report
	}

	if report.ExistsReport() {
		warning := bytes.NewBuffer(nil)
		report.Warning(warning)
		fmt.Println(warning)
	}

	if workflow.ID == "" {
		err = errors.New("workflow's id can not be empty")
		return
	}
	if workflow.StartNodeID == "" {
		err = errors.New("workflow's startNodeId can not be empty")
		return
	}
	if workflow.Name == "" {
		err = errors.New("workflow's name can not be empty")
		return
	}
	if len(workflow.Nodes) == 0 {
		err = errors.New("workflow must have at least one node")
		return
	}

	for i := range workflow.Nodes {
		// assign foreigner key on node
		workflow.Nodes[i].WorkflowID = workflow.ID

		node := workflow.Nodes[i]
		if node.CredentialID != "" {
			// check the related credential exists
			_, err = tx.GetCredentialByID(ctx, node.CredentialID)
			if err != nil {
				err = fmt.Errorf("querying credential %q of node %q", node.CredentialID, node.ID)
				return
			}
		}
	}

	defer func() {
		triggerNode, ok := workflow.GetNodeByID(workflow.StartNodeID)
		if ok {
			err = InitTrigger(ctx, triggerRegistry, tx, workflowID, &triggerNode)
			if err != nil {
				err = fmt.Errorf("initing trigger: %w", err)
				return
			}
		}
	}()

	existedWorkflow, err := tx.GetWorkflowByID(ctx, workflow.ID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.InsertWorkflow(ctx, &workflow.Workflow)
		if err != nil {
			err = fmt.Errorf("inserting workflow: %w", err)
			return
		}
		for _, node := range workflow.Nodes {
			err = tx.InsertNode(ctx, &node)
			if err != nil {
				err = fmt.Errorf("inserting node %q: %w", node.ID, err)
				return
			}
		}

		return workflow.ID, nil
	}
	if err != nil {
		err = fmt.Errorf("checking existed workflow: %w", err)
		return
	}

	// existed workflow
	existedNodes, err := tx.GetNodesByWorkflowID(ctx, existedWorkflow.ID)
	if err != nil {
		err = fmt.Errorf("checking existed nodes of workflow %s(id: %s): %w", existedWorkflow.Name, existedWorkflow.ID, err)
		return
	}

	var (
		expectedNodes = make(map[string]*model.Node, len(workflow.Nodes))
		visitedNodes  = make(map[string]bool, len(workflow.Nodes))
	)
	for i := range workflow.Nodes {
		name := workflow.Nodes[i].Name
		expectedNodes[name] = &workflow.Nodes[i]
		visitedNodes[name] = false
	}

	err = tx.UpdateWorkflowByID(ctx, &workflow.Workflow)
	if err != nil {
		err = fmt.Errorf("updating existed workflow: %w", err)
		return
	}

	for _, existedNode := range existedNodes {
		expected, ok := expectedNodes[existedNode.Name]
		if !ok {
			err = tx.DeleteNodeByID(ctx, existedNode.ID)
			if err != nil {
				err = fmt.Errorf("pruning abandoned node %s(id: %s): %w", existedNode.Name, existedNode.ID, err)
				return
			}
			continue
		}
		expected.ID = existedNode.ID
		err = tx.UpdateNodeByID(ctx, expected)
		if err != nil {
			err = fmt.Errorf("updating existed node %s(id: %s): %w", expected.Name, expected.ID, err)
			return
		}
		visitedNodes[expected.Name] = true
	}

	for name, visited := range visitedNodes {
		if visited {
			continue
		}

		node := expectedNodes[name]
		node.WorkflowID = workflow.ID
		err = tx.InsertNode(ctx, expectedNodes[name])
		if err != nil {
			err = fmt.Errorf("creating new node %q: %w", name, err)
			return
		}
	}

	return workflow.ID, nil
}

// InitTrigger creates a empty trigger first, then enables the trigger if the trigger config EnableTriggerAtFirst=true.
func InitTrigger(ctx context.Context, triggerRegistry *trigger.Registry, tx model.Operator, workflowID string, node *model.Node) (err error) {
	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(node.Class)
	adapter := adapterManager.LookupAdapter(spec.AdapterClass)

	var trigger model.Trigger
	trigger, err = tx.GetOrCreateTrigger(ctx, workflowID, node)
	if err != nil {
		err = fmt.Errorf("no trigger related to workflow: %w", err)
		return
	}

	// Is this trigger EnableTriggerAtFirst, should recall EnableTrigger logic.
	// todo(sword): refactor https://jihulab.com/ultrafox/ultrafox/-/issues/503
	if spec.EnableTriggerAtFirst {
		// It's only in importing workflow if adapter require authentication and node.credentialID is empty.
		if adapter.RequireAuth() && node.CredentialID == "" {
			return
		}

		triggerWithNode := model.TriggerWithNode{
			Trigger: trigger,
			Node:    node,
		}
		err = triggerRegistry.EnableTrigger(ctx, tx, &triggerWithNode)
		if err != nil {
			err = fmt.Errorf("enable trigger on node %q: %w", node.ID, err)
			return
		}
	}
	return
}

// ApplyFile a workflow file to database.
// It parses and validates the file, and makes the workflow/nodes in database
// consistent with file content.
func ApplyFile(ctx context.Context, triggerRegistry *trigger.Registry, file string, db *model.DB) (workflowID string, err error) {
	workflow, err := UnmarshalWorkflowFromFile(file)
	if err != nil {
		return
	}

	err = db.RunInTx(ctx, func(tx model.Operator) error {
		workflowID, err = Apply(ctx, triggerRegistry, &workflow, tx)
		return err
	})
	if err != nil {
		err = fmt.Errorf("transaction aborted: %w", err)
	}
	return
}

// UnmarshalWorkflowFromFile get workflow by read file.
func UnmarshalWorkflowFromFile(file string) (workflow model.WorkflowWithNodes, err error) {
	if err = yaml.UnmarshalWithFile(file, &workflow); err != nil {
		err = fmt.Errorf("unmarshal workflow file error: %w", err)
		return
	}
	return
}
