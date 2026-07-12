package model

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
	"github.com/eureka-cpu/nvoice/tui/styles"
)

func updateList(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.dirty {
			m.statusMsg = "Unsaved changes — press s to save or q again to discard"
			m.statusErr = true
			m.dirty = false
			return m, nil
		}
		return m, tea.Quit

	case "esc", "b":
		if m.deleteConfirmPending {
			m.deleteConfirmPending = false
			m.statusMsg = ""
			return m, nil
		}
		if m.dirty {
			m.statusMsg = "Unsaved changes — press s to save first"
			m.statusErr = true
			return m, nil
		}
		m.state = StateMonths
		m.statusMsg = ""
		m.statusErr = false

	case "up", "k":
		m.deleteConfirmPending = false
		if m.listCursor > 0 {
			m.listCursor--
		}

	case "down", "j":
		m.deleteConfirmPending = false
		if m.listCursor < len(m.entries)-1 {
			m.listCursor++
		}

	case "enter":
		if len(m.entries) > 0 {
			m.selectedIdx = m.listCursor
			m.fieldCursor = 0
			m.isNewEntry = false
			m.state = StateDetail
			m.statusMsg = ""
		}

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

	case "h":
		m.state = StateRenderHistory
		m.statusMsg = ""

	case "n":
		pushUndo(&m)
		today := time.Now().Format("20060102")
		m.entries = append(m.entries, store.Entry{
			"exchange_rate": 1.0,
			"user":          m.userName,
			"end_date":      today,
			"start_date":    today,
		})
		idx := len(m.entries) - 1
		if m.newEntries == nil {
			m.newEntries = make(map[int]bool)
		}
		m.newEntries[idx] = true
		m.selectedIdx = idx
		m.fieldCursor = 0
		m.isNewEntry = true
		m.dirty = true
		m.state = StateDetail
		m.statusMsg = ""

	case "d":
		if len(m.entries) > 0 {
			if m.deleteConfirmPending {
				pushUndo(&m)
				del := m.listCursor
				m.entries = append(m.entries[:del], m.entries[del+1:]...)
				if m.listCursor >= len(m.entries) && m.listCursor > 0 {
					m.listCursor--
				}
				m.newEntries = shiftDown(m.newEntries, del)
				m.dirtyEntries = shiftDown(m.dirtyEntries, del)
				m.stagedDeletes++
				m.dirty = true
				m.deleteConfirmPending = false
				m.statusMsg = "Entry deleted — press s to save"
				m.statusErr = false
			} else {
				m.deleteConfirmPending = true
				m.statusMsg = "Delete this entry? Press d to confirm, Esc to cancel"
				m.statusErr = false
			}
		}

	case "s":
		return m, store.Save(m.filePath, m.entries)

	case "r":
		sel := make([]bool, len(m.months))
		for i, mf := range m.months {
			if mf.Path == m.filePath {
				sel[i] = true
				break
			}
		}
		m.renderSelected = sel
		m.renderCursor = 0
		m.renderCombine = false
		m.state = StateRenderSelect
		m.statusMsg = ""
	}
	return m, nil
}

func shiftDown(m map[int]bool, del int) map[int]bool {
	if len(m) == 0 {
		return m
	}
	out := make(map[int]bool, len(m))
	for idx := range m {
		if idx == del {
			continue
		}
		if idx > del {
			out[idx-1] = true
		} else {
			out[idx] = true
		}
	}
	return out
}

func viewList(m Model) string {
	var sb strings.Builder

	name := pathToName(m.filePath)
	title := styles.Header.Render(name)
	if m.dirty {
		newCount := len(m.newEntries)
		updatedCount := 0
		for idx := range m.dirtyEntries {
			if !m.newEntries[idx] {
				updatedCount++
			}
		}
		var parts []string
		if newCount > 0 {
			parts = append(parts, fmt.Sprintf("%d new", newCount))
		}
		if updatedCount > 0 {
			parts = append(parts, fmt.Sprintf("%d updated", updatedCount))
		}
		if m.stagedDeletes > 0 {
			parts = append(parts, fmt.Sprintf("%d deleted", m.stagedDeletes))
		}
		summary := ""
		if len(parts) > 0 {
			summary = "  " + styles.Auto.Render(strings.Join(parts, " · "))
		}
		title += "  " + styles.Dirty + summary + "  " + styles.Auto.Render("dirty")
	}
	sb.WriteString(title + "\n\n")

	if len(m.entries) == 0 {
		sb.WriteString(styles.Auto.Render("No entries. Press n to add one.") + "\n")
	}

	for i, e := range m.entries {
		date := FormatDate(GetField(e, "end_date"))
		if date == "" {
			date = "—"
		}
		hours := GetField(e, "rounded_hours")
		task := GetField(e, "task")
		if len(task) > 30 {
			task = task[:27] + "..."
		}
		currency := GetField(e, "source_currency")
		cost := GetField(e, "source_cost")

		line := fmt.Sprintf("%-12s  %5sh  %-32s  %s %s",
			date, hours, task, currency, cost)

		unsaved := m.newEntries[i] || m.dirtyEntries[i]
		if i == m.listCursor {
			sb.WriteString(styles.Cursor.Render("► ") + styles.Selected.Render(line) + "\n")
		} else if unsaved {
			sb.WriteString("  " + styles.Auto.Render(line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n" + listFooter(m))
	sb.WriteString("\n" + statusBar(m))

	return sb.String()
}

func listFooter(m Model) string {
	var totalHours float64
	var totalCost float64
	var currency string
	for _, e := range m.entries {
		h, _ := toFloat(e["rounded_hours"])
		c, _ := toFloat(e["source_cost"])
		totalHours += h
		totalCost += c
		if currency == "" {
			currency, _ = e["source_currency"].(string)
		}
	}
	summary := fmt.Sprintf("Total: %.1fh%s%s %.2f",
		totalHours, styles.Separator, currency, totalCost)
	keys := "[n]ew  [d]elete  [r]ender  [h]istory  [s]ave  [u]ndo  [U]redo  [Esc] back  [q]uit"
	return styles.Footer.Render(summary + "    " + keys)
}

func pathToName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if len(name) > 5 && name[len(name)-5:] == ".json" {
		return name[:len(name)-5]
	}
	return name
}
