package model

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
)

func updateEdit(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		fd := Fields[m.editField]
		raw := m.editInput.Value()
		// Validate on a copy first so a bad value doesn't corrupt the undo stack.
		tmp := make(store.Entry, len(m.entries[m.selectedIdx]))
		for k, v := range m.entries[m.selectedIdx] {
			tmp[k] = v
		}
		if err := SetField(tmp, fd.Key, raw, fd.Kind); err != nil {
			m.statusMsg = fd.Label + ": " + err.Error()
			m.statusErr = true
			return m, nil
		}
		pushUndo(&m)
		m.entries[m.selectedIdx] = tmp
		if fd.Triggers {
			m.entries[m.selectedIdx] = Recalculate(m.entries[m.selectedIdx])
		}
		m.dirty = true
		if m.dirtyEntries == nil {
			m.dirtyEntries = make(map[int]bool)
		}
		m.dirtyEntries[m.selectedIdx] = true
		m.statusMsg = ""
		m.statusErr = false
		m.editInput.Blur()
		m.state = StateDetail
		return m, nil

	case tea.KeyEsc:
		_ = SetField(m.entries[m.selectedIdx], Fields[m.editField].Key, m.editOrigVal, Fields[m.editField].Kind)
		m.statusMsg = ""
		m.statusErr = false
		m.editInput.Blur()
		m.state = StateDetail
		return m, nil
	}

	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}
