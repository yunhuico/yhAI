package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiclient"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/console"
)

// ListApp a tui application for list credential.
type ListApp struct {
	ctx           context.Context
	perPage       int
	curPage       int
	totalPage     int
	initialLoad   bool
	apiClient     *apiclient.Client
	app           *tview.Application
	list          *tview.List
	detail        *tview.TextView
	main          *tview.Flex
	credentials   []model.Credential
	curCredential model.Credential
}

// Run run list applcation, never return unless error.
func (a *ListApp) Run() {
	a.init()
	if err := a.app.SetRoot(a.main, true).Run(); err != nil {
		console.RenderError(err.Error()).Println()
	}
}

// Stop the application manually.
func (a *ListApp) Stop() {
	a.app.Stop()
}

func (a *ListApp) init() {
	a.loadCredentials(a.curPage)

	a.list.SetChangedFunc(func(i int, _, _ string, _ rune) {
		if len(a.credentials) == 0 {
			return
		}
		a.curCredential = a.credentials[i]
		a.updateDetail()
	})

	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModMask(0))
		case 'q':
			a.app.Stop()
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModMask(0))
		case 'h':
			a.pressLeft()
		case 'l':
			a.pressRight()
		}

		switch event.Key() {
		case 260: // <-
			a.pressLeft()
		case 259: // ->
			a.pressRight()
		}
		return event
	})
}

func (a *ListApp) pressLeft() {
	if a.curPage == 1 {
		return
	}
	a.curPage -= 1
	a.loadCredentials(a.curPage)
}

func (a *ListApp) pressRight() {
	if a.curPage == a.totalPage {
		return
	}
	a.curPage += 1
	a.loadCredentials(a.curPage)
}

func (a *ListApp) updateDetail() {
	b, _ := json.MarshalIndent(a.curCredential, "", "  ")
	a.detail.SetText(string(b))
}

func NewListApp(ctx context.Context, apiClient *apiclient.Client, perPage int) *ListApp {
	app := &ListApp{
		ctx:         ctx,
		perPage:     perPage,
		curPage:     1,
		apiClient:   apiClient,
		app:         tview.NewApplication(),
		initialLoad: true,
	}

	main := tview.NewFlex()
	main.SetTitle(" Credentials loading... ")

	list := tview.NewList().ShowSecondaryText(true)
	list.SetSecondaryTextColor(tcell.Color102)
	list.SetBorder(true)
	list.SetHighlightFullLine(true)
	list.SetSelectedBackgroundColor(tcell.ColorLightBlue)
	list.SetSelectedFocusOnly(true)
	list.SetBorderPadding(0, 0, 1, 1)
	app.list = list

	detail := tview.NewTextView()
	detail.SetBorder(true)
	detail.SetTitle(" Detail ")
	detail.SetScrollable(true)
	detail.SetBorderPadding(0, 0, 1, 1)
	app.detail = detail

	main.AddItem(list, 0, 3, true).AddItem(detail, 0, 4, true)
	app.main = main

	return app
}

func (a *ListApp) loadCredentials(page int) {
	a.list.Clear()

	credentialsList, err := a.apiClient.ListCredentials(a.ctx, (page-1)*a.perPage, a.perPage)
	if err != nil {
		a.app.Stop()
		console.RenderError(err.Error()).Println()
		return
	}

	a.totalPage = int(math.Ceil(float64(credentialsList.Total) / float64(a.perPage)))
	a.list.SetTitle(fmt.Sprintf(" %d credentials <%d/%d> ", credentialsList.Total, page, a.totalPage))
	for _, credential := range credentialsList.Credentials {
		title := fmt.Sprintf(`%s %s`, strings.ToUpper(string(credential.Type)), credential.Name)
		secondaryText := fmt.Sprintf("Adapter: %s, CreatedAt: %s", credential.AdapterClass, credential.CreatedAt.Format(time.Stamp))
		a.list.AddItem(title, secondaryText, 0, nil)
	}
	a.credentials = credentialsList.Credentials

	if a.initialLoad {
		if len(a.credentials) == 0 {
			console.RenderWarning("no credentials").Println()
			a.app.Stop()
			return
		}

		a.initialLoad = false
		a.curCredential = a.credentials[0]
		a.updateDetail()
	}
}
