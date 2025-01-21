package workflow

import (
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"
)

// Check workflow file is valid or not.
func Check(file string) error {
	userWorkflow, err := workflow.UnmarshalWorkflowFromFile(file)
	if err != nil {
		return err
	}

	report := validate.ValidateWorkflow(userWorkflow)
	if !report.ExistsReport() {
		return nil
	}
	return report
}
