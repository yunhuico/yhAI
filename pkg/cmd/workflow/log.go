package workflow

import (
	"context"
	"fmt"
	"io"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/console"
)

type workflowLogListModel struct {
	parentModel *workflowListModel

	ctx           context.Context
	workflowID    string
	workflow      *model.Workflow
	list          list.Model
	access        workflowAccess
	width, height int
}

var _ tea.Model = (*workflowLogListModel)(nil)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#25A065")).
			Padding(0, 1)
	leftPaddingStyle = lipgloss.NewStyle().
				PaddingLeft(1)
)

type workflowInstanceItem struct {
	id                 string
	age                string
	status             string
	actualRanNodeCount int
	nodeCount          int
	workflow           *model.Workflow
	nodes              model.Nodes
	instance           *model.WorkflowInstance
}

func (i workflowInstanceItem) Title() string       { return i.id }
func (i workflowInstanceItem) Description() string { return i.age }
func (i workflowInstanceItem) FilterValue() string { return i.id }

func newWorkflowLogListModelFromWorkflow(ctx context.Context, workflow *model.Workflow, access workflowAccess) (*workflowLogListModel, error) {
	count, err := access.CountWorkflowInstanceByWorkflowID(ctx, workflow.ID)
	if err != nil {
		return nil, fmt.Errorf("get workflow log count error: %w", err)
	}

	nodes, err := access.GetNodesByWorkflowID(ctx, workflow.ID)
	if err != nil {
		return nil, fmt.Errorf("get workflow node error: %w", err)
	}

	workflowInstances, err := access.GetWorkflowInstancesByWorkflowID(ctx, workflow.ID)
	if err != nil {
		return nil, fmt.Errorf("get workflow instance error: %w", err)
	}

	items := make([]list.Item, len(workflowInstances))
	for i, ins := range workflowInstances {
		actualRanNodeCount := 0 // todo(sword): fix it!

		nodeCount := len(nodes)
		if ins.Status == model.WorkflowInstanceStatusCompleted {
			nodeCount = actualRanNodeCount
		}

		items[i] = workflowInstanceItem{id: ins.ID,
			age:                (time.Duration(ins.DurationMs) * time.Millisecond).String(),
			status:             string(ins.Status),
			nodes:              nodes,
			nodeCount:          nodeCount,
			actualRanNodeCount: actualRanNodeCount,
			workflow:           workflow,
			instance:           &ins,
		}
	}

	delegate := myDelegate{list.NewDefaultDelegate()}
	delegate.Styles.SelectedTitle = highlightTitleStyle
	delegate.Styles.NormalTitle = normalTitleStyle
	delegate.ShowDescription = false

	limit := limitPerPage
	if int(count) < limit {
		limit = int(count)
	}

	list := list.New(items, delegate, 0, 0)
	list.Title = fmt.Sprintf("Total %d workflow Logs", count)
	list.Styles.Title = titleStyle
	list.SetShowHelp(false)
	list.SetShowStatusBar(false)
	list.SetShowPagination(false)
	list.SetShowFilter(false)
	list.Paginator = newPaginatorModel()
	list.Paginator.SetTotalPages(int(count))
	list.Paginator.PerPage = limit

	return &workflowLogListModel{
		ctx:        ctx,
		access:     access,
		workflowID: workflow.ID,
		workflow:   workflow,
		list:       list,
	}, nil
}

func (m *workflowLogListModel) setParent(p *workflowListModel) {
	m.parentModel = p
}

func (*workflowLogListModel) Init() tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{}
	}
}

// Update is called when a message is received. Use it to inspect messages
// and, in response, update the model and/or send a command.
func (m *workflowLogListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.parentModel != nil {
				return m.parentModel, nil
			}
			return m, tea.Quit
		case "enter":
			selected := m.list.SelectedItem().(workflowInstanceItem)
			detailModel, err := newWorkflowLogDetail(m.ctx, selected.id, selected.workflow, m.access)
			detailModel.height = m.height
			detailModel.width = m.width
			if err != nil {
				return m, fatal(err.Error())
			}
			detailModel.setParent(m)
			return detailModel, nil
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the program's UI, which is just a string. The view is
// rendered after every Update.
func (m *workflowLogListModel) View() string {
	return outputStyle.Render(m.list.View() + "\n\n" + leftPaddingStyle.Render(m.list.Paginator.View()))
}

// Log all workflows.
func Log(ctx context.Context, access workflowAccess) (program *tea.Program) {
	p, err := NewListProgram(ctx, access, withMode(logMode))
	if err != nil {
		console.RenderError(err.Error()).Println()
		return
	}
	program = p
	if err := p.Start(); err != nil {
		console.RenderError(fmt.Sprintf("program start error: %s", err)).Println()
		return program
	}
	return p
}

// LogWorkflow one workflow.
func LogWorkflow(ctx context.Context, workflow *model.Workflow, access workflowAccess) (program *tea.Program) {
	model, err := newWorkflowLogListModelFromWorkflow(ctx, workflow, access)
	if err != nil {
		console.RenderError(fmt.Sprintf("new workflow log list model error: %s", err)).Println()
		return nil
	}
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithoutCatchPanics())
	if err := p.Start(); err != nil {
		console.RenderError(fmt.Sprintf("program start error: %s", err)).Println()
		return program
	}
	return p
}

type myDelegate struct {
	list.DefaultDelegate
}

func (d myDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var (
		taskWidth = 6
		ageWidth  = 5
	)

	insItem := item.(workflowInstanceItem)

	if index == 0 {
		fmt.Fprintf(w, "%s%s%s%s\n\n",
			widthStyle(2).Render(""),
			widthStyle(36+1).Render("ID"),
			widthStyle(taskWidth).Render("TASKS"),
			widthStyle(ageWidth).Render("AGE"),
		)
	}

	var status string
	if insItem.status == "completed" {
		status = greenDot
	} else if insItem.status == "failed" {
		status = redDot
	} else {
		status = yellowDot
	}
	fmt.Fprintf(w, "%s ", status)

	d.DefaultDelegate.Render(w, m, index, item)

	fmt.Fprintf(w, " %s", widthStyle(taskWidth).Render(fmt.Sprintf("%d/%d", insItem.actualRanNodeCount, insItem.nodeCount)))
	fmt.Fprintf(w, "%s", insItem.age)
}

func widthStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().Width(width)
}
