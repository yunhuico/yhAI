package validate

type (
	SwitchLogicNode struct {
		Paths []Path `json:"paths"`
	}
	Path struct {
		Name       string           `json:"name"`
		Conditions []ConditionGroup `json:"conditions"`
		Transition string           `json:"transition"`
		IsDefault  bool             `json:"isDefault"`
	}

	ConditionGroup []Condition

	Condition struct {
		Left      string    `json:"left"`
		Operation Operation `json:"operation"`
		Right     string    `json:"right"`
	}

	Operation string

	LoopFromListNode struct {
		InputCollection string `json:"inputCollection"`
		Transition      string `json:"transition"`
	}
)

const (
	ForeachClass       = "ultrafox/foreach#loopFromList"
	SwitchClass        = "ultrafox/logic#switch"
	CronTriggerClass   = "ultrafox/schedule#cron"
	CustomWebhookClass = "ultrafox/webhook#receiveData"
)

const (
	EqualsOperation                Operation = "equals"
	NotEqualsOperation             Operation = "not_equals"
	ContainsOperation              Operation = "contains"
	ContainsLowercasedOperation    Operation = "contains_lowercased"
	NotContainsOperation           Operation = "not_contains"
	NotContainsLowerCasedOperation Operation = "not_contains_lowercased"
	StringStartWithOperation       Operation = "strings.start_with"
	StringEndWithOperation         Operation = "strings.end_with"
	EmptyOperation                 Operation = "empty"
	NotEmptyOperation              Operation = "not_empty"
	GreaterThan                    Operation = "greater"
	LessThan                       Operation = "less"

	TimeBeforeOperation  Operation = "time.before"
	TimeAfterOperation   Operation = "time.after"
	TimeDayAgoOperation  Operation = "time.day_ago"
	TimeHourAgoOperation Operation = "time.hour_ago"
	TimeWeekAgoOperation Operation = "time.week_ago"
)
