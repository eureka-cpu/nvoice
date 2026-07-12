package model

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/store"
	"github.com/eureka-cpu/nvoice/tui/styles"
)

// StateRenderSelect — pick which month files to pull entries from

func updateRenderSelect(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b", "q":
		m.state = StateList
		m.statusMsg = ""
		m.statusErr = false

	case "up", "k":
		if m.renderCursor > 0 {
			m.renderCursor--
		}

	case "down", "j":
		if m.renderCursor < len(m.months)-1 {
			m.renderCursor++
		}

	case " ":
		if len(m.renderSelected) > m.renderCursor {
			m.renderSelected[m.renderCursor] = !m.renderSelected[m.renderCursor]
		}

	case "a":
		anyUnselected := false
		for _, s := range m.renderSelected {
			if !s {
				anyUnselected = true
				break
			}
		}
		for i := range m.renderSelected {
			m.renderSelected[i] = anyUnselected
		}

	case "enter":
		return enterEntrySelect(m)
	}
	return m, nil
}

func enterEntrySelect(m Model) (tea.Model, tea.Cmd) {
	var entries []store.Entry
	var sources []string
	for i, sel := range m.renderSelected {
		if !sel {
			continue
		}
		var loaded []store.Entry
		if m.months[i].Path == m.filePath {
			loaded = m.entries
		} else {
			var err error
			loaded, err = store.Load(m.months[i].Path)
			if err != nil {
				m.statusErr = true
				m.statusMsg = "Load error: " + err.Error()
				return m, nil
			}
		}
		for range loaded {
			sources = append(sources, m.months[i].Path)
		}
		entries = append(entries, loaded...)
	}
	if len(entries) == 0 {
		m.statusErr = true
		m.statusMsg = "No entries in selected files"
		return m, nil
	}
	selected := make([]bool, len(entries))
	for i := range selected {
		selected[i] = true
	}
	m.entrySelectEntries = entries
	m.entrySources = sources
	m.entrySelectSelected = selected
	m.entrySelectCursor = 0
	m.renderCombine = false
	m.state = StateEntrySelect
	m.statusMsg = ""
	return m, nil
}

func viewRenderSelect(m Model) string {
	var sb strings.Builder

	name := pathToName(m.filePath)
	sb.WriteString(styles.Header.Render(name) + "  " + styles.Footer.Render("Select months") + "\n\n")

	for i, mf := range m.months {
		var check string
		if i < len(m.renderSelected) && m.renderSelected[i] {
			check = "[✓]"
		} else {
			check = "[ ]"
		}
		line := fmt.Sprintf("%s %-20s  (%d entries)", check, mf.Name, mf.EntryCount)
		if i == m.renderCursor {
			sb.WriteString(styles.Cursor.Render("► ") + styles.Selected.Render(line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n" + styles.Footer.Render("[Space] toggle  [a] all  [Enter] pick entries  [b/Esc] cancel"))

	return sb.String()
}

// StateEntrySelect — pick which entries to include, set combine, render

func updateEntrySelect(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b":
		m.state = StateRenderSelect
		m.statusMsg = ""
		m.statusErr = false

	case "q":
		m.state = StateList
		m.statusMsg = ""
		m.statusErr = false

	case "up", "k":
		if m.entrySelectCursor > 0 {
			m.entrySelectCursor--
		}

	case "down", "j":
		if m.entrySelectCursor < len(m.entrySelectEntries)-1 {
			m.entrySelectCursor++
		}

	case " ":
		if m.entrySelectCursor < len(m.entrySelectSelected) {
			m.entrySelectSelected[m.entrySelectCursor] = !m.entrySelectSelected[m.entrySelectCursor]
		}

	case "a":
		anyUnselected := false
		for _, s := range m.entrySelectSelected {
			if !s {
				anyUnselected = true
				break
			}
		}
		for i := range m.entrySelectSelected {
			m.entrySelectSelected[i] = anyUnselected
		}

	case "c":
		m.renderCombine = !m.renderCombine

	case "enter":
		return m, doRenderEntries(m)
	}
	return m, nil
}

func doRenderEntries(m Model) tea.Cmd {
	if m.nvoiceBin == "" {
		return func() tea.Msg {
			return RenderDoneMsg{Err: fmt.Errorf("NVOICE not set")}
		}
	}

	type item struct {
		entry  store.Entry
		source string
	}
	var selected []item
	for i, sel := range m.entrySelectSelected {
		if sel {
			src := ""
			if i < len(m.entrySources) {
				src = m.entrySources[i]
			}
			selected = append(selected, item{m.entrySelectEntries[i], src})
		}
	}

	nvoiceBin := m.nvoiceBin
	combine := m.renderCombine

	env := os.Environ()
	if configPath, err := filepath.Abs(filepath.Join(m.dir, "config.nix")); err == nil && fileExists(configPath) {
		env = append(env, "NVOICE_CONFIG="+configPath)
	}

	return func() tea.Msg {
		if len(selected) == 0 {
			return RenderDoneMsg{Err: fmt.Errorf("no entries selected")}
		}

		if combine {
			entries := make([]store.Entry, len(selected))
			for i, s := range selected {
				entries[i] = s.entry
			}
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return RenderDoneMsg{Err: err}
			}
			return runWithTempFile(nvoiceBin, env, data, "--combine")
		}

		// group by source file to render each month as its own invoice
		type fileGroup struct {
			path    string
			entries []store.Entry
		}
		seen := map[string]int{}
		var groups []fileGroup
		for _, s := range selected {
			if idx, ok := seen[s.source]; ok {
				groups[idx].entries = append(groups[idx].entries, s.entry)
			} else {
				seen[s.source] = len(groups)
				groups = append(groups, fileGroup{s.source, []store.Entry{s.entry}})
			}
		}

		var outputs []string
		for _, g := range groups {
			data, err := json.MarshalIndent(g.entries, "", "  ")
			if err != nil {
				return RenderDoneMsg{Err: err}
			}
			msg := runWithTempFile(nvoiceBin, env, data)
			done := msg.(RenderDoneMsg)
			if done.Err != nil {
				return RenderDoneMsg{Err: fmt.Errorf("%s: %w", pathToName(g.path), done.Err)}
			}
			outputs = append(outputs, done.Output)
		}
		return RenderDoneMsg{Output: strings.Join(outputs, "\n")}
	}
}

func runWithTempFile(nvoiceBin string, env []string, data []byte, extraArgs ...string) tea.Msg {
	tmp, err := os.CreateTemp("", "nvoice-*.json")
	if err != nil {
		return RenderDoneMsg{Err: err}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return RenderDoneMsg{Err: err}
	}
	tmp.Close()
	args := append(extraArgs, tmpPath)
	cmd := exec.Command(nvoiceBin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return RenderDoneMsg{Err: fmt.Errorf("%s", strings.TrimSpace(string(out)))}
	}
	return RenderDoneMsg{Output: strings.TrimSpace(string(out))}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func viewEntrySelect(m Model) string {
	var sb strings.Builder

	multiMonth := len(uniqueSources(m.entrySources)) > 1

	sb.WriteString(styles.Header.Render("Select entries") + "\n\n")

	for i, e := range m.entrySelectEntries {
		var check string
		if i < len(m.entrySelectSelected) && m.entrySelectSelected[i] {
			check = "[✓]"
		} else {
			check = "[ ]"
		}

		date := FormatDate(GetField(e, "end_date"))
		if date == "" {
			date = "—"
		}
		hours := GetField(e, "rounded_hours")
		task := GetField(e, "task")
		if len(task) > 28 {
			task = task[:25] + "..."
		}
		currency := GetField(e, "source_currency")
		cost := GetField(e, "source_cost")

		var suffix string
		if multiMonth && i < len(m.entrySources) {
			suffix = "  " + styles.Auto.Render(pathToName(m.entrySources[i]))
		}

		line := fmt.Sprintf("%s %-12s  %5sh  %-30s  %s %s%s",
			check, date, hours, task, currency, cost, suffix)

		if i == m.entrySelectCursor {
			sb.WriteString(styles.Cursor.Render("► ") + styles.Selected.Render(line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	combineLabel := "OFF"
	if m.renderCombine {
		combineLabel = "ON "
	}
	sb.WriteString("\n  Combine: " + styles.Value.Render(combineLabel) + "  " + styles.Auto.Render("(press c to toggle)") + "\n")

	sb.WriteString("\n" + styles.Footer.Render("[Space] toggle  [a] all  [c] combine  [Enter] render  [b] back  [q] cancel"))

	return sb.String()
}

func uniqueSources(sources []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range sources {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
