package set

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	set := make(Set[int])
	set.Add(1, 2, 3)
	assert.Len(t, set, 3)
	assert.True(t, set.Delete(1))
	assert.Len(t, set, 2)
	assert.False(t, set.Delete(1))
	set.Add(2)
	assert.Equal(t, 2, set.Len())
	assert.True(t, set.Has(2))

	set.Foreach(func(i int) {
	})

	set2 := set.Copy()
	set2.Add(999)
	assert.NotEqual(t, set.Len(), set2.Len())
	actualAll := set2.All()
	sort.Ints(actualAll)
	assert.Equal(t, []int{2, 3, 999}, actualAll)
}
