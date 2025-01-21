package formatter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTime_Run(t *testing.T) {
	type fields struct {
		InputTime    string
		ToFormat     string
		FromFormat   string
		ToTimezone   string
		FromTimezone string
	}
	tests := []struct {
		name    string
		fields  fields
		want    any
		wantErr bool
	}{
		{
			name: "test y-m-d H:i:s",
			fields: fields{
				InputTime:  "2022-10-10 12:12:12",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
				FromFormat: "2006-01-02 15:04:05",
				ToTimezone: "Japan",
			},
			want:    "2022-10-10T21:12:12+09:00",
			wantErr: false,
		},
		{
			name: "test timestamp",
			fields: fields{
				InputTime:    "1597247662",
				ToFormat:     "2006-01-02T15:04:05Z07:00",
				FromFormat:   "timestamp",
				FromTimezone: "UTC",
				ToTimezone:   "Asia/Shanghai",
			},
			want:    "2020-08-12T23:54:22+08:00",
			wantErr: false,
		},
		{
			name: "test YYYY-MM-DD",
			fields: fields{
				InputTime:  "2022-12-01",
				FromFormat: "2006-01-02",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			want:    "2022-12-01T00:00:00Z",
			wantErr: false,
		},
		{
			name: "test YYYY/MM/DD",
			fields: fields{
				InputTime:  "2022/12/01",
				FromFormat: "2006/01/02",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			want:    "2022-12-01T00:00:00Z",
			wantErr: false,
		},
		{
			name: "test from format error",
			fields: fields{
				InputTime:  "2022/12/01",
				FromFormat: "2006,01,02",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			wantErr: true,
		},
		{
			name: "test YYYY-MM",
			fields: fields{
				InputTime:  "2022-12",
				FromFormat: "2006-01",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			want:    "2022-12-01T00:00:00Z",
			wantErr: false,
		},
		{
			name: "test MM-DD",
			fields: fields{
				InputTime:  "12-12",
				FromFormat: "01-02",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			want:    "0000-12-12T00:00:00Z",
			wantErr: false,
		},
		{
			name: "test YYYY年MM月DD日",
			fields: fields{
				InputTime:  "2019年12月23日",
				FromFormat: "2006年01月02日",
				ToFormat:   "2006-01-02T15:04:05Z07:00",
			},
			want:    "2019-12-23T00:00:00Z",
			wantErr: false,
		},
		{
			name: "test today",
			fields: fields{
				InputTime:  "today",
				FromFormat: "2006年01月02日",
				ToFormat:   "2006年01月02日",
			},
			want:    time.Now().Format("2006年01月02日"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FormatTime{
				InputTime:    tt.fields.InputTime,
				ToFormat:     tt.fields.ToFormat,
				FromFormat:   tt.fields.FromFormat,
				ToTimezone:   tt.fields.ToTimezone,
				FromTimezone: tt.fields.FromTimezone,
			}
			got, err := f.Run(nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equalf(t, tt.want, got, "Run()")
		})
	}
}
