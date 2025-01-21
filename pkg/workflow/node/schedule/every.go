package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&EveryDay{})
	workflow.RegistryNodeMeta(&EveryHour{})
	workflow.RegistryNodeMeta(&EveryMonth{})
	workflow.RegistryNodeMeta(&EveryWeek{})
}

type EveryDay struct {
	everyBase

	// 10:35:00
	Time string `json:"time"`
}

func (e *EveryDay) UnmarshalJSON(data []byte) (err error) {
	type wrapped EveryDay

	var (
		httpRequest workflow.HTTPRequest
		w           wrapped
	)
	err = json.Unmarshal(data, &httpRequest)
	if err == nil && len(httpRequest.Body) > 0 {
		err = json.Unmarshal(httpRequest.Body, &w)
		if err != nil {
			err = fmt.Errorf("unmarshaling HTTP request body into EveryDay: %w", err)
			return
		}

		*e = EveryDay(w)
		return
	}

	err = json.Unmarshal(data, &w)
	if err != nil {
		err = fmt.Errorf("unmarshaling bytes into EveryDay: %w", err)
		return
	}

	*e = EveryDay(w)
	return
}

func (e *EveryDay) GetConfigObject() any {
	return new(EveryDay)
}

func (e *EveryDay) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	config := ctx.GetConfigObject().(*EveryDay)

	if config.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	tz, err := time.LoadLocation(config.Timezone)
	if err != nil {
		err = fmt.Errorf("parsing timezone %q: %w", e.Timezone, err)
		return
	}

	hour, min, second, err := trigger.ParseTimeInDay(config.Time)
	if err != nil {
		err = fmt.Errorf("parsing time in a day: %w", err)
		return
	}

	var (
		now  = time.Now().In(tz)
		next = time.Date(now.Year(), now.Month(), now.Day(), hour, min, second, 0, tz)
	)

	if next.Before(now) {
		next = time.Date(next.Year(), next.Month(), next.Day()+1, next.Hour(), next.Minute(), next.Second(), 0, tz)
	}

	result = []trigger.SampleData{newOutput(next)}
	return
}

func (e *EveryDay) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("everyDay"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(EveryDay)
		},
		InputForm: spec.InputSchema,
	}
}

type EveryHour struct {
	everyBase

	// 0-59
	Minute int `json:"minute"`
}

func (e *EveryHour) UnmarshalJSON(data []byte) (err error) {
	type wrapped EveryHour

	var (
		httpRequest workflow.HTTPRequest
		w           wrapped
	)
	err = json.Unmarshal(data, &httpRequest)
	if err == nil && len(httpRequest.Body) > 0 {
		err = json.Unmarshal(httpRequest.Body, &w)
		if err != nil {
			err = fmt.Errorf("unmarshaling HTTP request body into EveryHour: %w", err)
			return
		}

		*e = EveryHour(w)
		return
	}

	err = json.Unmarshal(data, &w)
	if err != nil {
		err = fmt.Errorf("unmarshaling bytes into EveryHour: %w", err)
		return
	}

	*e = EveryHour(w)
	return
}

func (e *EveryHour) GetConfigObject() any {
	return new(EveryHour)
}

func (e *EveryHour) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	config := ctx.GetConfigObject().(*EveryHour)

	if config.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	tz, err := time.LoadLocation(config.Timezone)
	if err != nil {
		err = fmt.Errorf("parsing timezone %q: %w", e.Timezone, err)
		return
	}

	var (
		now  = time.Now().In(tz)
		next = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), config.Minute, 0, 0, tz)
	)

	if next.Before(now) {
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour()+1, next.Minute(), 0, 0, tz)
	}

	result = []trigger.SampleData{newOutput(next)}
	return
}

func (e *EveryHour) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("everyHour"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(EveryHour)
		},
		InputForm: spec.InputSchema,
	}
}

type EveryMonth struct {
	everyBase

	// 1-31
	DaysOfMonth []int `json:"daysOfMonth"`
	// 10:35:00
	Time string `json:"time"`
}

func (e *EveryMonth) UnmarshalJSON(data []byte) (err error) {
	type wrapped EveryMonth

	var (
		httpRequest workflow.HTTPRequest
		w           wrapped
	)
	err = json.Unmarshal(data, &httpRequest)
	if err == nil && len(httpRequest.Body) > 0 {
		err = json.Unmarshal(httpRequest.Body, &w)
		if err != nil {
			err = fmt.Errorf("unmarshaling HTTP request body into EveryMonth: %w", err)
			return
		}

		*e = EveryMonth(w)
		return
	}

	err = json.Unmarshal(data, &w)
	if err != nil {
		err = fmt.Errorf("unmarshaling bytes into EveryMonth: %w", err)
		return
	}

	*e = EveryMonth(w)
	return
}

func (e *EveryMonth) GetConfigObject() any {
	return new(EveryMonth)
}

func (e *EveryMonth) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	config := ctx.GetConfigObject().(*EveryMonth)

	if config.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	tz, err := time.LoadLocation(config.Timezone)
	if err != nil {
		err = fmt.Errorf("parsing timezone %q: %w", e.Timezone, err)
		return
	}

	if len(config.DaysOfMonth) == 0 {
		err = errors.New("no day of month selected")
		return
	}
	if len(config.DaysOfMonth) > 31 {
		err = errors.New("too many days selected")
		return
	}

	sort.Ints(config.DaysOfMonth)

	var seen int
	for i, v := range config.DaysOfMonth {
		if v < 1 || v > 31 {
			err = fmt.Errorf("invalid day %d at index %d", v, i)
			return
		}
		if v == seen {
			err = fmt.Errorf("duplicated day %d at index %d and %d", v, i-1, i)
			return
		}

		seen = v
	}

	hour, min, second, err := trigger.ParseTimeInDay(config.Time)
	if err != nil {
		err = fmt.Errorf("parsing time of day: %w", err)
		return
	}

	var (
		now  = time.Now().In(tz)
		next time.Time
	)

	// We are sure we can find a valid date in the next following 3 months
OUTER:
	for monthOffset := 0; monthOffset <= 3; monthOffset++ {
		base := time.Date(now.Year(), now.Month()+time.Month(monthOffset), 1, 0, 0, 0, 0, tz)

		for _, day := range config.DaysOfMonth {
			next = time.Date(base.Year(), base.Month(), day, hour, min, second, 0, tz)
			if next.Month() != base.Month() {
				// the day does not exist, e.g. 02-30
				//
				// following days won't exist too, let's skip them all.
				continue OUTER
			}
			if next.After(now) {
				break OUTER
			}
		}
	}

	result = []trigger.SampleData{newOutput(next)}
	return
}

func (e *EveryMonth) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("everyMonth"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(EveryMonth)
		},
		InputForm: spec.InputSchema,
	}
}

type EveryWeek struct {
	everyBase

	// 0-6 as SUN-SAT
	WeekDays []int `json:"weekDays"`
	// 10:35:00
	Time string `json:"time"`
}

func (e *EveryWeek) UnmarshalJSON(data []byte) (err error) {
	type wrapped EveryWeek

	var (
		httpRequest workflow.HTTPRequest
		w           wrapped
	)
	err = json.Unmarshal(data, &httpRequest)
	if err == nil && len(httpRequest.Body) > 0 {
		err = json.Unmarshal(httpRequest.Body, &w)
		if err != nil {
			err = fmt.Errorf("unmarshaling HTTP request body into EveryWeek: %w", err)
			return
		}

		*e = EveryWeek(w)
		return
	}

	err = json.Unmarshal(data, &w)
	if err != nil {
		err = fmt.Errorf("unmarshaling bytes into EveryWeek: %w", err)
		return
	}

	*e = EveryWeek(w)
	return
}

func (e *EveryWeek) GetConfigObject() any {
	return new(EveryWeek)
}

func (e *EveryWeek) GetSampleList(ctx *trigger.TriggerDeps) (result []trigger.SampleData, err error) {
	config := ctx.GetConfigObject().(*EveryWeek)

	if config.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	tz, err := time.LoadLocation(config.Timezone)
	if err != nil {
		err = fmt.Errorf("parsing timezone %q: %w", e.Timezone, err)
		return
	}

	if len(config.WeekDays) == 0 {
		err = errors.New("no weekday selected")
		return
	}
	if len(config.WeekDays) > 7 {
		err = errors.New("too many weekdays selected")
		return
	}

	sort.Ints(config.WeekDays)

	seen := -1
	for i, v := range config.WeekDays {
		if v < 0 || v > 6 {
			err = fmt.Errorf("invalid weekday %d at index %d", v, i)
			return
		}
		if v == seen {
			err = fmt.Errorf("duplicated weekday %d at index %d and %d", v, i-1, i)
			return
		}

		seen = v
	}

	hour, min, second, err := trigger.ParseTimeInDay(config.Time)
	if err != nil {
		err = fmt.Errorf("parsing time of day: %w", err)
		return
	}

	var (
		now  = time.Now().In(tz)
		next time.Time
	)

	// We can safely assume the next execution time is in the following 7 days
OUTER:
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		next = time.Date(now.Year(), now.Month(), now.Day()+dayOffset, hour, min, second, 0, tz)

		if next.Before(now) {
			continue
		}

		nextWeekDay := next.Weekday()
		for _, wantWeekday := range config.WeekDays {
			if nextWeekDay == time.Weekday(wantWeekday) {
				break OUTER
			}
		}
	}

	result = []trigger.SampleData{newOutput(next)}
	return
}

func (e *EveryWeek) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("everyWeek"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(EveryWeek)
		},
		InputForm: spec.InputSchema,
	}
}

type everyBase struct {
	// timezone for cron expression
	Timezone string `json:"timezone"`
}

type everyOutput struct {
	// 2022-12-14T21:49:48+00:00
	DateTime string `json:"dateTime"`
	// 2022-12-14
	Date string `json:"date"`
	// 21:49:48
	Time string `json:"time"`
	// 3 for wednesday, one-based, Monday as 1
	DayOfWeek int `json:"dayOfWeek"`
	// Wednesday
	DayOfWeekName string `json:"dayOfWeekName"`
	// 2022
	DateYear int `json:"dateYear"`
	// 1-12, one-based
	DateMonth int `json:"dateMonth"`
	// 1-31, one-based
	DateDay int `json:"dateDay"`
	// 0-23, zero-based
	TimeHour int `json:"timeHour"`
	// 0-59, zero-based
	TimeMinute int `json:"timeMinute"`
	// 0-59, zero-based
	TimeSecond int `json:"timeSecond"`
}

func (e everyOutput) GetID() string {
	return e.DateTime
}

func (e everyOutput) GetVersion() string {
	return strconv.Itoa(int(time.Now().UnixMilli()))
}

func newOutput(t time.Time) (output everyOutput) {
	output = everyOutput{
		DateTime:   t.Format("2006-01-02T15:04:05-07:00"),
		Date:       t.Format("2006-01-02"),
		Time:       t.Format("15:04:05"),
		DayOfWeek:  int(t.Weekday()),
		DateYear:   t.Year(),
		DateMonth:  int(t.Month()),
		DateDay:    t.Day(),
		TimeHour:   t.Hour(),
		TimeMinute: t.Minute(),
		TimeSecond: t.Second(),
	}

	if output.DayOfWeek == 0 {
		// Go's Sunday is 0, and we need 7
		output.DayOfWeek = 7
	}
	switch output.DayOfWeek {
	case 1:
		output.DayOfWeekName = "Monday"
	case 2:
		output.DayOfWeekName = "Tuesday"
	case 3:
		output.DayOfWeekName = "Wednesday"
	case 4:
		output.DayOfWeekName = "Thursday"
	case 5:
		output.DayOfWeekName = "Friday"
	case 6:
		output.DayOfWeekName = "Saturday"
	case 7:
		output.DayOfWeekName = "Sunday"
	}

	return
}

func (e everyBase) Run(c *workflow.NodeContext) (output any, err error) {
	if e.Timezone == "" {
		err = errors.New("timezone is required")
		return
	}

	// uses the user-specified timezone
	tz, err := time.LoadLocation(e.Timezone)
	if err != nil {
		err = fmt.Errorf("parsing timezone %q: %w", e.Timezone, err)
		return
	}

	output = newOutput(time.Now().In(tz))
	return
}
