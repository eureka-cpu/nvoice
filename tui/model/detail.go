package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
	"github.com/eureka-cpu/nvoice/tui/styles"
)

func updateDetail(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b":
		m.state = StateList
		m.statusMsg = ""
		m.statusErr = false

	case "u":
		nm, ok := applyUndo(m)
		if !ok {
			m.statusMsg = "Nothing to undo"
			return m, nil
		}
		return nm, nil

	case "U":
		nm, ok := applyRedo(m)
		if !ok {
			m.statusMsg = "Nothing to redo"
			return m, nil
		}
		return nm, nil

	case "up", "k":
		if m.fieldCursor > 0 {
			m.fieldCursor--
		}

	case "down", "j":
		if m.fieldCursor < len(Fields)-1 {
			m.fieldCursor++
		}

	case "enter":
		fd := Fields[m.fieldCursor]
		current := GetField(m.entries[m.selectedIdx], fd.Key)
		m.editOrigVal = current
		m.editField = m.fieldCursor
		m.editInput.SetValue(current)
		m.editInput.Focus()
		m.editInput.CursorEnd()
		m.state = StateEdit
		m.statusMsg = ""
		return m, textinput.Blink

	case "s":
		m.isNewEntry = false
		return m, store.Save(m.filePath, m.entries)
	}
	return m, nil
}

func viewDetail(m Model) string {
	var sb strings.Builder

	name := pathToName(m.filePath)
	idx := fmt.Sprintf("Entry %d/%d", m.selectedIdx+1, len(m.entries))
	title := styles.Header.Render(name) + "  " + styles.Footer.Render(idx)
	if m.dirty {
		title += "  " + styles.Dirty
	}
	sb.WriteString(title + "\n\n")

	e := m.entries[m.selectedIdx]

	for i, fd := range Fields {
		label := styles.Label.Render(fd.Label)
		focused := i == m.fieldCursor

		var valueStr string
		if m.state == StateEdit && focused {
			valueStr = m.editInput.View()
		} else {
			val := GetField(e, fd.Key)
			if val == "" {
				val = styles.Auto.Render("—")
			} else if fd.Auto {
				val = styles.Value.Render(val) + "  " + styles.Auto.Render("(auto)")
			} else {
				val = styles.Value.Render(val)
			}
			valueStr = val
		}

		if focused {
			sb.WriteString(styles.Cursor.Render("► ") + label + valueStr + "\n")
		} else {
			sb.WriteString("  " + label + valueStr + "\n")
		}
	}

	sb.WriteString("\n" + statusBar(m))
	sb.WriteString("\n" + styles.Footer.Render("[↑↓] navigate  [Enter] edit  [s] save  [u]ndo  [U]redo  [Esc] back"))

	return sb.String()
}

func statusBar(m Model) string {
	if m.statusMsg == "" {
		return ""
	}
	if m.statusErr {
		return styles.ErrStyle.Render(m.statusMsg)
	}
	return styles.Status.Render(m.statusMsg)
}
