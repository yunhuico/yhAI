package apiserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isUserInsider(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{
			email: "someone@example.com",
			want:  false,
		},
		{
			email: "wooooh@gitlab.cn",
			want:  true,
		},
		{
			email: "wooooh@jihulab.com",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equalf(t, tt.want, isUserInsider(tt.email), "isUserInsider(%v)", tt.email)
		})
	}
}

func Test_isLDAPDummyEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{
			email: "temp-email-for-oauth-knight@gitlab.localhost",
			want:  true,
		},
		{
			email: "someone@example.com",
			want:  false,
		},
		{
			email: "wooooh@gitlab.cn",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equalf(t, tt.want, isLDAPDummyEmailAddress(tt.email), "isLDAPDummyEmailAddress(%v)", tt.email)
		})
	}
}
