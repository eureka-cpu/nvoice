package model

import (
	"fmt"
	"math"
	"strconv"

	"github.com/eureka-cpu/nvoice/tui/store"
)

type FieldKind int

const (
	KindString FieldKind = iota
	KindFloat
	KindDate
)

type FieldDef struct {
	Key      string
	Label    string
	Kind     FieldKind
	Triggers bool // editing this field calls Recalculate
	Auto     bool // derived by Recalculate; shown with (auto) suffix
}

var Fields = []FieldDef{
	{"agency", "Agency", KindString, false, false},
	{"client", "Client", KindString, false, false},
	{"start_date", "Start Date", KindDate, false, false},
	{"end_date", "End Date", KindDate, false, false},
	{"task", "Task", KindString, false, false},
	{"user", "User", KindString, false, false},
	{"rounded_hours", "Hours", KindFloat, true, false},
	{"exchange_rate", "Exchange Rate", KindFloat, true, false},
	{"source_currency", "Source Currency", KindString, false, false},
	{"source_hourly_rate", "Source Rate", KindFloat, true, false},
	{"target_currency", "Target Currency", KindString, false, false},
	{"source_cost", "Source Cost", KindFloat, false, true},
	{"target_hourly_rate", "Target Rate", KindFloat, false, true},
	{"target_cost", "Target Cost", KindFloat, false, true},
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func Recalculate(e store.Entry) store.Entry {
	hours, _ := toFloat(e["rounded_hours"])
	rate, _ := toFloat(e["source_hourly_rate"])
	exch, _ := toFloat(e["exchange_rate"])
	if exch == 0 {
		exch = 1.0
	}
	if hours > 0 && rate > 0 {
		e["source_cost"] = round2(hours * rate)
		e["target_hourly_rate"] = round2(rate * exch)
		e["target_cost"] = round2(hours * round2(rate*exch))
	}
	return e
}

func GetField(e store.Entry, key string) string {
	v, ok := e[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Trunc(val) {
			return strconv.FormatFloat(val, 'f', 1, 64)
		}
		return strconv.FormatFloat(val, 'f', 2, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func SetField(e store.Entry, key string, raw string, kind FieldKind) error {
	if raw == "" {
		delete(e, key)
		return nil
	}
	switch kind {
	case KindFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		e[key] = f
	case KindDate:
		if len(raw) != 8 {
			return fmt.Errorf("must be YYYYMMDD")
		}
		for _, c := range raw {
			if c < '0' || c > '9' {
				return fmt.Errorf("must be YYYYMMDD")
			}
		}
		e[key] = raw
	default:
		e[key] = raw
	}
	return nil
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	}
	return 0, false
}

func FormatDate(s string) string {
	if len(s) != 8 {
		return s
	}
	return s[0:4] + "-" + s[4:6] + "-" + s[6:8]
}
