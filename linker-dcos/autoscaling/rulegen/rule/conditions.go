package rule

import "fmt"

const (
	relationAnd = "AND"
	relationOr  = "OR"
)

type Condition struct {
	Index      string
	CompareSym string
	Threshold  float32
}

type Conditions struct {
	conditions []Condition
	relations  []string
}

// NewConditions create Conditions from a existing Condition
func NewConditions(c Condition) *Conditions {
	return &Conditions{
		conditions: []Condition{c},
	}
}

func (cs *Conditions) And(c Condition) *Conditions {
	if cs == nil {
		return cs
	}
	cs.conditions = append(cs.conditions, c)
	cs.relations = append(cs.relations, relationAnd)
	return cs
}

func (cs *Conditions) Or(c Condition) *Conditions {
	if cs == nil {
		return cs
	}
	cs.conditions = append(cs.conditions, c)
	cs.relations = append(cs.relations, relationOr)
	return cs
}

func (cs *Conditions) String() string {
	if cs == nil || len(cs.conditions) == 0 {
		return ""
	}
	// example: container_memory_usage_high_result > 1
	if len(cs.conditions) == 1 {
		c := cs.conditions[0]
		return fmt.Sprintf("%s %s %v", c.Index, c.CompareSym, c.Threshold)
	}
	// example: (container_memory_usage_low_result < 1 AND container_memory_usage_low_result > 0)
	var buf string
	buf += "("
	for i, c := range cs.conditions {
		buf += fmt.Sprintf("%s %s %v", c.Index, c.CompareSym, c.Threshold)
		if i < len(cs.relations) {
			buf += fmt.Sprintf(" %s ", cs.relations[i])
		}
	}
	buf += ")"
	return buf
}
