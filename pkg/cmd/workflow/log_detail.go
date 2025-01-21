package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/console"
)

// NodePresentItem present the node input, output
type NodePresentItem struct {
	Input    json.RawMessage `json:"input"`
	Output   json.RawMessage `json:"output"`
	Duration int64           `json:"duration"`
}

var nopNodePresentItem = NodePresentItem{
	Input:  json.RawMessage(`"No execution"`),
	Output: json.RawMessage(`"No execution"`),
}

// NodePresentMap join all node data as map, key is node name.
type NodePresentMap map[string]NodePresentItem

// TransDataToNodePresentMap transform the node data to NodePresentMap
func TransDataToNodePresentMap(instanceNodes model.WorkflowInstanceNodes) (NodePresentMap, error) {
	m := NodePresentMap{}
	for _, node := range instanceNodes {
		m[node.NodeID] = NodePresentItem{
			Input:    node.Input,
			Output:   node.Output,
			Duration: node.DurationMs,
		}
	}
	return m, nil
}

func (m NodePresentMap) Get(node string) NodePresentItem {
	if item, ok := m[node]; ok {
		return item
	}
	return nopNodePresentItem
}

type workflowLogDetailModel struct {
	parentModel *workflowLogListModel

	ctx        context.Context
	list       list.Model
	viewport   viewport.Model
	instanceID string
	access     workflowAccess
	workflow   *model.Workflow
	width      int
	height     int
	title      string
	ready      bool
	instance   *model.WorkflowInstanceWithNodes
}

var (
	listViewStyle = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1)

	viewportTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	viewportInfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()

	marginLeftStyle = lipgloss.NewStyle().MarginLeft(2)
)

type workflowLogDetailItem struct {
	node     model.Node
	data     NodePresentItem
	hasError bool
}

func (i workflowLogDetailItem) Title() string       { return i.node.Class }
func (i workflowLogDetailItem) Description() string { return "" }
func (i workflowLogDetailItem) FilterValue() string { return "" }

var _ tea.Model = (*workflowLogDetailModel)(nil)

func newWorkflowLogDetail(ctx context.Context, instanceID string, w *model.Workflow, access workflowAccess) (*workflowLogDetailModel, error) {
	instance, err := access.GetWorkflowInstanceWithNodesByID(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get workflow instance when new w log detail: %w", err)
	}

	nodePresentMap, err := TransDataToNodePresentMap(instance.Nodes)
	if err != nil {
		return nil, fmt.Errorf("workflow instance data invalid: %w", err)
	}

	nodes, err := access.GetNodesByWorkflowID(ctx, w.ID)
	if err != nil {
		err = fmt.Errorf("querying nodes by workflowID: %w", err)
		return nil, err
	}

	items := []list.Item{}
	nodeMap := nodes.MapByID()
	nodeID := instance.StartNodeID
	for nodeID != "" {
		detailItem := workflowLogDetailItem{
			node: nodeMap[nodeID],
			data: nodePresentMap.Get(nodeID),
		}

		if nodeID == instance.FailNodeID {
			detailItem.hasError = true
			items = append(items, detailItem)
			break
		}

		items = append(items, detailItem)
		nodeID = nodeMap[nodeID].Transition
	}

	// a simple list.
	l := list.New(items, detailItemDelegate{len(items)}, 0, 0)
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	l.Paginator.PerPage = len(items)
	l.Paginator.SetTotalPages(len(items))

	m := &workflowLogDetailModel{
		ctx:        ctx,
		title:      fmt.Sprintf("Log <%s/%s>", w.Name, instanceID),
		instanceID: instanceID,
		access:     access,
		workflow:   w,
		list:       l,
		instance:   &instance,
	}
	return m, nil
}

func (m *workflowLogDetailModel) setParent(parent *workflowLogListModel) {
	m.parentModel = parent
}

func (m *workflowLogDetailModel) Init() tea.Cmd {
	return nil
}

func (m *workflowLogDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateListWidth()
		m.updateViewport()
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.parentModel != nil {
				return m.parentModel, nil
			}
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		case "j":
			fallthrough
		case "k":
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case "down":
			fallthrough
		case "up":
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *workflowLogDetailModel) updateViewport() {
	statusBarHeight := lipgloss.Height(m.title)
	height := m.height - statusBarHeight - 8
	detailViewWidth := m.width - m.list.Width()
	m.viewport.Height = height
	m.viewport.Width = detailViewWidth
	m.viewport.SetContent(m.detailView())
}

func (m *workflowLogDetailModel) View() string {
	if !m.ready {
		m.ready = true
		m.updateListWidth()
		m.updateViewport()
	}

	var b strings.Builder
	fmt.Fprintf(&b, titleStyle.Render(m.title)) //nolint
	fmt.Fprintf(&b, "\n")                       //nolint

	view := lipgloss.JoinVertical(
		lipgloss.Top,
		b.String(),
		lipgloss.JoinHorizontal(lipgloss.Left, listViewStyle.Render(m.list.View()), m.detailView()),
	)
	return view
}

func (m *workflowLogDetailModel) detailView() string {
	m.viewport.SetContent(m.buildContextView())
	return marginLeftStyle.Render(fmt.Sprintf("%s\n%s\n%s", m.detailHeaderView(), wordwrap.String(m.viewport.View(), m.viewport.Width), m.detailFooterView()))
}

func (m *workflowLogDetailModel) buildContextView() string {
	selectedItem := m.list.SelectedItem().(workflowLogDetailItem)
	nodeBytes, _ := json.MarshalIndent(selectedItem.node, "", "  ")
	input, _ := utils.FormatJSONIndent(selectedItem.data.Input)
	output, _ := utils.FormatJSONIndent(selectedItem.data.Output)

	if selectedItem.hasError {
		return fmt.Sprintf(`/************** node ***************/
%s

/************** input ***************/
%s

/************** error ***************/
%s`, string(nodeBytes), input, console.RenderError(m.instance.Error).String())
	}

	return fmt.Sprintf(`/************** node ***************/
%s

/************** input ***************/
%s

/************** output ***************/
%s`, string(nodeBytes), input, output)
}

func (m *workflowLogDetailModel) detailFooterView() string {
	info := viewportInfoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (m *workflowLogDetailModel) detailHeaderView() string {
	title := viewportTitleStyle.Render("Log context down/↓ up/↑")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m *workflowLogDetailModel) updateListWidth() {
	listViewWidth := int(0.3 * float64(m.width))
	listWidth := listViewWidth - listViewStyle.GetHorizontalFrameSize()
	m.list.SetSize(listWidth, m.height-10)
}

type detailItemDelegate struct {
	count int
}

func (d detailItemDelegate) Height() int                               { return 1 }
func (d detailItemDelegate) Spacing() int                              { return 0 }
func (d detailItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d detailItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(workflowLogDetailItem)
	if !ok {
		return
	}

	str := i.node.Name

	fn := lipgloss.NewStyle().Render
	if index == m.Index() {
		fn = func(s string) string {
			return highlightTitleStyle.Render(s)
		}
	}

	if !i.hasError {
		fmt.Fprintf(w, greenDot+" ") //nolint
	} else {
		fmt.Fprintf(w, redDot+" ") //nolint
	}

	fmt.Fprintf(w, fn(str))                                                                                                                    //nolint
	fmt.Fprintf(w, "\n [%s] %s", (time.Duration(i.data.Duration) * time.Millisecond).String(), console.RenderGreyColor(i.node.Class).String()) //nolint
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}
