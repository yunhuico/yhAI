package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type author struct {
	ID       int
	Username string
}

func Test_contextScope_evaluate(t *testing.T) {
	s := &contextScopeImpl{
		Node: map[string]any{
			"node1": map[string]any{
				"output": any(map[string]any{
					"object": map[string]any{
						"id":    1,
						"state": "published",
						"author": &author{
							ID:       1,
							Username: "ultrafox",
						},
					},
				}),
			},
		},
	}
	v, err := s.evaluate(`.Node.node1.output.object.id`)
	assert.NoError(t, err)
	assert.Equal(t, 1, v)

	v, err = s.evaluate(`.Node.node1.output.object.id == 1`)
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	v, err = s.evaluate(`.Node.node1.output.object.id > 1`)
	assert.NoError(t, err)
	assert.Equal(t, false, v)

	v, err = s.evaluate(".Node.node1.output.object.state == \"published\"")
	assert.NoError(t, err)
	assert.Equal(t, true, v)

	_, err = s.evaluate(".Node.node1.output.object.author.ID == 1")
	assert.Error(t, err) // Error method: invalid value: &{1 ultrafox}
}
