package daemon

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"linkernetworks.com/dcos-backend/autoscaling/metrics-collector/types"
)

const (
	prefix   = "{"
	suffix   = "}"
	sepComma = ","
	sepEqual = "="
	sepColon = "\""
)

func parseMetrics(rawMetrics types.RawMetrics) (metrics *types.Metrics, err error) {
	metrics = &types.Metrics{}
	for _, l := range rawMetrics.Lines {
		line, err := parseLine(l)
		if err != nil {
			log.Printf("parse line error: %v\n", err)
			continue
		}
		if line != nil {
			metrics.Lines = append(metrics.Lines, *line)
		}
	}
	return
}

// TODO Benchmark
// parse one line to struct line
// sample data:
// some_index{name="tom",age="10"} 3.14
// container_fs_inodes_total{device="/dev/mapper/docker-253:1-1179779-a25e55a7c63ff0a215055e118f6ee8ba1092c8692df2ce4e4dd510ff3e09afa9",id="/"} 1.0484736e+07
// container_fs_inodes_total{device="/dev/sda2",id="/"} 512000
func parseLine(data string) (line *types.Line, err error) {
	line = &types.Line{}

	lBracket := strings.Index(data, prefix)
	rBracket := strings.Index(data, suffix)

	// fmt.Println("=====DEBUG======")
	// fmt.Println(data[:])
	// fmt.Println(data[:lBracket])
	// fmt.Println(data[lBracket : rBracket+1])
	// fmt.Println(data[rBracket+1:])

	line.Index = data[:lBracket]

	m, err := parseFakeJSON(data[lBracket : rBracket+1])
	if err != nil {
		return
	}
	line.Map = m

	floatVar := strings.TrimSpace(data[rBracket+1:])
	f, err := strconv.ParseFloat(floatVar, 64)
	if err != nil {
		err = fmt.Errorf("parse error: convert %v to float64 error: %v", floatVar, err)
		return
	}
	line.FloatVar = f

	return
}

// TODO Benchmark
// parse data to a map
// do not allow space, tabs, enter
// do not allow comma, equal, quote in value
// sample data:
// {name="tom",age="10"}
func parseFakeJSON(data string) (m map[string]string, err error) {
	m = make(map[string]string)
	// data: {name="tom",age="10"}
	if !strings.HasPrefix(data, prefix) || !strings.HasSuffix(data, suffix) {
		err = fmt.Errorf("parse error: no prefix '%s' or suffix '%s'", prefix, suffix)
		return
	}
	// data: name="tom",age="10"
	data = strings.TrimPrefix(data, prefix)
	data = strings.TrimSuffix(data, suffix)
	// kvArr: [name="tom",age="10"]
	kvArr := strings.Split(data, sepComma)
	for _, kv := range kvArr {
		// kv: name="tom"
		arr := strings.Split(kv, sepEqual)
		// arr: [name,"tom"]
		if len(arr) != 2 {
			err = fmt.Errorf("parse error: invalid length %d, not 2", len(arr))
			return
		}
		// k: name
		// v: "tom"
		k := arr[0]
		v := arr[1]
		if !strings.HasPrefix(v, sepColon) || !strings.HasSuffix(v, sepColon) {
			err = fmt.Errorf("parse error: value '%s' not surrounded with '%s'", v, sepColon)
			return
		}
		// v: "tom"
		v = strings.TrimPrefix(v, sepColon)
		v = strings.TrimSuffix(v, sepColon)
		m[string(k)] = string(v)
	}
	return
}
