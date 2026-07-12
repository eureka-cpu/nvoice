package model

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/styles"
)

type openDoneMsg struct {
	count int
	path  string
}

func updateRenderHistory(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b", "q":
		m.state = StateList
		m.statusMsg = ""
		m.statusErr = false

	case "up", "k":
		if m.historyCursor > 0 {
			m.historyCursor--
		}

	case "down", "j":
		if m.historyCursor < len(m.renderHistory)-1 {
			m.historyCursor++
		}

	case "enter", "o":
		if len(m.renderHistory) > 0 {
			path := m.renderHistory[m.historyCursor]
			return m, openStorePath(path)
		}
	}
	return m, nil
}

func openStorePath(path string) tea.Cmd {
	return func() tea.Msg {
		pdfs, _ := filepath.Glob(path + "/*.pdf")
		if len(pdfs) == 0 {
			return openDoneMsg{count: 0, path: path}
		}
		opener := "xdg-open"
		if runtime.GOOS == "darwin" {
			opener = "open"
		}
		for _, pdf := range pdfs {
			exec.Command(opener, pdf).Start()
		}
		return openDoneMsg{count: len(pdfs), path: path}
	}
}

func viewRenderHistory(m Model) string {
	var sb strings.Builder

	sb.WriteString(styles.Header.Render("Render History") + "\n\n")

	if len(m.renderHistory) == 0 {
		sb.WriteString(styles.Auto.Render("No renders this session.") + "\n")
	}

	for i, path := range m.renderHistory {
		pdfs, _ := filepath.Glob(path + "/*.pdf")
		var names []string
		for _, p := range pdfs {
			names = append(names, filepath.Base(p))
		}
		pdfLabel := styles.Auto.Render("(no PDFs)")
		if len(names) > 0 {
			pdfLabel = styles.Value.Render(strings.Join(names, ", "))
		}
		line := fmt.Sprintf("%-52s  %s", path, pdfLabel)
		if i == m.historyCursor {
			sb.WriteString(styles.Cursor.Render("► ") + styles.Selected.Render(line) + "\n")
		} else {
			sb.WriteString("  " + line + "\n")
		}
	}

	sb.WriteString("\n" + statusBar(m))
	sb.WriteString("\n" + styles.Footer.Render("[↑↓] navigate  [Enter/o] open PDFs  [b/Esc] back"))

	return sb.String()
}
