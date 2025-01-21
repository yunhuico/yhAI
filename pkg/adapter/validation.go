package adapter

import (
	"fmt"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/spf13/cast"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/trans"
)

type ruleWrapper struct {
	key       string
	rules     []validation.Rule
	condition *ruleCondition
}

func (r ruleWrapper) getRule(inputFields map[string]any) (rule *validation.KeyRules) {
	if r.condition == nil {
		return validation.Key(r.key, r.rules...)
	}
	needValidate := r.condition.calc(inputFields)
	if !needValidate {
		return nil
	}
	return validation.Key(r.key, r.rules...)
}

type ruleCondition struct {
	display Display
}

func (c ruleCondition) calc(inputFields map[string]any) bool {
	for _, group := range c.display {
		groupPass := true

		for _, condition := range group {
			var (
				actual = cast.ToString(inputFields[condition.Key])
				expect = condition.Value
			)

			if condition.Operation == displayConditionEquals {
				expect = cast.ToString(expect)
				if actual != expect {
					groupPass = false
					break
				}
			} else if condition.Operation == displayConditionIn {
				if !in(actual, expect) {
					groupPass = false
					break
				}
			} else {
				// if operation not supported, can't pass.
				groupPass = false
				break
			}
		}

		if groupPass {
			return true
		}
	}

	return false
}

func in(actual any, expect any) bool {
	expectArray, err := trans.ToStringSlice(expect)
	if err != nil {
		return false
	}
	for _, expectItem := range expectArray {
		if expectItem == actual { // group-condition pass!
			return true
		}
	}
	return false
}

type basicValidator struct {
	rules []ruleWrapper
}

func (v *basicValidator) validate(inputFields map[string]any) (err error) {
	if v == nil {
		return
	}

	// if inputFields is nil, validation will pass in any case!
	// check test case "required field not provided"
	if inputFields == nil {
		inputFields = map[string]any{}
	}

	mapRule := validation.Map(v.getRules(inputFields)...).AllowExtraKeys()
	err = validation.Validate(inputFields, mapRule)
	if err != nil {
		err = fmt.Errorf("validating input fields: %w", err)
		return
	}
	return
}

func (v *basicValidator) getRules(inputFields map[string]any) (krs []*validation.KeyRules) {
	for _, rule := range v.rules {
		kr := rule.getRule(inputFields)
		if kr == nil {
			continue
		}
		krs = append(krs, kr)
	}
	return
}

func buildBasicValidator(fields InputFormFields) *basicValidator {
	if len(fields) == 0 {
		return nil
	}

	var ruleWrappers []ruleWrapper
	for _, field := range fields {
		var condition *ruleCondition
		if field.UI != nil && field.UI.Display != nil { // display only available in the outer fields.
			condition = &ruleCondition{display: *field.UI.Display}
		}
		rules := getRulesRecursively(field)
		if len(rules) == 0 {
			continue
		}
		ruleWrappers = append(ruleWrappers, ruleWrapper{
			key:       field.Key,
			rules:     rules,
			condition: condition,
		})
	}
	return &basicValidator{rules: ruleWrappers}
}

func getRulesRecursively(field *InputFormField) (rules []validation.Rule) {
	if field.Required {
		if field.Type == BoolFieldType ||
			field.Type == IntFieldType ||
			field.Type == FloatFieldType {
			rules = append(rules, validation.NotNil)
		} else {
			rules = append(rules, validation.Required)
		}
	}

	switch field.Type {
	case ListFieldType:
		subRules := getRulesRecursively(field.Child)
		if len(subRules) == 0 {
			return
		}
		rules = append(rules, validation.Each(subRules...))
		return
	case StructFieldType:
		if field == nil {
			return
		}
		for _, subField := range field.Fields {
			subRules := getRulesRecursively(subField)
			if len(subRules) == 0 {
				continue
			}

			subKeyRules := validation.Key(subField.Key, subRules...)
			if !subField.Required {
				subKeyRules = subKeyRules.Optional()
			}
			rules = append(rules, validation.Map(subKeyRules).AllowExtraKeys())
		}
		return
	default:
	}

	return
}
