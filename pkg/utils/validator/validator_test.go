package validator

import "testing"

type dummy struct {
	A string `validate:"required"`
	B int    `validate:"max=5"`
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{
			name:    "nil",
			value:   nil,
			wantErr: true,
		},
		{
			name:    "unexpected kind",
			value:   1,
			wantErr: true,
		},
		{
			name: "good",
			value: dummy{
				A: "a",
				B: 0,
			},
			wantErr: false,
		},
		{
			name: "bad1",
			value: dummy{
				A: "",
				B: 0,
			},
			wantErr: true,
		},
		{
			name: "bad2",
			value: dummy{
				A: "",
				B: 6,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.value); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
