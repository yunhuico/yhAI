package workflow

import (
	"context"
	"fmt"
	"testing"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

func TestApplyFail(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test file not Exists",
			args: args{
				file: "test.yaml",
			},
			wantErr: true,
		},
		{
			name:    "test workflow not node",
			args:    args{file: "testdata/workflow-no-node.yaml"},
			wantErr: true,
		},
		{
			name:    "test workflow name empty",
			args:    args{file: "testdata/workflow-name-empty.yaml"},
			wantErr: true,
		},
		{
			name:    "test node invalid",
			args:    args{file: "testdata/workflow-node-invalid.yaml"},
			wantErr: true,
		},
	}
	sqliteDB, err := model.NewDB(context.Background(), model.DBConfig{
		Dialect: model.DialectSQLite,
		DSN:     "file::memory:?cache=shared",
	})
	if err != nil {
		err = fmt.Errorf("NewDB SQLite: %w", err)
		panic(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ApplyFile(context.TODO(), nil, tt.args.file, sqliteDB); (err != nil) != tt.wantErr {
				t.Errorf("ApplyFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
