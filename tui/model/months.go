package model

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
	"github.com/eureka-cpu/nvoice/tui/styles"
)

var validMonthRe = regexp.MustCompile(`^[a-z]+-\d{4}$`)

func updateMonths(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.creatingMonth {
		switch msg.Type {
		case tea.KeyEnter:
			name := strings.TrimSpace(m.newMonthInput.Value())
			if !validMonthRe.MatchString(name) {
				m.statusMsg = "Name must be like july-2026"
				m.statusErr = true
				return m, nil
			}
			path := filepath.Join(m.dir, name+".json")
			if _, err := os.Stat(path); err == nil {
				m.statusMsg = name + ".json already exists"
				m.statusErr = true
				m.creatingMonth = false
				return m, nil
			}
			if err := store.CreateEmpty(path); err != nil {
				m.statusErr = true
				m.statusMsg = "Could not create file: " + err.Error()
				m.creatingMonth = false
				return m, nil
			}
			m.creatingMonth = false
			m.months = refreshMonths(m.dir, m.months)
			return openMonth(m, path)

		case tea.KeyEsc:
			m.creatingMonth = false
			m.statusMsg = ""
			return m, nil

		default:
			var cmd tea.Cmd
			m.newMonthInput, cmd = m.newMonthInput.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {
	case "q", "b", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.monthCursor > 0 {
			m.monthCursor--
		}
	case "down", "j":
		if m.monthCursor < len(m.months)-1 {
			m.monthCursor++
		}
	case "enter":
		if len(m.months) > 0 {
			return openMonth(m, m.months[m.monthCursor].Path)
		}
	case "n":
		m.creatingMonth = true
		m.newMonthInput.SetValue("")
		m.statusMsg = ""
		return m, textinput.Blink
	}
	return m, nil
}

func openMonth(m Model, path string) (tea.Model, tea.Cmd) {
	entries, err := store.Load(path)
	if err != nil {
		m.statusErr = true
		m.statusMsg = "Load error: " + err.Error()
		return m, nil
	}
	m.filePath = path
	m.entries = entries
	m.savedEntries = deepCopyEntries(entries)
	m.undoStack = nil
	m.redoStack = nil
	m.newEntries = nil
	m.dirtyEntries = nil
	m.stagedDeletes = 0
	m.dirty = false
	m.listCursor = 0
	m.statusMsg = ""
	m.statusErr = false
	m.state = StateList
	return m, nil
}

func viewMonths(m Model) string {
	var sb strings.Builder

	sb.WriteString(styles.Header.Render("nvoice") + "  " + styles.Footer.Render(m.dir) + "\n\n")

	if len(m.months) == 0 && !m.creatingMonth {
		sb.WriteString(styles.Auto.Render("No month files found. Press n to create one.") + "\n")
	}

	for i, mf := range m.months {
		line := fmt.Sprintf("%-20s  %d entries", mf.Name, mf.EntryCount)
		if i == m.monthCursor && !m.creatingMonth {
			sb.WriteString(styles.Cursor.Render("► ") + styles.Selected.Render(line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	if m.creatingMonth {
		sb.WriteString("\n" + styles.Cursor.Render("► ") + "New month: " + m.newMonthInput.View() + "\n")
	}

	sb.WriteString("\n" + statusBar(m))
	sb.WriteString("\n" + styles.Footer.Render("[↑↓] navigate  [Enter] open  [n] new  [q] quit"))

	return sb.String()
}
