package schema

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/log"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils"
)

func TestBuildOutput(t *testing.T) {
	log.Init("go-test", log.DebugLevel)
	ctx := context.Background()

	t.Run("test a basic example", func(t *testing.T) {
		// just one field.
		baseField := adapter.BaseField{
			Key:   "projectId",
			Label: "Project ID",
			Type:  adapter.StringFieldType,
		}
		schemaField := &adapter.OutputField{
			BaseField: baseField,
			Where:     []adapter.Where{adapter.WhereTemplate},
		}

		schemaFields := adapter.OutputFields{ // nolint: staticcheck
			schemaField,
		}
		nodeID, _ := utils.NanoID()
		projectID, _ := utils.NanoID()
		fields := BuildOutput(ctx, schemaFields, nodeID, map[string]any{ // nolint: staticcheck
			"projectId": projectID,
		})
		assert.Len(t, fields, 1)
		assert.Equal(t, OutputField{ // nolint: staticcheck
			BaseField: baseField,
			Where:     []adapter.Where{adapter.WhereTemplate},
			AsStr:     projectID,
			Reference: fmt.Sprintf(".Node.%s.output.projectId", nodeID),
		}, fields[0])
	})

	t.Run("test []string field", func(t *testing.T) {
		schemaField := &adapter.OutputField{
			BaseField: adapter.BaseField{
				Key:   "labels",
				Label: "All labels",
				Type:  adapter.ListFieldType,
			},
			Child: &adapter.OutputField{
				BaseField: adapter.BaseField{
					Key:   "",
					Label: "Label",
					Type:  adapter.StringFieldType,
				},
				Where: []adapter.Where{
					adapter.WhereTemplate,
				},
			},
			Where: []adapter.Where{
				adapter.WhereTemplate,
				adapter.WhereForeach,
			},
		}

		schemaFields := adapter.OutputFields{ // nolint: staticcheck
			schemaField,
		}
		nodeID, _ := utils.NanoID()
		fields := BuildOutput(ctx, schemaFields, nodeID, map[string]any{
			"labels": []string{
				"DEV",
				"PROD",
			},
		})
		assert.Len(t, fields, 1)
		assert.Len(t, fields[0].Where, 2)
		expectField := OutputField{
			BaseField: adapter.BaseField{
				Key:   "labels",
				Label: "All labels",
				Type:  adapter.ListFieldType,
			},
			ChildType: adapter.StringFieldType,
			Where: []adapter.Where{
				adapter.WhereTemplate,
				adapter.WhereForeach,
			},
			AsStr:     "DEV,PROD",
			Reference: fmt.Sprintf(`.Node.%s.output.labels`, nodeID),
			Fields: []OutputField{
				{
					BaseField: adapter.BaseField{
						Key:   "",
						Label: "Label",
						Type:  adapter.StringFieldType,
					},
					Where: []adapter.Where{
						adapter.WhereTemplate,
					},
					AsStr:     "DEV",
					Reference: `.Iter.loopItem`,
				},
			},
		}
		assert.Equal(t, expectField, fields[0])
	})

	t.Run("test []label field", func(t *testing.T) {
		schemaFields := adapter.OutputFields{ // nolint: staticcheck
			{
				BaseField: adapter.BaseField{
					Key:   "",
					Label: "All Labels",
					Type:  "list",
				},
				Child: &adapter.OutputField{
					BaseField: adapter.BaseField{
						Type: adapter.StructFieldType,
					},
					Fields: []*adapter.OutputField{
						{
							BaseField: adapter.BaseField{
								Key:   "id",
								Label: "Label ID",
								Type:  adapter.IntFieldType,
							},
							Where: []adapter.Where{
								adapter.WhereTemplate,
							},
						},
						{
							BaseField: adapter.BaseField{
								Key:   "title",
								Label: "Label Title",
								Type:  adapter.StringFieldType,
							},
							Where: []adapter.Where{
								adapter.WhereTemplate,
							},
						},
					},
				},
				Where: []adapter.Where{
					adapter.WhereForeach,
				},
			},
			{
				BaseField: adapter.BaseField{
					Key:   "[].id",
					Label: "All Label ID",
					Type:  "list",
				},
				Child: &adapter.OutputField{
					BaseField: adapter.BaseField{
						Type: adapter.IntFieldType,
					},
				},
				Where: []adapter.Where{
					adapter.WhereTemplate,
				},
			},
			{
				BaseField: adapter.BaseField{
					Key:   "[].title",
					Label: "All Label Title",
					Type:  "list",
				},
				Child: &adapter.OutputField{
					BaseField: adapter.BaseField{
						Type: adapter.StringFieldType,
					},
				},
				Where: []adapter.Where{
					adapter.WhereTemplate,
				},
			},
		}

		nodeID, _ := utils.NanoID()
		fields := BuildOutput(ctx, schemaFields, nodeID, []any{ // nolint: staticcheck
			map[string]any{
				"id":    1,
				"title": "Ultrafox",
			},
			map[string]any{
				"id":    2,
				"title": "Ultrafox Dev",
			},
		})

		assert.Len(t, fields, 3)
		expectField := OutputField{ // nolint: staticcheck
			BaseField: adapter.BaseField{
				Key:   "",
				Label: "All Labels",
				Type:  adapter.ListFieldType,
			},
			ChildType: adapter.StructFieldType,
			Where:     []adapter.Where{adapter.WhereForeach},
			Reference: fmt.Sprintf(`.Node.%s.output`, nodeID),
			Fields: []OutputField{ // nolint: staticcheck
				{
					BaseField: adapter.BaseField{
						Key:   "loopItem.id",
						Label: "Label ID",
						Type:  adapter.IntFieldType,
					},
					Where:     []adapter.Where{adapter.WhereTemplate},
					AsStr:     "1",
					Reference: ".Iter.loopItem.id",
				},
				{
					BaseField: adapter.BaseField{
						Key:   "loopItem.title",
						Label: "Label Title",
						Type:  adapter.StringFieldType,
					},
					Where:     []adapter.Where{adapter.WhereTemplate},
					AsStr:     "Ultrafox",
					Reference: ".Iter.loopItem.title",
				},
			},
		}
		assert.Equal(t, expectField, fields[0])

		expectField = OutputField{ // nolint: staticcheck
			BaseField: adapter.BaseField{
				Key:   "[].title",
				Label: "All Label Title",
				Type:  adapter.ListFieldType,
			},
			ChildType: adapter.StringFieldType,
			Where:     []adapter.Where{adapter.WhereTemplate},
			AsStr:     "Ultrafox,Ultrafox Dev",
			Reference: fmt.Sprintf(".Node.%s.output.[].title", nodeID),
			Fields:    nil,
		}
		assert.Equal(t, expectField, fields[2])
	})
}
