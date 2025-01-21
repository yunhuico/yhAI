package workflow

import (
	"context"
	"fmt"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/console"
)

const (
	limitPerPage = 15
)

type excitable struct {
	exit bool
	msg  string
}

type workflowListModel struct {
	excitable

	ctx           context.Context
	limit         int
	count         int
	index         int
	workflows     []model.Workflow
	paginator     paginator.Model
	access        workflowAccess
	mode          string
	width, height int
}

const (
	listMode = "list"
	logMode  = "log"
)

var (
	workflowItemStyle = lipgloss.NewStyle().MarginTop(2)
	outputStyle       = lipgloss.NewStyle().
				MarginTop(1).
				MarginBottom(1).
				MarginLeft(2).
				MarginRight(2)

	tipStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("228"))

	activeLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#25A065"))

	inactiveLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FEC47F"))

	highlightTitleStyle = lipgloss.NewStyle().
				BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
				Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"})

	normalTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))
)

var (
	greenDot  = activeLabelStyle.Render("●")
	yellowDot = inactiveLabelStyle.Render("●")
	redDot    = errorStyle.Render("●")
)

func (m *workflowListModel) getCurrentPageWorkflows(offset int) ([]model.Workflow, error) {
	workflows, _, err := m.access.ListWorkflows(m.ctx, m.limit, offset)
	return workflows, err
}

func (m *workflowListModel) Init() tea.Cmd {
	return nil
}

type fatalMsg struct {
	msg string
}

func fatal(msg string) tea.Cmd {
	return func() tea.Msg {
		return &fatalMsg{msg: msg}
	}
}

func (m *workflowListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case *fatalMsg:
		m.exit = true
		m.msg = msg.msg
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.mode == logMode {
				logListModel, err := newWorkflowLogListModelFromWorkflow(m.ctx, &m.workflows[m.index], m.access)
				logListModel.width = m.width
				logListModel.height = m.height
				if err != nil {
					return m, fatal(err.Error())
				}
				logListModel.setParent(m)
				return logListModel, nil
			}
		case "j":
			m.moveDown()
		case "k":
			m.moveUp()
		}
	}
	m.paginator, cmd = m.paginator.Update(msg)
	return m, cmd
}

func (m *workflowListModel) moveUp() {
	if m.index == 0 {
		m.index = len(m.workflows) - 1
		return
	}
	m.index--
}

func (m *workflowListModel) moveDown() {
	if m.index == len(m.workflows)-1 {
		m.index = 0
		return
	}
	m.index++
}

func (m *workflowListModel) View() string {
	if m.exit {
		return console.RenderError(m.msg).String()
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Total %d workflows", m.count)))
	start, _ := m.paginator.GetSliceBounds(m.limit)
	// load from database
	workflows, err := m.getCurrentPageWorkflows(start)
	if err != nil {
		return console.RenderError("get current page workflows error: " + err.Error()).String()
	}

	m.workflows = workflows

	var b1 strings.Builder
	for i, workflow := range workflows {
		b1.Reset()

		if workflow.Status == model.WorkflowStatusEnabled {
			b1.WriteString(greenDot)
		} else {
			b1.WriteString(yellowDot)
		}

		b1.WriteString(" ")
		if i == m.index {
			b1.WriteString(highlightTitleStyle.Render(workflow.Name))
		} else {
			b1.WriteString(workflow.Name)
		}
		if workflow.Description != "" {
			b1.WriteString(" ")
			b1.WriteString(console.RenderGreyColor(workflow.Description).String())
		}
		b1.WriteString("\n")
		b1.WriteString("id: ")
		b1.WriteString(console.RenderGreyColor(workflow.ID).String())
		b.WriteString(workflowItemStyle.Render(b1.String()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.paginator.View())
	b.WriteString(tipStyle.Render("\n\nh:← m:→ j:↓ k:↑ • q: quit\n"))
	return outputStyle.Render(b.String())
}

type workflowAccess interface {
	CountWorkflow(ctx context.Context) (count int, err error)
	ListWorkflows(ctx context.Context, limit int, offset int) (workflows []model.Workflow, count int, err error)
	CountWorkflowInstanceByWorkflowID(ctx context.Context, workflowID string) (count int, err error)
	GetWorkflowInstancesByWorkflowID(ctx context.Context, workflowID string) (instances []model.WorkflowInstance, err error)
	GetNodesByWorkflowID(ctx context.Context, workflowID string) (nodes model.Nodes, err error)
	GetWorkflowInstanceWithNodesByID(ctx context.Context, id string) (instances model.WorkflowInstanceWithNodes, err error)
}

type opt struct {
	mode    string
	teaOpts []tea.ProgramOption
}

type optFunc func(o *opt)

func withMode(mode string) optFunc {
	return func(o *opt) {
		o.mode = mode
	}
}

func withTeaOpt(teaOpt tea.ProgramOption) optFunc {
	return func(o *opt) {
		o.teaOpts = append(o.teaOpts, teaOpt)
	}
}

// NewListProgram creates a workflow-list program
func NewListProgram(ctx context.Context,
	access workflowAccess,
	opts ...optFunc) (program *tea.Program, err error) {
	count, err := access.CountWorkflow(ctx)
	if err != nil {
		return
	}

	if count == 0 {
		err = fmt.Errorf("no workflow found")
		return
	}

	opt := &opt{}
	for _, optFunc := range opts {
		optFunc(opt)
	}

	pmodel := newPaginatorModel()
	pmodel.SetTotalPages(int(count))

	model := workflowListModel{
		ctx:       ctx,
		limit:     limitPerPage,
		count:     int(count),
		access:    access,
		paginator: pmodel,
		mode:      opt.mode,
	}
	teaOpts := append(opt.teaOpts, tea.WithAltScreen(), tea.WithoutCatchPanics(), tea.WithMouseCellMotion())
	return tea.NewProgram(&model, teaOpts...), nil
}

func newPaginatorModel() paginator.Model {
	pmodel := paginator.New()
	pmodel.Type = paginator.Dots
	pmodel.PerPage = limitPerPage
	pmodel.UsePgUpPgDownKeys = false
	pmodel.UseLeftRightKeys = false
	pmodel.UseUpDownKeys = false
	pmodel.UseHLKeys = true
	pmodel.UseJKKeys = false
	pmodel.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	pmodel.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	return pmodel
}

// List all workflow in database
func List(ctx context.Context, access workflowAccess) (program *tea.Program) {
	p, err := NewListProgram(ctx, access, withMode(listMode))
	if err != nil {
		console.RenderError(err.Error()).Println()
		return
	}
	if err := p.Start(); err != nil {
		console.RenderError(err.Error()).Println()
		return program
	}
	return p
}
