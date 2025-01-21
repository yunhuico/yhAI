package trans

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"github.com/xanzy/go-gitlab"
)

func TestToSliceSlice(t *testing.T) {
	type args struct {
		input any
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "test nil",
			args: args{
				input: nil,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "test []any",
			args: args{
				input: []any{
					"a", "b", "c",
				},
			},
			want: []string{
				"a", "b", "c",
			},
			wantErr: false,
		},
		{
			name: "test []int",
			args: args{
				input: []int{
					1, 2, 3,
				},
			},
			want: []string{
				"1", "2", "3",
			},
			wantErr: false,
		},
		{
			name: "test []int",
			args: args{
				input: map[string]any{"foo": "bar"},
			},
			wantErr: true,
		},
		{
			name: "test gitlab.Labels",
			args: args{
				input: gitlab.Labels{"foo", "bar"},
			},
			want:    []string{"foo", "bar"},
			wantErr: false,
		},
		{
			name: "test *gitlab.Labels",
			args: args{
				input: &gitlab.Labels{"foo", "bar"},
			},
			want:    []string{"foo", "bar"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToStringSlice(tt.args.input)
			if tt.wantErr {
				assert.Errorf(t, err, fmt.Sprintf("%s should return error", tt.name))
				return
			}
			assert.Equalf(t, tt.want, got, "ToStringSlice(%v)", tt.args.input)
		})
	}
}

func TestToAnySlice(t *testing.T) {
	instanceID := "id1"
	instanceName := "name1"
	cvmList := []*cvm.Instance{
		{
			InstanceId:   &instanceID,
			InstanceName: &instanceName,
		},
	}

	type args struct {
		input any
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "test nil",
			args: args{
				input: nil,
			},
			want:    []any{},
			wantErr: false,
		},
		{
			name: "test []any{1, 2, 3}",
			args: args{
				input: []any{1, 2, 3},
			},
			want:    []any{1, 2, 3},
			wantErr: false,
		},
		{
			name: "test cvm list",
			args: args{
				input: cvmList,
			},
			want: []any{
				&cvm.Instance{
					InstanceId:   &instanceID,
					InstanceName: &instanceName,
				},
			},
		},
		{
			name: "test cvm list pointer",
			args: args{
				input: &cvmList,
			},
			want: []any{
				&cvm.Instance{
					InstanceId:   &instanceID,
					InstanceName: &instanceName,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, err := ToAnySlice(tt.args.input)
			if tt.wantErr {
				assert.Errorf(t, err, fmt.Sprintf("%s should return error", tt.name))
				return
			}
			assert.Equalf(t, tt.want, gotResult, "ToAnySlice(%v)", tt.args.input)
		})
	}
}
