package types

import (
	"fmt"
	"sort"
	"strings"
)

// RawMetrics
type RawMetrics struct {
	Lines []string
}

type Metrics struct {
	Lines []Line
}

type Line struct {
	Index    string
	Map      map[string]string
	FloatVar float64
}

func (p *Line) String() string {
	if p != nil {
		var mapStr string
		if p.Map != nil {
			// sort key in alphabetical
			kArr := make([]string, len(p.Map))
			i := 0
			for k, _ := range p.Map {
				kArr[i] = k
				i++
			}
			sort.Strings(kArr)

			for _, k := range kArr {
				mapStr += fmt.Sprintf("%s=\"%s\",", k, p.Map[k])
			}
			mapStr = fmt.Sprintf("{%s}", strings.TrimRight(mapStr, ","))
		} else {
			mapStr = "{}"
		}

		return fmt.Sprintf("%s%s %e\n", p.Index, mapStr, p.FloatVar)
	}
	return ""
}
