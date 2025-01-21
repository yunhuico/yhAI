package fs

import (
	"testing"
)

func TestFileExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test a not exists file",
			args: args{
				path: "./testdata/test.txt",
			},
			want: false,
		},
		{
			name: "test a exists file",
			args: args{
				path: "./testdata/foo",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExist(tt.args.path); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirExist(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"test a not exists dir",
			args{
				path: "./testdata/not_exist",
			},
			false,
			false,
		},
		{
			"test a file",
			args{
				path: "./fs.go",
			},
			false,
			false,
		},
		{
			"test a file",
			args{
				path: "./fs.go",
			},
			false,
			false,
		},
		{
			"test a dir",
			args{
				path: "./testdata",
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DirExist(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DirExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DirExist() = %v, want %v", got, tt.want)
			}
		})
	}
}
