package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryFieldSelectReq_Normalize(t *testing.T) {
	t.Run("class is empty", func(t *testing.T) {
		req := &QueryFieldSelectReq{}
		err := req.Normalize()
		assert.NoError(t, err)
	})

	t.Run("class don't contain query fields", func(t *testing.T) {
		req := &QueryFieldSelectReq{
			Class: "foo",
		}
		err := req.Normalize()
		assert.NoError(t, err)
		assert.Equal(t, "foo", req.Class)
	})

	t.Run("normal class, join pagination to InputFields", func(t *testing.T) {
		req := &QueryFieldSelectReq{
			Class:      "foo",
			Page:       1,
			PerPage:    1,
			Search:     "search",
			NextCursor: "nextCursor",
		}
		err := req.Normalize()
		assert.NoError(t, err)
		assert.Equal(t, "foo", req.Class)
		assert.Len(t, req.InputFields, 4)
		assert.Equal(t, 1, req.InputFields["page"])
		assert.Equal(t, 1, req.InputFields["perPage"])
		assert.Equal(t, "search", req.InputFields["search"])
		assert.Equal(t, "nextCursor", req.InputFields["nextCursor"])
	})

	t.Run("class contains one query field", func(t *testing.T) {
		req := &QueryFieldSelectReq{
			Class: "foo?projectId2=projectId1",
			InputFields: map[string]any{
				"projectId1": 1,
				"projectId2": 2,
			},
		}
		err := req.Normalize()
		assert.NoError(t, err)
		assert.Equal(t, "foo", req.Class)
		assert.Equal(t, 2, req.InputFields["projectId1"])
		assert.Len(t, req.InputFields, 1+4)
	})

	t.Run("class contains two query field", func(t *testing.T) {
		req := &QueryFieldSelectReq{
			Class: "foo?projectId2=projectId1&issueId2=issueId1",
			InputFields: map[string]any{
				"projectId2": 2,
				"issueId2":   10,
			},
		}
		err := req.Normalize()
		assert.NoError(t, err)
		assert.Equal(t, "foo", req.Class)
		assert.Equal(t, 2, req.InputFields["projectId1"])
		assert.Equal(t, 10, req.InputFields["issueId1"])
		assert.Len(t, req.InputFields, 2+4)
	})

	t.Run("class contains empty query field, return error", func(t *testing.T) {
		req := &QueryFieldSelectReq{
			Class: "foo?projectId2=projectId1&issueId2",
			InputFields: map[string]any{
				"projectId2": 2,
				"issueId2":   10,
			},
		}
		err := req.Normalize()
		assert.Error(t, err)
	})
}
