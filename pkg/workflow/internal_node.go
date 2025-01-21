package workflow

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/permission"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/validate"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/compare"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
)

//go:embed internal_adapter/switch.json
var switchSpec string

//go:embed internal_adapter/loopFromList.json
var loopFromListSpec []byte

//go:embed foreachAdapter.json
var foreachAdapter []byte

//go:embed internal_adapter/printTarget.json
var debugSpec string

//go:embed internal_adapter/triggerEcho.json
var triggerEchoSpec string

//go:embed internal_adapter/confirm.json
var confirmSpec string

func init() {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	confirmBodyTemplate, err = template.New("").Parse(confirmBodyTemplateText)
	if err != nil {
		err = fmt.Errorf("parsing confirm body template: %w", err)
		return
	}
}

func registerInternalAdapters() {
	registerLogic()
	registerForeach()
	registerConfirm()
	registerDebug()
}

//go:embed debugAdapter.json
var debugAdapterJSON []byte

func registerDebug() {
	meta := adapter.RegisterAdapterByRaw(debugAdapterJSON)
	meta.RegisterSpecByRaw([]byte(debugSpec))
	meta.RegisterSpecByRaw([]byte(triggerEchoSpec))
	RegistryNodeMeta(&debugTarget{})
	RegistryNodeMeta(&triggerEcho{})
}

//go:embed confirmAdapter.json
var confirmAdapterJSON []byte

func registerConfirm() {
	meta := adapter.RegisterAdapterByRaw(confirmAdapterJSON)
	meta.RegisterSpecByRaw([]byte(confirmSpec))
	RegistryNodeMeta(&confirmAdapter{})
}

//go:embed logicAdapter.json
var logicAdapterJSON []byte

func registerLogic() {
	meta := adapter.RegisterAdapterByRaw(logicAdapterJSON)
	meta.RegisterSpecByRaw([]byte(switchSpec))
	RegistryNodeMeta(&switchLogic{})
}

func registerForeach() {
	meta := adapter.RegisterAdapterByRaw(foreachAdapter)
	meta.RegisterSpecByRaw(loopFromListSpec)
	RegistryNodeMeta(&loopFromList{})
}

type switchLogic validate.SwitchLogicNode

func (s *switchLogic) UltrafoxNode() NodeMeta {
	adapter.MustLookupSpec(validate.SwitchClass)
	return NodeMeta{
		Class: validate.SwitchClass,
		New: func() Node {
			return new(switchLogic)
		},
		InputForm: adapter.AnySchema,
	}
}

type switchNodeResult []switchNodeResultItem
type switchNodeResultItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ExecutionResult bool   `json:"executionResult"`
}

func (s *switchLogic) Run(c *NodeContext) (any, error) {
	c.requestLogicControl()
	var result switchNodeResult

	var (
		matched      = false
		matchedIndex = len(s.Paths) - 1
	)
	for i, path := range s.Paths {
		// default branch does not need validate the conditions.
		if path.IsDefault && i == len(s.Paths)-1 {
			c.setNextNode(path.Transition)
			matched = true
			break // the loop over in this line.
		}

		// normal branches
		pass, err := checkPathPass(c, path.Conditions)
		if err != nil {
			err = fmt.Errorf("validating on branch #%d: %w", i+1, err)
			return result, err
		}
		if !pass {
			continue
		}
		c.setNextNode(path.Transition)
		matched = true
		matchedIndex = i
		break
	}

	if !matched {
		// no branch matches, we should end workflow execution
		c.setNextNode("")
	}

	for i, path := range s.Paths {
		pathResult := switchNodeResultItem{
			ID:   strconv.Itoa(i + 1),
			Name: path.Name,
		}
		if i == matchedIndex {
			pathResult.ExecutionResult = true
		}
		if path.IsDefault && i == len(s.Paths)-1 {
			pathResult.ID = "default"
		}
		result = append(result, pathResult)
	}

	return result, nil
}

func checkPathPass(c *NodeContext, groups []validate.ConditionGroup) (bool, error) {
	if len(groups) == 0 {
		return false, fmt.Errorf("condition groups empty")
	}

	comparator := compare.NewComparator()

	for _, group := range groups {
		groupPass := true

		for _, condition := range group {
			var (
				err   error
				left  any
				right any
			)

			left, err = c.dynamicCalc(condition.Left)
			if err != nil {
				c.Error("dynamic Calc condition left error", log.String("left", condition.Left), log.ErrField(err))
				groupPass = false
				break
			}
			right, err = c.dynamicCalc(condition.Right)
			if err != nil {
				c.Error("dynamic Calc condition right error", log.String("right", condition.Right), log.ErrField(err))
				groupPass = false
				break
			}

			operation, err := toCompareOperation(condition.Operation)
			if err != nil {
				c.Error("converting to compare operation", log.ErrField(err))
				groupPass = false
				break
			}
			pass, err := comparator.Compare(operation, left, right)
			logFields := []log.Field{log.String("operation", string(condition.Operation)), log.Any("left", left), log.Any("right", right)}
			if err != nil {
				c.Error("compare left and right error", append(logFields, log.ErrField(err))...)
				groupPass = false
				break
			}

			c.Debug("compare left and right", append(logFields, log.Bool("pass", pass))...)
			if !pass {
				groupPass = false
				break
			}
		}

		if groupPass {
			return true, nil
		}
	}

	return false, nil
}

func toCompareOperation(operation validate.Operation) (compare.Operation, error) {
	switch operation {
	case validate.EqualsOperation:
		return compare.EqualsOperation, nil
	case validate.NotEqualsOperation:
		return compare.NotEqualsOperation, nil
	case validate.ContainsOperation:
		return compare.ContainsOperation, nil
	case validate.ContainsLowercasedOperation:
		return compare.ContainsLowerCasedOperation, nil
	case validate.NotContainsOperation:
		return compare.NotContainsOperation, nil
	case validate.NotContainsLowerCasedOperation:
		return compare.NotContainsLowerCasedOperation, nil
	case validate.StringStartWithOperation:
		return compare.StringStartWithOperation, nil
	case validate.StringEndWithOperation:
		return compare.StringEndWithOperation, nil
	case validate.TimeBeforeOperation:
		return compare.TimeBeforeOperation, nil
	case validate.TimeAfterOperation:
		return compare.TimeAfterOperation, nil
	case validate.TimeDayAgoOperation:
		return compare.TimeDayAgoOperation, nil
	case validate.TimeHourAgoOperation:
		return compare.TimeHourAgoOperation, nil
	case validate.TimeWeekAgoOperation:
		return compare.TimeWeekAgoOperation, nil
	case validate.EmptyOperation:
		return compare.EmptyOperation, nil
	case validate.NotEmptyOperation:
		return compare.NotEmptyOperation, nil
	case validate.LessThan:
		return compare.LessThan, nil
	case validate.GreaterThan:
		return compare.GreaterThan, nil
	default:
		return 0, fmt.Errorf("unexpected operation %q", operation)
	}
}

type loopFromList validate.LoopFromListNode

const (
	iterLengthKey = "loopTotalIterations"
	iterIndexKey  = "loopIteration"
	iterItemKey   = "loopItem"
	iterIsLastKey = "loopIterationIsLast"
)

func (s *loopFromList) UltrafoxNode() NodeMeta {
	return NodeMeta{
		Class: validate.ForeachClass,
		New: func() Node {
			return new(loopFromList)
		},
		InputForm: adapter.AnySchema,
	}
}

type ForeachOutput struct {
	LoopIteration       int   `json:"loopIteration"`
	LoopTotalIterations int   `json:"loopTotalIterations"`
	LoopIterationIsLast bool  `json:"loopIterationIsLast"`
	LoopItem            any   `json:"loopItem,omitempty"`
	Results             []any `json:"results,omitempty"`
}

// Run loopFromList only support one level of iteration
func (s *loopFromList) Run(c *NodeContext) (output any, err error) {
	if s.InputCollection == "" {
		err = fmt.Errorf("ultrafox/foreach inputCollection property cannot empty")
		return
	}

	if !c.workflowContext.isTestMode() {
		if s.Transition == "" {
			err = fmt.Errorf("ultrafox/foreach transition property cannot empty")
			return
		}
	}

	inputCollection, _ := trimBraceBrackets(s.InputCollection)
	collection, err := c.workflowContext.getIteration(inputCollection)
	if err != nil {
		err = fmt.Errorf("getting iteration list data: %w", err)
		return
	}

	foreachOutput := &ForeachOutput{
		LoopIteration:       0,
		LoopTotalIterations: 0,
		LoopIterationIsLast: false,
	}
	defer func() {
		if err != nil {
			return
		}
		output = foreachOutput
	}()

	collectionLength := len(collection)
	if collectionLength == 0 {
		return
	}

	if c.workflowContext.isTestMode() {
		foreachOutput.LoopIteration = 1
		foreachOutput.LoopTotalIterations = collectionLength
		foreachOutput.LoopIterationIsLast = collectionLength == 1
		foreachOutput.LoopItem = collection[0]
		return
	}

	defer c.clearIter()
	c.workflowContext.setIterKeyValue(iterLengthKey, collectionLength)
	results := []any{}
	for i, item := range collection {
		if err = c.contextAborted(); err != nil {
			err = fmt.Errorf("context aborted in foreach: %w", err)
			return
		}

		c.workflowContext.setCurrentIteration(i+1, item, i+1 == collectionLength)

		var itemResult any
		itemResult, err = c.workflowContext.runSubflow(s.Transition)
		if err != nil {
			err = fmt.Errorf("running foreach %d loop failed: %w", i, err)
			return
		}

		results = append(results, itemResult)
	}

	foreachOutput.LoopIterationIsLast = true
	foreachOutput.LoopIteration = collectionLength
	foreachOutput.LoopTotalIterations = collectionLength
	foreachOutput.Results = results
	return
}

// ErrNeedWorkflowPaused is used to signal the worker to stop the workflow execution
// and set its status to paused.
var ErrNeedWorkflowPaused = errors.New("workflow needs to be paused")

// confirmAdapter pauses current workflow execution and send emails to confirmers
// for proceeding approval.
type confirmAdapter struct {
	// Description to use in the email
	Description string `json:"description"`
	// IDs of users to do the authorization
	Confirmers []int `json:"confirmers"`
	// Expiring Time in hours
	Timeout int `json:"timeout"`
}

var _ Node = (*confirmAdapter)(nil)

func (*confirmAdapter) UltrafoxNode() NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/confirm#email")

	return NodeMeta{
		Class: spec.Class,
		New: func() Node {
			return &confirmAdapter{}
		},
		InputForm: spec.InputSchema,
	}
}

type ConfirmDecision string

const (
	ConfirmDecisionApproved ConfirmDecision = "approve"
	ConfirmDecisionDeclined ConfirmDecision = "decline"
)

type ConfirmAdapterOutput struct {
	// ID of the corresponding confirm record
	ConfirmID string `json:"confirmId"`
	// URL to the confirm page
	URL string `json:"url"`

	// The following fields are updated after user decision.

	Decision ConfirmDecision `json:"decision,omitempty"`
	// user ID of the decider
	ConfirmerUserID int `json:"confirmerUserId,omitempty"`
	// username of the decider
	ConfirmerUsername string `json:"confirmerUsername,omitempty"`
	// email of the decider
	ConfirmerEmail string `json:"confirmerEmail,omitempty"`
	// Unix nano
	ConfirmedAt string `json:"confirmedAt,omitempty"`
}

//go:embed confirm-body.html
var confirmBodyTemplateText string

var confirmBodyTemplate *template.Template

type confirmTemplatePayload struct {
	URL           string
	Description   string
	ExpiringHours int
}

func (f *confirmAdapter) Run(c *NodeContext) (result any, err error) {
	if c.workflowContext.isTestMode() {
		return f.emailAndPause(c)
	}

	var ctx = c.Context()

	confirm, err := c.workflowContext.db.GetConfirmByNodeInstance(ctx, c.workflowContext.workflowInstanceID, c.node.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return f.emailAndPause(c)
	}
	if err != nil {
		err = fmt.Errorf("querying for confirm history: %w", err)
		return
	}

	var decision ConfirmDecision
	switch confirm.Status {
	case model.ConfirmStatusApproved:
		decision = ConfirmDecisionApproved
	case model.ConfirmStatusDeclined:
		decision = ConfirmDecisionDeclined
	default:
		err = fmt.Errorf("unexpected confirm status %q", confirm.Status)
		return
	}

	confirmer, err := c.workflowContext.db.GetUserByID(ctx, confirm.ConfirmerUserID)
	if err != nil {
		err = fmt.Errorf("querying for confirmer: %w", err)
		return
	}

	// we are revisiting this confirm node after resuming
	result = ConfirmAdapterOutput{
		ConfirmID:         confirm.ID,
		URL:               f.confirmPageURL(c, confirm.ID),
		Decision:          decision,
		ConfirmerUserID:   confirmer.ID,
		ConfirmerUsername: confirmer.Name,
		ConfirmerEmail:    confirmer.Email,
		ConfirmedAt:       confirm.UpdatedAt.Format(time.RFC3339),
	}

	// there will be whole copies of node instances recorded again
	err = c.workflowContext.db.DeleteWorkflowInstanceNodeByWorkflowInstanceID(ctx, c.workflowContext.workflowInstanceID)
	if err != nil {
		err = fmt.Errorf("removing outdated confirm workflow instance node: %w", err)
		return
	}

	if confirm.Status == model.ConfirmStatusDeclined {
		// stop the execution
		err = fmt.Errorf("confirm is declined by user %q", confirmer.Name)
		return
	}

	return
}

func (f *confirmAdapter) confirmPageURL(c *NodeContext, confirmID string) string {
	return c.serverHost.APIFullURL("/confirm/workflow/" + confirmID)
}

func (f *confirmAdapter) emailAndPause(c *NodeContext) (result any, err error) {
	var ctx = c.Context()

	// sanity check
	if f.Description == "" {
		err = errors.New("confirmAdapter: description field is required")
		return
	}
	if f.Timeout <= 0 {
		err = errors.New("confirmAdapter: timeout must be positive")
		return
	}
	if len(f.Confirmers) == 0 {
		err = errors.New("confirmAdapter: there must be at least one confirmer")
		return
	}

	const maxConfirmers = 10
	if len(f.Confirmers) > maxConfirmers {
		err = fmt.Errorf("confirmAdapter: too many confirmers, exceding limit %d", maxConfirmers)
		return
	}

	// ensure all confirmers has access to the workflow
	enforcer := permission.NewEnforcer(c.workflowContext.db)
	err = enforcer.UsersAllHavePermissions(ctx, f.Confirmers, c.workflowContext.workflow.OwnerRef, permission.WorkflowExecutionAuthorization)
	if err != nil {
		err = fmt.Errorf("checking confirmer permssions: %w", err)
		return
	}

	expiredAt := time.Now().Add(time.Duration(f.Timeout) * time.Hour)
	newConfirm := model.Confirm{
		WorkflowID:          c.workflowContext.workflow.ID,
		NodeID:              c.node.ID,
		WorkflowInstanceID:  c.workflowContext.workflowInstanceID,
		Status:              model.ConfirmStatusWaiting,
		Confirmers:          f.Confirmers,
		ExpiredAt:           expiredAt,
		RenderedDescription: f.Description,
	}
	if c.workflowContext.isTestMode() {
		// let's pretend we have inserted the confirm record
		newConfirm.ID = "sample"
	} else {
		err = c.workflowContext.db.InsertConfirm(ctx, &newConfirm)
		if err != nil {
			err = fmt.Errorf("inserting confirm: %w", err)
			return
		}
	}
	// remove the dangling confirm record if there's an error following
	defer func() {
		if c.workflowContext.isTestMode() || errors.Is(err, ErrNeedWorkflowPaused) {
			// relax
			return
		}

		cleanUpCtx, cancelCleanup := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.workflowContext.db.DeleteConfirmByID(cleanUpCtx, newConfirm.ID)
		cancelCleanup()
	}()

	var (
		body               bytes.Buffer
		pageURL            = f.confirmPageURL(c, newConfirm.ID)
		formattedExpiredAt = expiredAt.Format(time.RFC3339)
	)
	err = confirmBodyTemplate.Execute(&body, confirmTemplatePayload{
		URL:           pageURL,
		Description:   f.Description,
		ExpiringHours: f.Timeout,
	})
	if err != nil {
		err = fmt.Errorf("rendering email body: %w", err)
		return
	}

	receivers, err := c.workflowContext.db.GetEmailsByUserID(ctx, f.Confirmers...)
	if err != nil {
		err = fmt.Errorf("querying email of confirmers: %w", err)
		return
	}

	// send the email
	notificationMail := smtp.Mail{
		To:       receivers,
		Subject:  fmt.Sprintf("您的极狐链工作流【%s】需要审批", c.workflowContext.workflow.Name),
		Body:     body.Bytes(),
		HTMLBody: true,
	}
	err = c.workflowContext.mailSender.SendMail(ctx, notificationMail)
	if err != nil {
		err = fmt.Errorf("sending notification mail: %w", err)
		return
	}

	if c.workflowContext.isTestMode() {
		result = ConfirmAdapterOutput{
			ConfirmID:         newConfirm.ID,
			URL:               pageURL,
			Decision:          ConfirmDecisionApproved,
			ConfirmerUserID:   0,
			ConfirmerUsername: "Sample User",
			ConfirmerEmail:    "sample@example.com",
			ConfirmedAt:       formattedExpiredAt,
		}
	} else {
		result = ConfirmAdapterOutput{
			ConfirmID:   newConfirm.ID,
			URL:         pageURL,
			ConfirmedAt: formattedExpiredAt,
		}
	}

	// throw the signal error when not during testing
	if !c.workflowContext.isTestMode() {
		err = ErrNeedWorkflowPaused
	}

	return
}

type debugTarget struct {
	Target string `json:"target"`
}

func (s *debugTarget) UltrafoxNode() NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/debug#printTarget")
	return NodeMeta{
		Class: spec.Class,
		New: func() Node {
			return new(debugTarget)
		},
		InputForm: adapter.AnySchema,
	}
}

func (s *debugTarget) Run(c *NodeContext) (any, error) {
	v, err := c.dynamicCalc(s.Target)
	if err != nil {
		return fmt.Sprintf("UltraFox: failed to resolve the variable %s: %s", s.Target, err), nil
	}
	return v, nil
}

type triggerEcho map[string]any

func (s *triggerEcho) GetConfigObject() any {
	return &triggerEcho{}
}

func (s *triggerEcho) Create(c trigger.WebhookContext) (map[string]any, error) {
	return nil, nil
}

func (s *triggerEcho) Delete(c trigger.WebhookContext) error {
	return nil
}

var _ trigger.TriggerProvider = (*triggerEcho)(nil)

func (s *triggerEcho) UltrafoxNode() NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/debug#triggerEcho")
	return NodeMeta{
		Class: spec.Class,
		New: func() Node {
			return new(triggerEcho)
		},
		InputForm: adapter.AnySchema,
	}
}

func (s *triggerEcho) Run(c *NodeContext) (any, error) {
	return s, nil
}
