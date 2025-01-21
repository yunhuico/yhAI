package apiserver

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/schema"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

const languageHeaderKey = "X-Language"

// GetAdapterList list all adapters
// @Summary list all adapters
// @Accept json
// @Produce json
// @Success 200 {object} apiserver.R{data=adapter.ListPresentData}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/adapters [get]
func (h *APIHandler) GetAdapterList(c *gin.Context) {
	var (
		headerLang = c.Request.Header.Get(languageHeaderKey)
		queryLang  = c.Query("language")
		lang       = headerLang
	)

	if queryLang != "" {
		lang = queryLang
	}

	rendered := adapter.GetAdapterManager().Present(lang)
	OK(c, rendered)
}

// QueryFieldSelect
// @Accept json
// @Produce json
// @Param   body body payload.QueryFieldSelectReq true "the payload"
// @Success 200 {object} apiserver.R{data=response.QueryFieldSelectResp}
// @Failure 400 {object} apiserver.R
// @Router /api/v1/adapters/fieldSelect [post]
func (h *APIHandler) QueryFieldSelect(c *gin.Context) {
	var (
		ctx      = c.Request.Context()
		err      error
		req      payload.QueryFieldSelectReq
		language = c.GetHeader(languageHeaderKey)
	)

	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	err = c.ShouldBindJSON(&req)
	if err != nil {
		_ = c.Error(err)
		err = errBizInvalidRequestPayload
		return
	}

	err = req.Normalize()
	if err != nil {
		err = fmt.Errorf("normalize request: %w", err)
		return
	}

	adapterManager := adapter.GetAdapterManager()
	spec := adapterManager.LookupSpec(req.Class)
	if spec == nil {
		err = errBizInvalidRequestPayload
		return
	}

	inputFields := req.InputFields
	// the req.InputFields is current editing actor/trigger's form,
	// all value is string, such as projectId, request is {"projectId": "12"} or {"projectId": "{{ .Node.1.output.projectId }}"},
	// so should transform the inputFields to right type.
	// how to transform to the right value if value is an expression?
	// use sample data as context data, calculate the value dynamically.
	if req.WorkflowID != "" { // if workflowID, should allow call this api, because this api is a pure function in design.
		var samplesOutput map[string]any
		samplesOutput, err = h.buildContextDataBySample(ctx, req.WorkflowID)
		if err != nil {
			err = fmt.Errorf("build context data failed by samlpe: %v", err)
			return
		}

		contextData := make(map[string]any, len(samplesOutput))
		for nodeID, output := range samplesOutput {
			contextData[nodeID] = map[string]any{
				"output": output,
			}
		}

		inputFields, err = schema.RenderInputFieldsBySchema(spec.InputSchema, req.InputFields, schema.NewMapReader(map[string]any{"Node": contextData}))
		if err != nil {
			err = fmt.Errorf("transform trigger node input fields :%w", err)
			return
		}
	}
	if inputFields == nil {
		inputFields = make(map[string]any)
	}

	// embed user language preference as an input
	inputFields["X-Language"] = language

	var credential model.CredentialWithParent
	if req.CredentialID != "" {
		credential, err = h.db.GetCredentialWithParentByID(ctx, req.CredentialID)
		if err != nil {
			err = fmt.Errorf("getting credential with parent: %w", err)
			return
		}
	}

	// in the internal logic, UltraFox execute actual node, so that
	// build a model.Node for this manual running.
	action, err := workflow.NewQueryFieldResultAction(workflow.RunNodeInstanceActionOpt{
		BaseWorkflowActionOpt: workflow.BaseWorkflowActionOpt{
			Ctx:                  ctx,
			DB:                   h.db,
			Cache:                h.cache,
			WorkflowWithNodes:    model.WorkflowWithNodesCredential{},
			Cipher:               h.cipher,
			ServerHost:           h.serverHost,
			MailSender:           h.mailSender,
			PassportVendorLookup: h.passportVendorLookup,
		},
		CredentialID: req.CredentialID,
		Spec:         spec,
		InputFields:  inputFields,
		Credential:   &credential,
	})
	if err != nil {
		err = fmt.Errorf("forging QueryFieldResultAction: %w", err)
		return
	}

	result, err := action.Run()
	if err != nil {
		err = fmt.Errorf("action run failed: %w", err)
		return
	}

	OK(c, response.QueryFieldSelectResp{
		Result: result,
	})
}
