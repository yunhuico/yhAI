package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMasker(t *testing.T) {
	masker := NewMasker(InputFormFields{
		{
			BaseField: BaseField{
				Key: "k1",
			},
			AdvancedField: AdvancedField{
				Encrypted: true,
			},
		},
		{
			BaseField: BaseField{
				Key: "k2",
			},
			AdvancedField: AdvancedField{
				Encrypted: true,
			},
		},
		{
			BaseField: BaseField{
				Key: "k3",
			},
			AdvancedField: AdvancedField{
				Encrypted: true,
			},
		},
		{
			BaseField: BaseField{
				Key: "k4",
			},
			AdvancedField: AdvancedField{
				Encrypted: true,
			},
		},
	})
	fields := map[string]string{
		"k1": "password",
		"k2": "!!!Ultrafox!!!",
		"k3": "abc",
		"k4": "abcdef",
	}
	fields2 := masker.Mask(fields)
	assert.Equal(t, "********", fields2["k1"])
	assert.Equal(t, "!!!********!!!", fields2["k2"])
	assert.Equal(t, "***", fields2["k3"])
	assert.Equal(t, "******", fields2["k4"])

	newK2Value := "!!Ultrafox"
	fields2["k2"] = newK2Value
	fields3, changed := masker.MergeChangedValue(fields, fields2)
	assert.True(t, changed)
	assert.Equal(t, "password", fields3["k1"])
	assert.Equal(t, newK2Value, fields3["k2"])
	assert.Equal(t, "abc", fields3["k3"])
	assert.Equal(t, "abcdef", fields3["k4"])

	fields4 := masker.Mask(fields)
	fields5, changed := masker.MergeChangedValue(fields, fields4)
	assert.False(t, changed)
	assert.Equal(t, fields, fields5)
}

// https://jihulab.com/ultrafox/ultrafox/-/work_items/721?iid_path=true
//
// # Orignal data
// token: "this_is_a_token"
//
// # Frontend data
// token: "thi*********ken"
//
// # User edited
// token: "thi---------ken"
//
// # : In updating logic
// ❌ token: "this_is_a_token"
// ✅ token: "thi---------ken"
func TestMasker_Bug(t *testing.T) {
	masker := NewMasker(InputFormFields{
		{
			BaseField: BaseField{
				Key: "k1",
			},
			AdvancedField: AdvancedField{
				Encrypted: true,
			},
		},
	})
	fields := map[string]string{
		"k1": "this-is-ultrafox!!!",
	}
	frontendFields := masker.Mask(fields)
	maskK1 := "thi*************!!!"
	assert.Equal(t, maskK1, frontendFields["k1"])

	newK1 := "thi-------------!!!"
	assert.Equal(t, len(maskK1), len(newK1))
	result, changed := masker.MergeChangedValue(fields, map[string]string{
		"k1": newK1,
	})
	assert.True(t, changed)
	assert.Equal(t, newK1, result["k1"])
}
