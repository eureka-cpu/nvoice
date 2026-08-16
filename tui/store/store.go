package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var monthOrder = map[string]int{
	"january": 1, "jan": 1,
	"february": 2, "feb": 2,
	"march": 3, "mar": 3,
	"april": 4, "apr": 4,
	"may": 5,
	"june": 6, "jun": 6,
	"july": 7, "jul": 7,
	"august": 8, "aug": 8,
	"september": 9, "sep": 9,
	"october": 10, "oct": 10,
	"november": 11, "nov": 11,
	"december": 12, "dec": 12,
}

// matches files that begin with a month name (full or abbreviated) followed by a 4-digit year
var monthPrefixRe = regexp.MustCompile(`^(january|february|march|april|may|june|july|august|september|october|november|december|jan|feb|mar|apr|jun|jul|aug|sep|oct|nov|dec)-(\d{4})`)

type MonthFile struct {
	Name       string
	Path       string
	EntryCount int
}

type SaveOkMsg struct{}
type SaveErrMsg struct{ Err error }

type Entry map[string]interface{}

// fileSortKey returns a string that sorts month-prefixed files chronologically
// (e.g. "july-2026", "aug-2026-anduril") and everything else alphabetically after.
func fileSortKey(name string) string {
	m := monthPrefixRe.FindStringSubmatch(name)
	if m == nil {
		return "z:" + name
	}
	month := fmt.Sprintf("%02d", monthOrder[m[1]])
	return m[2] + "-" + month + ":" + name
}

func ScanDir(dir string) ([]MonthFile, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	var files []MonthFile
	for _, path := range matches {
		entries, err := Load(path)
		if err != nil {
			continue // skip files that aren't valid entry arrays (schemas, configs, etc.)
		}
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		files = append(files, MonthFile{
			Name:       name,
			Path:       path,
			EntryCount: len(entries),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return fileSortKey(files[i].Name) < fileSortKey(files[j].Name)
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
