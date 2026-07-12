package model

import (
	"encoding/json"

	"github.com/eureka-cpu/nvoice/tui/store"
)

type undoState struct {
	entries       []store.Entry
	newEntries    map[int]bool
	dirtyEntries  map[int]bool
	stagedDeletes int
	listCursor    int
}

func deepCopyEntries(entries []store.Entry) []store.Entry {
	if len(entries) == 0 {
		return nil
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return nil
	}
	var copied []store.Entry
	json.Unmarshal(data, &copied)
	return copied
}

func captureState(m Model) undoState {
	data, _ := json.Marshal(m.entries)
	var entries []store.Entry
	json.Unmarshal(data, &entries)
	ne := make(map[int]bool, len(m.newEntries))
	for k, v := range m.newEntries {
		ne[k] = v
	}
	de := make(map[int]bool, len(m.dirtyEntries))
	for k, v := range m.dirtyEntries {
		de[k] = v
	}
	return undoState{entries, ne, de, m.stagedDeletes, m.listCursor}
}

func pushUndo(m *Model) {
	const maxUndo = 100
	m.undoStack = append(m.undoStack, captureState(*m))
	if len(m.undoStack) > maxUndo {
		m.undoStack = m.undoStack[1:]
	}
	m.redoStack = nil
}

func restoreState(m Model, s undoState) Model {
	m.entries = s.entries
	m.newEntries = s.newEntries
	m.dirtyEntries = s.dirtyEntries
	m.stagedDeletes = s.stagedDeletes
	m.listCursor = s.listCursor
	m.state = StateList
	m.statusMsg = ""
	m.statusErr = false
	da, _ := json.Marshal(m.entries)
	db, _ := json.Marshal(m.savedEntries)
	m.dirty = string(da) != string(db)
	return m
}

func applyUndo(m Model) (Model, bool) {
	if len(m.undoStack) == 0 {
		return m, false
	}
	m.redoStack = append(m.redoStack, captureState(m))
	s := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	return restoreState(m, s), true
}

func applyRedo(m Model) (Model, bool) {
	if len(m.redoStack) == 0 {
		return m, false
	}
	m.undoStack = append(m.undoStack, captureState(m))
	s := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	return restoreState(m, s), true
}
