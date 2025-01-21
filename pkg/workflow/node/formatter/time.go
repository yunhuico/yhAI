package formatter

import (
	"embed"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

const (
	now       = "now"
	today     = "today"
	yesterday = "yesterday"
	tomorrow  = "tomorrow"
)

const (
	auto1           = ""
	auto2           = "auto"
	timestampFormat = "timestamp"
)

const (
	calcTime = "Add/Subtract Time"
	formTime = "Format"
)

const (
	typeYears   = 0
	typeMonths  = 1
	typeWeeks   = 3
	typeDays    = 4
	dateTypeCnt = 4
)

const negativeSymbol = 2

var convertFormatTable = [][2]string{
	{"hours", "h"},
	{"minutes", "m"},
	{"seconds", "s"},
}

var dateDescs = []string{"years", "months", "weeks", "days"}
var datePattern = regexp.MustCompile(`(\+)?(-)?(\s*(\d+)\s*years?,?)?(\s*(\d+)\s*months?,?)?(\s*(\d+)\s*weeks?,?)?(\s*(\d+)\s*days?,?)?`)

var utcTZ *time.Location

func init() {
	var err error
	utcTZ, err = time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}
}

type FormatTime struct {
	InputTime    string `json:"inputTime"`
	ToFormat     string `json:"toFormat"`
	FromFormat   string `json:"fromFormat"`
	ToTimezone   string `json:"toTimezone"`
	FromTimezone string `json:"fromTimezone"`
}

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)

	workflow.RegistryNodeMeta(&FormatTime{})
}

func (f *FormatTime) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/format#datetime")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(FormatTime)
		},
		InputForm: spec.InputSchema,
	}
}

func (f *FormatTime) Run(_ *workflow.NodeContext) (output any, err error) {
	if len(f.ToFormat) == 0 {
		return nil, fmt.Errorf("emtpy toformat")
	}

	parsedInput, err := f.parseInput()
	if err != nil {
		err = fmt.Errorf("invalid input: %w", err)
		return
	}

	if !isAuto(f.ToTimezone) {
		var tz *time.Location
		tz, err = time.LoadLocation(f.ToTimezone)
		if err != nil {
			err = fmt.Errorf("parse to timezone: %w", err)
			return
		}
		parsedInput = parsedInput.In(tz)
	}

	output = parsedInput.Format(f.ToFormat)

	return output, nil
}

func isAuto(t string) bool {
	return t == auto1 || t == auto2
}

func (f *FormatTime) handleRelativeTime() (t time.Time, isRelative bool, err error) {
	// use given from timezone to generate a current time.
	nowTime := time.Now()

	if !isAuto(f.FromFormat) {
		var tz *time.Location
		tz, err = time.LoadLocation(f.FromTimezone)
		if err != nil {
			err = fmt.Errorf("load from timezone: %w", err)
			return
		}
		nowTime = nowTime.In(tz)
	}

	isRelative = true
	switch f.InputTime {
	case now:
		t = nowTime
	case today:
		year, month, day := nowTime.Date()
		t = time.Date(year, month, day, 0, 0, 0, 0, utcTZ)
	case yesterday:
		t = nowTime.Add(-time.Hour * 24)
	case tomorrow:
		t = nowTime.Add(time.Hour * 24)
	default:
		isRelative = false
	}

	return
}

func (f *FormatTime) parseInput() (t time.Time, err error) {
	t, isRelative, err := f.handleRelativeTime()
	if err != nil {
		return
	}
	if isRelative {
		return
	}

	defer func() {
		if err != nil {
			return
		}
		if !isAuto(f.FromFormat) {
			var tz *time.Location
			tz, err = time.LoadLocation(f.FromTimezone)
			if err != nil {
				err = fmt.Errorf("load from timezone: %w", err)
				return
			}
			t = t.In(tz)
		}
	}()

	switch f.FromFormat {
	case timestampFormat:
		var timestamp int64
		timestamp, err = strconv.ParseInt(f.InputTime, 10, 64)
		if err != nil {
			err = fmt.Errorf("parse timestamp: %w", err)
			return
		}
		t = time.Unix(timestamp, 0)
		return
	case auto1: // empty format
		fallthrough
	case auto2:
		t, err = dateparse.ParseAny(f.InputTime)
		if err != nil {
			err = fmt.Errorf("can't parse time: %w", err)
			return
		}
	default:
		t, err = time.Parse(f.FromFormat, f.InputTime)
		if err != nil {
			err = fmt.Errorf("parse time %q by %q", f.InputTime, f.FromFormat)
			return
		}
	}

	return
}
