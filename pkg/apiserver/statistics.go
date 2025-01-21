package apiserver

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/gin-gonic/gin"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

//go:embed markdown.html
var markdownHTML string

var markdownTpl *template.Template

func init() {
	markdownTpl, _ = template.New("").Parse(markdownHTML)
}

const statisticTemplate = `## Workflow 统计

### 总计

数量: {{ .Workflow.Total }}

启用/停用: {{ .Workflow.EnableCount }}/{{ .Workflow.DisableCount }}

执行次数: {{ .Workflow.ExecutionCount }} (备注：24w 条线上的错误数据，没有删除)

  - 成功: {{ .Workflow.SuccessExecutionCount }}
  - 失败: {{ .Workflow.FailExecutionCount }}

### 单个 Workflow 运行情况

| workflow name | 所属 org  | 运行次数 | 成功运行次数 |
| ------ | ------ | ------ | ------ |
{{ range .Workflow.RunStatistics }}|  {{ .WorkflowName }} |  {{ .OrgName }} |  {{ .Total }} | {{ .SuccessCount }} |
{{ end }}

## Org 统计

Org 数量: {{ .Org.Total }}

### Workflow 使用情况

| 所属 org | workflow 数量 | active workflow 数量 |
| ------ | :------: | :------: |
{{ range .Workflow.OrgStatistics }}|  {{ .OwnerName }} |  {{ .Total }} |  {{ .ActiveCount }} |
{{ end }}

## Adapter 统计

- Actor 数量: {{ .Adapter.ActorCount }}
- Trigger 数量: {{ .Adapter.TriggerCount }}

### Adapter 使用排行
{{ range .Adapter.UsageLeaderboard }}
- {{ .Count }} {{ .Name }}
{{ end }}
`

// Statistics
// @Summary statistic the workflow stats.
// @Produce json
// @Success 200 {object} apiserver.R{data=response.StatisticsResp}
// @Failure 200 {object} apiserver.R
// @Router /api/v1/statistics [post]
func (h *APIHandler) Statistics(c *gin.Context) {
	var (
		err  error
		ctx  = c.Request.Context()
		resp = response.StatisticsResp{}
		key  = c.Query("key")
	)
	defer func() {
		if err != nil {
			_ = c.Error(err)
			c.Abort()
		}
	}()

	if key != "ultrafox-is-awesome" {
		c.Abort()
		return
	}

	resp.Workflow.Total, err = h.db.CountWorkflow(ctx)
	if err != nil {
		return
	}

	enableCount, err := h.db.CountWorkflowByStatus(ctx, model.WorkflowStatusEnabled)
	if err != nil {
		return
	}
	resp.Workflow.EnableCount = enableCount
	resp.Workflow.DisableCount = resp.Workflow.Total - enableCount
	resp.Workflow.ExecutionCount, err = h.db.CountWorkflowInstance(ctx)
	if err != nil {
		return
	}
	resp.Workflow.SuccessExecutionCount, err = h.db.CountWorkflowInstanceByStatus(ctx, model.WorkflowInstanceStatusCompleted)
	if err != nil {
		return
	}
	resp.Workflow.FailExecutionCount, err = h.db.CountWorkflowInstanceByStatus(ctx, model.WorkflowInstanceStatusFailed)
	if err != nil {
		return
	}
	result, err := h.db.CountWorkflowInstanceGroupByHour(ctx)
	if err != nil {
		return
	}
	resp.Workflow.DayExecutionCount = result.To24Hour()
	adapterManager := adapter.GetAdapterManager()
	resp.Adapter.ActorCount = adapterManager.GetSpecCountByType(adapter.SpecActorType)
	resp.Adapter.TriggerCount = adapterManager.GetSpecCountByType(adapter.SpecTriggerType)
	resp.Adapter.UsageLeaderboard, err = h.db.CountNodeGroupByAdapter(ctx)
	if err != nil {
		return
	}
	resp.Org.Total, err = h.db.CountOrganization(ctx)
	if err != nil {
		return
	}

	workflowOrgStatistics, err := h.db.CountWorkflowGroupByOrg(ctx)
	if err != nil {
		return
	}
	resp.Workflow.OrgStatistics = workflowOrgStatistics

	workflowRunStatistics, err := h.db.CountWorkflowInstancesGroupByWorkflow(ctx)
	if err != nil {
		return
	}
	resp.Workflow.RunStatistics = workflowRunStatistics

	if c.Query("html") != "" {
		opts := html.RendererOptions{
			Flags: html.FlagsNone,
		}
		renderer := html.NewRenderer(opts)

		mkTpl, err := template.New("").Funcs(sprig.HtmlFuncMap()).Parse(statisticTemplate)
		if err != nil {
			return
		}

		buf := bytes.NewBuffer(nil)
		err = mkTpl.Execute(buf, resp)
		if err != nil {
			panic(err)
		}
		mkContent := markdown.ToHTML(buf.Bytes(), nil, renderer)

		_ = markdownTpl.Execute(c.Writer, map[string]any{
			"content": string(mkContent),
		})
		c.Writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	} else {
		OK(c, resp)
	}
}
