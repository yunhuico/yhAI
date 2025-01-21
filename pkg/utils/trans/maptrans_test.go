package trans

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructToMap(t *testing.T) {
	type args struct {
		input any
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "test struct to map failed",
			args: args{
				12,
			},
			wantErr: true,
		},
		{
			name: "test originally map",
			args: args{
				input: map[string]any{
					"foo": "bar",
				},
			},
			wantErr: false,
			want: map[string]any{
				"foo": "bar",
			},
		},
		{
			name: "test struct to map success",
			args: args{
				input: &struct {
					Name string `json:"name"`
					Age  int    `json:"age"`
				}{
					Name: "test",
					Age:  10,
				},
			},
			want: map[string]any{
				"name": "test",
				"age":  10,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StructToMap(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StructToMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StructToMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapToStruct(t *testing.T) {
	t.Run("test obj is not a pointer", func(t *testing.T) {
		err := MapToStruct(map[string]any{
			"foo": "bar",
		}, nil)
		assert.Error(t, err)
	})

	t.Run("test decode error", func(t *testing.T) {
		type person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		p := &person{}
		err := MapToStruct(map[string]any{
			"name": "test",
			"age":  "10",
		}, p)
		assert.Error(t, err)
	})

	t.Run("pass", func(t *testing.T) {
		type person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		p := &person{}
		err := MapToStruct(map[string]any{
			"name": "ultrafox",
			"age":  1,
		}, p)
		assert.NoError(t, err)
		assert.Equal(t, p.Age, 1)
		assert.Equal(t, p.Name, "ultrafox")
	})
}

type foo struct {
	Foo       string `json:"foo"`
	Bar       string
	Int       int
	Hidden    string `json:"-"`
	inner     string
	Object    person
	ObjectPtr *person `json:"objectPtr"`
	Empty     string  `json:"empty,omitempty"`
}

type person struct {
	Name string `json:"name"`
}

func TestTransformToNodeData(t *testing.T) {
	m := map[string]any{
		"foo": "bar",
		"bar": "baz",
	}
	got, err := TransformToNodeData(m)
	assert.NoError(t, err)
	assert.Equal(t, m, got)

	got, err = TransformToNodeData(&m)
	assert.NoError(t, err)
	assert.Equal(t, m, got)

	foo := &foo{
		Foo:    "bar",
		Bar:    "",
		Int:    0,
		Hidden: "hidden",
		inner:  "inner",
		Object: person{
			Name: "ultrafox",
		},
		ObjectPtr: &person{
			Name: "ultrafox",
		},
		Empty: "",
	}
	out1, err := TransformToNodeData(foo)
	assert.NoError(t, err)
	out1Map := out1.(map[string]any)

	out2, err := TransformToNodeData(*foo)
	assert.NoError(t, err)
	assert.NotEmpty(t, out1)
	assert.Equal(t, out1, out2)
	assert.Equal(t, "bar", out1Map["foo"])
	assert.Equal(t, "", out1Map["Bar"])
	assert.Equal(t, float64(0), out1Map["Int"])
	assert.NotContains(t, out1Map, "Hidden")
	assert.NotContains(t, out1Map, "inner")
	assert.Contains(t, out1Map, "Object")
	assert.Contains(t, out1Map, "objectPtr")
	assert.NotContains(t, out1Map, "empty")
}

func TestTransformToNodeData_Empty(t *testing.T) {
	emptyBytes := []byte(``)
	got, err := TransformToNodeData(emptyBytes)
	assert.NoError(t, err)
	assert.Equal(t, nil, got)
}
