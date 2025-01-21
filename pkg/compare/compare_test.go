package compare

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCompareEquals(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare two empty string",
			args: args{
				left:  "",
				right: "",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare empty string and nil",
			args: args{
				left:  "",
				right: nil,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare empty string and bool",
			args: args{
				left:  "true",
				right: true,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "left is array",
			args: args{
				left:  []string{"a", "b", "c"},
				right: true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComparator()
			got, err := c.Compare(EqualsOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNotEquals(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare two empty string",
			args: args{
				left:  "",
				right: "",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare empty string and nil",
			args: args{
				left:  "",
				right: nil,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare empty string and bool",
			args: args{
				left:  "true",
				right: true,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "right is array",
			args: args{
				left:  true,
				right: []string{"a", "b", "c"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(NotEqualsOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareContains(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare nil contains nil",
			args: args{
				left:  nil,
				right: nil,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare foobar contains foo",
			args: args{
				left:  "foobar",
				right: "foo",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare [foo, bar] contains foo",
			args: args{
				left:  []string{"foo", "bar"},
				right: "foo",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare [foo, bar] contains baz",
			args: args{
				left:  []string{"foo", "bar"},
				right: "baz",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare [foo, bar, [baz]] contains baz error",
			args: args{
				left:  []any{"foo", "bar", []string{"baz"}},
				right: "baz",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(ContainsOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNotContains(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare nil not contains nil",
			args: args{
				left:  nil,
				right: nil,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare [foo, bar] not contains foo",
			args: args{
				left:  []string{"foo", "bar"},
				right: "foo",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare foobar not contains [foo, bar]",
			args: args{
				left:  "foobar",
				right: []string{"foo", "bar"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(NotContainsOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareGreaterThan(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare left greater than right",
			args: args{
				left:  "30.01",
				right: "29",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare left greater than right",
			args: args{
				left:  "0.1",
				right: "0.01",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare left greater than right",
			args: args{
				left:  "1.0",
				right: "0.99999",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare left greater than right",
			args: args{
				left:  "1.0",
				right: "0.9999999999",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(GreaterThan, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareStartWith(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare foobar start with foo",
			args: args{
				left:  "foobar",
				right: "foo",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare left is [foo, bar]",
			args: args{
				left: []string{"foo", "bar"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(StringStartWithOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareEndWith(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare foobar end with foo",
			args: args{
				left:  "foobar",
				right: "foo",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "compare foobar end with bar",
			args: args{
				left:  "foobar",
				right: "bar",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(StringEndWithOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareTimeAgoOperation(t *testing.T) {
	type args struct {
		left  any
		right any
		op    Operation
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare left is not time",
			args: args{
				left: "foo:bar:baz",
				op:   TimeDayAgoOperation,
			},
			wantErr: true,
		},
		{
			name: "compare now before one day ago",
			args: args{
				left:  time.Now().Format(time.RFC3339),
				right: 1,
				op:    TimeDayAgoOperation,
			},
			want: false,
		},
		{
			name: "compare two hours ago before one hour ago",
			args: args{
				left:  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				right: 1,
				op:    TimeHourAgoOperation,
			},
			want: true,
		},
		{
			name: "compare one day ago before 23 hour ago",
			args: args{
				left:  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				right: 23,
				op:    TimeHourAgoOperation,
			},
			want: true,
		},
		{
			name: "left is not time",
			args: args{
				left: "xxxxxxx",
				op:   TimeHourAgoOperation,
			},
			wantErr: true,
		},
		{
			name: "right is not int",
			args: args{
				left:  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				right: "xxxxxx",
				op:    TimeHourAgoOperation,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComparator()
			got, err := c.Compare(tt.args.op, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareEmpty(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "compare nil is empty",
			args: args{
				left: nil,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare [] is empty",
			args: args{
				left: []any{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "compare []int{1,2,3} is not empty",
			args: args{
				left: []int{1, 2, 3},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := comparator{}
			got, err := c.Compare(EmptyOperation, tt.args.left, tt.args.right)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Compare() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNotEmpty(t *testing.T) {
	c := comparator{}
	got, err := c.Compare(NotEmptyOperation, nil, false)
	assert.NoError(t, err)
	assert.Equal(t, false, got)
}

func Test_assertTimeBefore(t *testing.T) {
	type args struct {
		left  any
		right any
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "left is invalid",
			args: args{
				left:  "xxx",
				right: nil,
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				return true
			},
		},
		{
			name: "left == right",
			args: args{
				left:  "2022-02-02 12:12:12",
				right: "2022-02-02 12:12:12",
			},
			want: false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
		{
			name: "left < right",
			args: args{
				left:  "2022-02-02 12:12:12",
				right: "2022-02-02 12:12:20",
			},
			want: true,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
		{
			name: "left < right with tz",
			args: args{
				left:  "2023-02-20T10:00:00+08:00",
				right: "2023-02-20T08:00:00+02:00",
			},
			want: true,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.NoError(t, err)
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := assertTimeBefore(tt.args.left, tt.args.right)
			if !tt.wantErr(t, err, fmt.Sprintf("assertTimeBefore(%v, %v)", tt.args.left, tt.args.right)) {
				return
			}
			assert.Equalf(t, tt.want, got, "assertTimeBefore(%v, %v)", tt.args.left, tt.args.right)
		})
	}
}
