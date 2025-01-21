package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDag(t *testing.T) {
	t.Run("test cycle", func(t *testing.T) {
		d := NewDag()
		err := d.Build()
		assert.NoError(t, err)
	})

	t.Run("test cycle", func(t *testing.T) {
		d := NewDag()
		d.AddEdge("a", "b")
		d.AddEdge("b", "c")
		d.AddEdge("c", "d")
		d.AddEdge("d", "b")

		err := d.Build()
		assert.Error(t, err)
		t.Log(err)
		assert.ErrorIs(t, err, ErrCycle)
	})

	t.Run("test full cycle", func(t *testing.T) {
		d := NewDag()
		d.AddEdge("a", "b")
		d.AddEdge("b", "c")
		d.AddEdge("c", "d")
		d.AddEdge("d", "a")

		err := d.Build()
		assert.Error(t, err)
		t.Log(err)
	})

	t.Run("no cycle", func(t *testing.T) {
		d := NewDag()
		d.AddEdge("a", "b")
		d.AddEdge("b", "c")
		d.AddEdge("c", "d")
		err := d.Build()
		assert.NoError(t, err)
	})
}
