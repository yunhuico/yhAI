package jira

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		baseURL     string
		wantBaseURL string
		wantErr     bool
	}{
		{
			baseURL:     "http://exmaple.com",
			wantBaseURL: "",
			wantErr:     true,
		},
		{
			baseURL:     "https://example.com",
			wantBaseURL: "https://example.com",
			wantErr:     false,
		},
		{
			baseURL:     "https://example.com/a/b",
			wantBaseURL: "https://example.com/a/b",
			wantErr:     false,
		},
		{
			baseURL:     "https://example.com/a/b/",
			wantBaseURL: "https://example.com/a/b",
			wantErr:     false,
		},
		{
			baseURL:     "https://example.com/a/b//",
			wantBaseURL: "https://example.com/a/b",
			wantErr:     false,
		},
		{
			baseURL:     "a/b",
			wantBaseURL: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.baseURL, func(t *testing.T) {
			gotClient, err := NewClient(Config{
				AccountEmail: "jeff@example.com",
				APIToken:     "mySuperSecrectToken",
				BaseURL:      tt.baseURL,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if gotClient.baseURL != tt.wantBaseURL {
				t.Errorf("want base URL %q, got %q", tt.wantBaseURL, gotClient.baseURL)
			}
		})
	}
}
