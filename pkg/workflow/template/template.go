package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/araddon/dateparse"
	"github.com/itchyny/gojq"
	"github.com/spf13/cast"

	"github.com/Masterminds/sprig/v3"
	"github.com/caarlos0/duration"
	"github.com/itchyny/timefmt-go"
)

type Engine struct {
	scope ScopeDataProvider
}

func NewTemplateEngine(scope ScopeDataProvider) *Engine {
	return &Engine{
		scope: scope,
	}
}

func NewTemplateEngineFromMap(m map[string]any) *Engine {
	return &Engine{
		scope: mapScope(m),
	}
}

type ScopeDataProvider interface {
	GetScopeData() map[string]any
}

type mapScope map[string]any

func (m mapScope) GetScopeData() map[string]any {
	return m
}

// RenderTemplate using go template render the content.
func (e *Engine) RenderTemplate(content string) ([]byte, error) {
	content = empowerContent(content)

	tmpl, err := template.New("tmpl").Funcs(e.tmplFuncMap()).Parse(content)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, e.scope.GetScopeData())
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// regVariableReference match the variable reference in the template.
//
// test expressions:
// N {{}}
// N {{ }}
// N {{ .Node.id.output.id}}
// Y {{.Node.id.output.id}}
// Y {{ .Node.id.output.labels.[].id }}
// Y {{ .Node.id.output.labels.[].id }}
// N {{ .Node.id.output.labels.[].id | join "," }}
// Y {{ .Node.node1.output.assignees[].emails[]? }}
// Y {{ .Iter.loopItem.labels[]? }}
var regVariableReference = regexp.MustCompile(`\{\{\s*([^\{\}|\s]+)\s*\}\}`)

// empowerContent add some function for render more friendly content.
//
// if variable contains '[]', will add `jq`.
// all variable will add ` | normalize` to transform complex value to string
// instead of golang default toString.
//
// example: []string{"foo", "bar"}, expect render to `"foo,bar"`, but golang
// render to ["foo" "bar] as default.
func empowerContent(content string) string {
	return regVariableReference.ReplaceAllStringFunc(content, func(s string) string {
		variable := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "{{"), "}}"))
		if strings.Contains(variable, "[]") {
			variable = `jq "` + variable + `"`
		}

		return "{{ " + variable + " | normalize }}"
	})
}

func (e *Engine) tmplFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()
	// TODO: maybe have better implement
	funcMap["timeFormat"] = e.FormatTime
	funcMap["jq"] = e.jqFunc
	funcMap["normalize"] = e.normalizeFunc
	return funcMap
}

// FormatTime formats time as given format
// opts: sudDuration, fromTimezone,toTimezone
func (e *Engine) FormatTime(src interface{}, format string, opts ...string) (string, error) {
	switch v := src.(type) {
	case time.Time:
		if len(opts) < 1 {
			return timefmt.Format(v, format), nil
		}
		dr, err := duration.Parse(opts[0])
		if err != nil {
			return "", err
		}
		return timefmt.Format(v.Add(dr), format), nil
	case string:
		tm, err := dateparse.ParseAny(v)
		if err != nil {
			return "", err
		}
		return timefmt.Format(tm, format), nil
	default:
		return "", fmt.Errorf("unsupported type")
	}
}

func (e *Engine) normalizeFunc(src interface{}) (any, error) {
	if src == nil {
		return nil, nil
	}

	v := reflect.ValueOf(src)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	k := v.Kind()
	// when json.Unmarshal to a map[string]any, all integers will be converted to float64.
	// so try to convert to int preferentially.
	if k == reflect.Float64 || k == reflect.Float32 {
		var f float64
		if k == reflect.Float64 {
			f = src.(float64)
		} else {
			f = float64(src.(float32))
		}
		if f == math.Trunc(f) {
			// can be safely converted to an integer.
			return int(f), nil
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}

	if (reflect.Bool <= k && k <= reflect.Complex128) || k == reflect.String {
		return src, nil
	}

	if k == reflect.Map || k == reflect.Struct {
		jsonBytes, err := json.Marshal(src)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal map: %w", err)
		}
		return string(jsonBytes), nil
	}

	if k == reflect.Array || k == reflect.Slice {
		l := v.Len()
		b := make([]string, 0, l)
		for i := 0; i < l; i++ {
			value := v.Index(i).Interface()
			if value != nil {
				item, err := e.normalizeFunc(value)
				if err != nil {
					return nil, fmt.Errorf("item or slice or array transform to string: %w", err)
				}
				b = append(b, cast.ToString(item))
			}
		}
		return strings.Join(b, ","), nil
	}

	return src, nil
}

func (e *Engine) jqFunc(src interface{}, input ...interface{}) (any, error) {
	expression, ok := src.(string)
	if !ok {
		return nil, fmt.Errorf("jq src must be a string")
	}

	wantsList := regexWantsListExpression.MatchString(expression)

	expr, err := e.CompileJq(expression)
	if err != nil {
		return nil, err
	}

	if len(input) == 0 {
		// use root context
		result, err := e.calcJq(expr, wantsList)
		if err != nil {
			return nil, err
		}
		return result, err
	}

	return e.calcJqWithInput(expr, input[0], wantsList)
}

func (*Engine) CompileJq(src string) (*gojq.Code, error) {
	query, err := gojq.Parse(src)
	if err != nil {
		return nil, err
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}
	return code, nil
}

func (e *Engine) calcJq(expr *gojq.Code, wantsList bool) (any, error) {
	return e.calcJqWithInput(expr, e.scope.GetScopeData(), wantsList)
}

func (e *Engine) calcJqWithInput(expr *gojq.Code, input any, wantsList bool) (result any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	iter := expr.RunWithContext(ctx, input)

	if wantsList {
		result = []any{}
	}

	tmpResult := []any{}

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok = v.(error); ok {
			return
		}
		tmpResult = append(tmpResult, v)
	}

	if len(tmpResult) == 0 {
		return
	}

	if wantsList {
		result = tmpResult
	} else {
		result = tmpResult[0]
	}

	return
}

// check if an expression contains "[]?" or "[]"
var regexWantsListExpression = regexp.MustCompile(`\.?\[\]`)

func (e *Engine) Evaluate(expr string) (any, error) {
	code, err := e.CompileJq(expr)
	if err != nil {
		return nil, fmt.Errorf("evaluate failed: %w", err)
	}

	wantsList := regexWantsListExpression.MatchString(expr)

	v, err := e.calcJq(code, wantsList)
	if err != nil {
		return nil, fmt.Errorf("evaluate failed: %w", err)
	}

	return v, nil
}
