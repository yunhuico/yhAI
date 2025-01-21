package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockScopeProvider map[string]any

func (m mockScopeProvider) GetScopeData() map[string]any {
	return m
}

type Author struct {
	ID       int
	Username string
}

var data = map[string]any{
	"Node": interface{}(map[string]any{
		"node1": map[string]any{
			"output": any(map[string]any{
				"object": map[string]any{
					"id":    1,
					"state": "published",
					"author": &Author{
						ID:       1,
						Username: "ultrafox",
					},
				},
			}),
		},
		"node2": map[string]any{
			"output": any(map[string]any{
				"object": map[string]any{
					"id":    float64(1006390),
					"float": 10237873.2389182,
				},
			}),
		},
	}),
}

func TestEngine_RenderTemplate(t *testing.T) {
	template := `{
	"foo": "{{ .node.trigger_node.output.object.title }}",
	"bar": {{ .node.trigger_node.output.object.id }},
	"baz": {
		"object": {
			"name": "{{ .node.trigger_node.output.object.author.name }}"
		}
	}
}`

	e := NewTemplateEngine(&mockScopeProvider{})
	_, err := e.RenderTemplate(template)
	assert.NoError(t, err)

	content2 := `{
	"foo": "{{ .node.trigger-node.output.object.title }}"
}`
	_, err = e.RenderTemplate(content2)
	assert.Error(t, err) // so node name should not contains '-'

	var mockData mockScopeProvider = data
	e = NewTemplateEngine(&mockData)
	content, err := e.RenderTemplate("{{ .Node.node1.output.object.author.ID }}")
	assert.NoError(t, err)
	assert.Equal(t, []byte("1"), content)
}

func TestEngine_JQWorkGreat(t *testing.T) {
	t.Run("test simple", func(t *testing.T) {
		mockData := map[string]any{
			"object_kind": "issue",
			"user": map[string]any{
				"name":     "ultrafox",
				"username": "Ultrafox",
			},
			"object_attributes": map[string]any{
				"id":    12434,
				"state": "opened",
			},
			"labels": []string{
				"DEV",
				"PROD",
			},
			"mergeRequests": []any{
				map[string]any{
					"projectId": 1,
					"name":      "Ultrafox Dev",
				},
				map[string]any{
					"projectId": 2,
					"name":      "Ultrafox Test",
				},
			},
			"tags": []any{
				[]int{1, 2, 3},
				[]int{4, 5, 6},
			},
		}
		e := NewTemplateEngineFromMap(mockData)
		v, err := e.RenderTemplate(`Hello, {{ jq ".user.username" }}`)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Ultrafox", string(v))

		v, err = e.RenderTemplate(`Hello, {{ jq ".user" | jq ".username" }}`)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Ultrafox", string(v))

		v, err = e.RenderTemplate(`{{ .labels | join "," }}`)
		assert.NoError(t, err)
		assert.Equal(t, "DEV,PROD", string(v))

		v, err = e.RenderTemplate(`{{ jq ".labels" | join "," }}`)
		assert.NoError(t, err)
		assert.Equal(t, "DEV,PROD", string(v))

		v, err = e.RenderTemplate(`{{ jq ".mergeRequests.[].projectId" | join "," }}`)
		assert.NoError(t, err)
		assert.Equal(t, "1,2", string(v))

		v, err = e.RenderTemplate(`{{ mod .object_attributes.id 2 }}`)
		assert.NoError(t, err)
		assert.Equal(t, "0", string(v))

		v, err = e.RenderTemplate(`All labels: {{ .labels }}
mergeRequests: {{ .mergeRequests.[].projectId }}`)
		assert.NoError(t, err)
		assert.Equal(t, "All labels: DEV,PROD\nmergeRequests: 1,2", string(v))

		v, err = e.RenderTemplate(`mergeRequests: {{ .mergeRequests }}`)
		assert.NoError(t, err)
		assert.Equal(t, `mergeRequests: {"name":"Ultrafox Dev","projectId":1},{"name":"Ultrafox Test","projectId":2}`, string(v))

		v, err = e.RenderTemplate(`{{ .tags }}`)
		assert.NoError(t, err)
		assert.Equal(t, `1,2,3,4,5,6`, string(v))
	})
}

func Test_empowerContent(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test no variable",
			args: args{
				content: "this is a content",
			},
			want: "this is a content",
		},
		{
			name: "test simple variable",
			args: args{
				content: "title: {{ .Node.id.output.title }}",
			},
			want: "title: {{ .Node.id.output.title | normalize }}",
		},
		{
			name: "test many complex variable",
			args: args{
				content: `labels: {{ .Node.id.output.labels.[].title }}
tags: {{ .Node.id.output.tags }}`,
			},
			want: `labels: {{ jq ".Node.id.output.labels.[].title" | normalize }}
tags: {{ .Node.id.output.tags | normalize }}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, empowerContent(tt.args.content), "empowerContent(%v)", tt.args.content)
		})
	}
}

var issueList = map[string]any{
	"issues": []any{
		map[string]any{
			"id":    1,
			"title": "issue1",
		},
		map[string]any{
			"id":    2,
			"title": "issue2",
		},
	},
}

func TestEngine_Evaluate(t *testing.T) {
	var mockData mockScopeProvider = issueList
	e := NewTemplateEngine(&mockData)
	titles, err := e.Evaluate(".issues.[].title")
	assert.NoError(t, err)
	assert.Equal(t, []any{"issue1", "issue2"}, titles)

	titles, err = e.Evaluate("[.issues.[].title]")
	assert.NoError(t, err)
	assert.Equal(t, []any{[]any{"issue1", "issue2"}}, titles)

	ids, err := e.Evaluate(".issues.[].id")
	assert.NoError(t, err)
	assert.Equal(t, []any{1, 2}, ids)

	id1, err := e.Evaluate(".issues.[0].id")
	assert.NoError(t, err)
	assert.Equal(t, 1, id1)

	// formal jq grammar.
	id1, err = e.Evaluate(".issues[0].id")
	assert.NoError(t, err)
	assert.Equal(t, 1, id1)

	ids, err = e.Evaluate(".issues.[]?.id")
	assert.NoError(t, err)
	assert.Equal(t, []any{1, 2}, ids)
}

// https://jihulab.com/ultrafox/ultrafox/-/issues/687
func TestEngine_RenderFloatInTemplate(t *testing.T) {
	var mockData mockScopeProvider = data
	e := NewTemplateEngine(&mockData)
	content, err := e.RenderTemplate(`{{ .Node.node2.output.object.id }}`)
	assert.NoError(t, err)
	assert.Equal(t, []byte("1006390"), content)

	content, err = e.RenderTemplate(`float string is: {{ .Node.node2.output.object.float }}`)
	assert.NoError(t, err)
	assert.Equal(t, []byte("float string is: 10237873.2389182"), content)
}
