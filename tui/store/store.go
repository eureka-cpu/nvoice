package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var monthOrder = map[string]int{
	"january": 1, "february": 2, "march": 3, "april": 4,
	"may": 5, "june": 6, "july": 7, "august": 8,
	"september": 9, "october": 10, "november": 11, "december": 12,
}

var monthFileRe = regexp.MustCompile(`^([a-z]+)-(\d{4})\.json$`)

type MonthFile struct {
	Name       string
	Path       string
	EntryCount int
}

type SaveOkMsg struct{}
type SaveErrMsg struct{ Err error }

type Entry map[string]interface{}

func ScanDir(dir string) ([]MonthFile, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	var files []MonthFile
	for _, path := range matches {
		name := filepath.Base(path)
		m := monthFileRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		entries, _ := Load(path)
		files = append(files, MonthFile{
			Name:       strings.TrimSuffix(name, ".json"),
			Path:       path,
			EntryCount: len(entries),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		mi := monthFileRe.FindStringSubmatch(filepath.Base(files[i].Path))
		mj := monthFileRe.FindStringSubmatch(filepath.Base(files[j].Path))
		yi, _ := strconv.Atoi(mi[2])
		yj, _ := strconv.Atoi(mj[2])
		if yi != yj {
			return yi < yj
		}
		return monthOrder[mi[1]] < monthOrder[mj[1]]
	})

	return files, nil
}

func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return entries, nil
}

func CreateEmpty(path string) error {
	return os.WriteFile(path, []byte("[]\n"), 0644)
}

func Save(path string, entries []Entry) tea.Cmd {
	return func() tea.Msg {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return SaveErrMsg{err}
		}
		data = append(data, '\n')
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0644); err != nil {
			return SaveErrMsg{err}
		}
		if err := os.Rename(tmp, path); err != nil {
			return SaveErrMsg{err}
		}
		return SaveOkMsg{}
	}
}
