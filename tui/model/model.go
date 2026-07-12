package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
)

type AppState int

const (
	StateMonths AppState = iota
	StateList
	StateDetail
	StateEdit
	StateRenderSelect
	StateEntrySelect
	StateRenderHistory
)

type RenderDoneMsg struct {
	Output string
	Err    error
}

type Model struct {
	dir         string
	months      []store.MonthFile
	monthCursor int

	filePath   string
	entries    []store.Entry
	dirty      bool
	listCursor int
	viewport   viewport.Model

	selectedIdx int
	fieldCursor int
	isNewEntry  bool

	editInput   textinput.Model
	editField   int
	editOrigVal string

	// used when creating a new month file
	newMonthInput textinput.Model
	creatingMonth bool

	state         AppState
	width, height int
	statusMsg     string
	statusErr     bool

	renderCursor   int
	renderSelected []bool

	entrySelectEntries  []store.Entry
	entrySources        []string
	entrySelectSelected []bool
	entrySelectCursor   int
	renderCombine       bool

	renderHistory []string
	historyCursor int

	undoStack    []undoState
	redoStack    []undoState
	savedEntries []store.Entry

	newEntries    map[int]bool
	dirtyEntries  map[int]bool
	stagedDeletes int

	deleteConfirmPending bool
	userName             string

	nvoiceBin string
}

func New(dir string, months []store.MonthFile, nvoiceBin string, userName string) Model {
	ti := textinput.New()
	ti.CharLimit = 256

	ni := textinput.New()
	ni.Placeholder = "july-2026"
	ni.CharLimit = 32

	vp := viewport.New(0, 0)

	return Model{
		dir:           dir,
		months:        months,
		nvoiceBin:     nvoiceBin,
		userName:      userName,
		editInput:     ti,
		newMonthInput: ni,
		viewport:      vp,
		state:         StateMonths,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = m.height - 4
		return m, nil

	case store.SaveOkMsg:
		m.dirty = false
		m.isNewEntry = false
		m.newEntries = nil
		m.dirtyEntries = nil
		m.stagedDeletes = 0
		m.savedEntries = deepCopyEntries(m.entries)
		m.statusMsg = "Saved."
		m.statusErr = false
		m.months = refreshMonths(m.dir, m.months)
		return m, nil

	case store.SaveErrMsg:
		m.statusErr = true
		m.statusMsg = "Save error: " + msg.Err.Error()
		return m, nil

	case openDoneMsg:
		if msg.count == 0 {
			m.statusErr = true
			m.statusMsg = "No PDFs found in " + msg.path
		} else {
			m.statusErr = false
			m.statusMsg = fmt.Sprintf("Opened %d PDF(s)", msg.count)
		}
		return m, nil

	case RenderDoneMsg:
		if msg.Err != nil {
			m.state = StateList
			m.statusErr = true
			m.statusMsg = "Render failed: " + msg.Err.Error()
		} else {
			for _, line := range strings.Split(msg.Output, "\n") {
				if strings.HasPrefix(line, "/nix/store/") {
					m.renderHistory = append(m.renderHistory, line)
				}
			}
			m.historyCursor = len(m.renderHistory) - 1
			m.state = StateRenderHistory
			m.statusErr = false
			m.statusMsg = ""
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case StateMonths:
			return updateMonths(m, msg)
		case StateList:
			return updateList(m, msg)
		case StateDetail:
			return updateDetail(m, msg)
		case StateEdit:
			return updateEdit(m, msg)
		case StateRenderSelect:
			return updateRenderSelect(m, msg)
		case StateEntrySelect:
			return updateEntrySelect(m, msg)
		case StateRenderHistory:
			return updateRenderHistory(m, msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case StateMonths:
		return viewMonths(m)
	case StateList:
		return viewList(m)
	case StateDetail:
		return viewDetail(m)
	case StateEdit:
		return viewDetail(m)
	case StateRenderSelect:
		return viewRenderSelect(m)
	case StateEntrySelect:
		return viewEntrySelect(m)
	case StateRenderHistory:
		return viewRenderHistory(m)
	}
	return ""
}

func refreshMonths(dir string, current []store.MonthFile) []store.MonthFile {
	months, err := store.ScanDir(dir)
	if err != nil {
		return current
	}
	return months
}
