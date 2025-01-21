package yaml

import "testing"

func TestUnmarshalWithFile(t *testing.T) {
	type args struct {
		file string
		dist interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test file not exists",
			args{
				"test.yaml",
				&struct {
					Name string `yaml:"name"`
				}{},
			},
			true,
		},
		{
			"test file exists but unmarshal failed",
			args{
				"testdata/foo.yaml",
				&struct {
					Name int `yaml:"name"`
				}{},
			},
			true,
		},
		{
			"test file exists and unmarshal success",
			args{
				"testdata/foo.yaml",
				&struct {
					Name string `yaml:"name"`
				}{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnmarshalWithFile(tt.args.file, tt.args.dist); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalWithFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
