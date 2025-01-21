package workflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
)

type IDeleteCommand interface {
	// Delete all workflow related resource.
	Delete(ctx context.Context, workflowID string) error
	// DeleteTriggerResource delete the trigger external resource.
	DeleteTriggerResource(ctx context.Context, workflowID string) error
	// DeleteMetaData delete everything related to the workflow in UltraFox (doesn't contain the external resource).
	DeleteMetaData(ctx context.Context, workflowID string) error
}

type deleteCommand struct {
	db              model.Operator
	triggerRegistry *trigger.Registry
}

func NewDeleteCommand(db model.Operator, triggerRegistry *trigger.Registry) IDeleteCommand {
	return &deleteCommand{
		db:              db,
		triggerRegistry: triggerRegistry,
	}
}

func (d deleteCommand) Delete(ctx context.Context, workflowID string) error {
	var err error

	if err = d.DeleteTriggerResource(ctx, workflowID); err != nil {
		return fmt.Errorf("delete trigger resource %s: %w", workflowID, err)
	}

	if err = d.DeleteMetaData(ctx, workflowID); err != nil {
		return fmt.Errorf("delete workflow meta data %s: %w", workflowID, err)
	}

	return nil
}

func (d deleteCommand) DeleteTriggerResource(ctx context.Context, workflowID string) error {
	userTrigger, err := d.db.GetTriggerByWorkflowID(ctx, workflowID)
	if errors.Is(err, sql.ErrNoRows) {
		// relax
		return nil
	}
	if err != nil {
		err = fmt.Errorf("querying trigger: %w", err)
		return err
	}

	triggerNode, err := d.db.GetTriggerNodeByWorkflowID(ctx, workflowID)
	if err != nil {
		err = fmt.Errorf("querying trigger nodes: %w", err)
		return err
	}

	err = d.triggerRegistry.DisableTrigger(ctx, d.db, model.TriggerWithNode{
		Trigger: userTrigger,
		Node:    &triggerNode,
	})
	if err != nil {
		err = fmt.Errorf("deleting trigger: %w", err)
		return err
	}

	return nil
}

func (d deleteCommand) DeleteMetaData(ctx context.Context, workflowID string) (err error) {
	if err = d.db.DeleteNodesByWorkflowID(ctx, workflowID); err != nil {
		return fmt.Errorf("delete node by workflow id %s: %w", workflowID, err)
	}

	if err = d.db.DeleteWorkflowByID(ctx, workflowID); err != nil {
		return fmt.Errorf("delete workflow %s: %w", workflowID, err)
	}

	if err = d.db.DeleteTriggersByWorkflowID(ctx, workflowID); err != nil {
		return fmt.Errorf("delete trigger by workflow id %v: %w", workflowID, err)
	}

	if err = d.db.DeleteWorkflowInstancesByWorkflowID(ctx, workflowID); err != nil {
		return fmt.Errorf("delete workflow instance by workflow id %v: %w", workflowID, err)
	}

	return
}
