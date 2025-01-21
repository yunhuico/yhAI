package rule

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
)

// PromRule is strcut of a segment in prometheus rule file
// It looks like this:
//
// ALERT HighMemoryAlert
//   IF container_memory_usage_high_result > 1
//   FOR 30s
//   ANNOTATIONS {
//     summary = "High Memory usage alert for container",
//     description = "High Memory usage for linker container on {{$labels.image}} for container {{$labels.name}} (current value: {{$value}})",
//   }
type PromRule struct {
	AlertName   string
	Conditions  Conditions
	Duration    string
	Annotations map[string]string
}

// Marshal convert PromRule to bytes
func (r PromRule) Marshal() ([]byte, error) {
	if reflect.DeepEqual(r, PromRule{}) {
		return []byte{}, nil
	}

	// iterating map by dictionary-sorted key
	var keys []string
	for k, _ := range r.Annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var m string
	for _, key := range keys {
		m += fmt.Sprintf("    %s = \"%s\",\n", key, r.Annotations[key])
	}

	var buf bytes.Buffer
	buf.WriteString("ALERT " + r.AlertName + "\n")
	// buf.WriteString("  IF " + r.Index + " " + r.CompareSym + " " + fmt.Sprintf("%v", r.Threshold) + "\n")
	buf.WriteString("  IF " + r.Conditions.String() + "\n")
	buf.WriteString("  FOR " + r.Duration + "\n")
	buf.WriteString("  ANNOTATIONS " + "{\n")
	buf.WriteString(m)
	buf.WriteString("  }\n")
	return buf.Bytes(), nil
}
