package schema

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow/template"
)

// DEPRECATED: will be removed.
type OutputField struct {
	adapter.BaseField

	Where []adapter.Where `json:"where"`

	// ChildType define the type of list item.
	ChildType adapter.FieldType `json:"childType,omitempty"`

	// AsStr display the value as string.
	AsStr string `json:"asStr,omitempty"`

	// Fields if this field use by foreach input, should define Fields for iteration.
	Fields []OutputField `json:"fields,omitempty"`

	// Warning for developers, provide some error message.
	Warning string `json:"warning,omitempty"`

	// Reference the template content will use the reference.
	Reference string `json:"reference"`
}

const (
	loopIterationKey       = "loopIteration"
	loopTotalIterationsKey = "loopTotalIterations"
	loopIterationIsLast    = "loopIterationIsLast"
)

func isForeachFixedOutputKey(key string) bool {
	return key == loopIterationKey || key == loopTotalIterationsKey || key == loopIterationIsLast
}

// DEPRECATED: will be removed.
func BuildForeachOutput(ctx context.Context, schemaFields adapter.OutputFields, value any) (fields []OutputField) {
	contextData := map[string]any{
		"Iter": value,
	}

	templateEngine := template.NewTemplateEngineFromMap(contextData)
	for _, schemaField := range schemaFields {
		if len(schemaField.Where) == 0 {
			continue
		}
		field := OutputField{
			BaseField: schemaField.BaseField,
			Where:     schemaField.Where,
		}
		var reference string
		if isForeachFixedOutputKey(schemaField.Key) {
			reference = ".Iter." + schemaField.Key
		} else {
			reference = field.buildIterReference()
			field.Key = "loopItem." + field.Key
		}
		v, err := templateEngine.RenderTemplate("{{ " + reference + " }}")
		if err != nil {
			field.Warning = err.Error()
		} else {
			field.AsStr = string(v)
		}
		field.Reference = reference
		fields = append(fields, field)
	}
	return fields
}

// DEPRECATED: will be removed.
func BuildOutput(ctx context.Context, schemaFields adapter.OutputFields, nodeID string, value any) (fields []OutputField) {
	if len(schemaFields) == 0 {
		return
	}

	contextData := map[string]any{
		"Node": map[string]any{
			nodeID: map[string]any{
				"output": value,
			},
		},
	}
	return buildFields(ctx, contextData, schemaFields, nodeID, false)
}

// DEPRECATED: will be removed.
func buildFields(ctx context.Context, contextData map[string]any, schemaFields adapter.OutputFields, nodeID string, inForeach bool) (fields []OutputField) {
	templateEngine := template.NewTemplateEngineFromMap(contextData)

	for _, schemaField := range schemaFields {
		if len(schemaField.Where) == 0 {
			continue
		}

		field := OutputField{
			BaseField: schemaField.BaseField,
			Where:     schemaField.Where,
		}

		if schemaField.Type == adapter.ListFieldType {
			field.ChildType = schemaField.Child.Type
		}

		for _, where := range schemaField.Where {
			var reference string
			if inForeach {
				reference = field.buildIterReference()
			} else {
				reference = field.buildNodeReference(nodeID)
			}
			field.Reference = reference

			if where == adapter.WhereTemplate {
				v, err := templateEngine.RenderTemplate("{{ " + reference + " }}")
				if err != nil {
					field.Warning = err.Error()
				} else {
					field.AsStr = string(v)
				}
			} else if where == adapter.WhereForeach {
				v, err := templateEngine.Evaluate(reference)
				if err != nil {
					field.Warning = err.Error()
					continue
				}
				// expect value is slice.
				items := reflect.ValueOf(v)
				if items.Kind() != reflect.Slice {
					field.Warning = fmt.Sprintf("%s is not array", schemaField.Label)
					continue
				}

				if items.Len() == 0 {
					field.Warning = fmt.Sprintf("%s is empty", schemaField.Label)
					continue
				}

				contextData["Iter"] = map[string]any{
					"loopItem":            items.Index(0).Interface(),
					"loopTotalIterations": items.Len(),
					"loopIteration":       1,
					"loopIterationIsLast": items.Len() == 1,
				}

				if schemaField.Child.Type == adapter.StructFieldType {
					field.Fields = buildFields(ctx, contextData, schemaField.Child.Fields, nodeID, true)
				} else {
					if schemaField.Child == nil {
						log.For(ctx).Error("foreach child not defined")
						continue
					}

					childField := OutputField{
						BaseField: schemaField.Child.BaseField,
						Where:     schemaField.Child.Where,
					}

					// []{string | bool ...} child field cannot foreach again.
					// hardcode adapter.WhereTemplate
					childReference := childField.buildIterReference()
					childField.Reference = childReference

					templateEngine := template.NewTemplateEngineFromMap(contextData)
					v, err := templateEngine.RenderTemplate("{{ " + childReference + " }}")
					if err != nil {
						childField.Warning = err.Error()
					} else {
						childField.AsStr = string(v)
					}

					// the child where must be template.
					field.Fields = []OutputField{childField}
				}
			} else {
				log.For(ctx).Error("unknown where", log.String("where", string(where)))
				continue
			}
		}

		// if in foreach, configure like following:
		// `id`, `iid`, `title`
		// so key will become `loopItem.id`, `loopItem.iid`, `loopItem.title`
		if inForeach {
			field.Key = "loopItem." + field.Key
		}
		fields = append(fields, field)
	}
	return
}

func (f OutputField) buildIterReference() (result string) {
	return f.buildReferenceByPrefix(".Iter.loopItem")
}

func (f OutputField) buildNodeReference(nodeID string) (result string) {
	nodeIDPrefix := ".Node." + nodeID + ".output"
	return f.buildReferenceByPrefix(nodeIDPrefix)
}

func (f OutputField) buildReferenceByPrefix(prefix string) (result string) {
	if f.Key == "" {
		result = prefix
	} else if strings.Contains(f.Key, "$$") {
		result = strings.Replace(f.Key, "$$", prefix, 1)
	} else {
		result = prefix + "." + f.Key // add nodeID prefix first.
	}
	return
}
