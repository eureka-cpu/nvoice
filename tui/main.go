package main

import (
	"fmt"
	"os"
	"os/user"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eureka-cpu/nvoice/tui/model"
	"github.com/eureka-cpu/nvoice/tui/store"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	months, err := store.ScanDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error scanning %s: %v\n", dir, err)
		os.Exit(1)
	}

	nvoiceBin := os.Getenv("NVOICE")

	userName := ""
	if u, err := user.Current(); err == nil {
		userName = u.Name
		if userName == "" {
			userName = u.Username
		}
	}

	m := model.New(dir, months, nvoiceBin, userName)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
