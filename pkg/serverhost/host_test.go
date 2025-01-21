package serverhost

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestServerHost_IsInboundURL(t *testing.T) {
	m, err := New(Opt{
		API:     "https://example.com/",
		WebHook: "https://example.com:4567",
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		URL      string
		wantRoot string
		wantErr  bool
	}{
		{
			name:     "empty",
			URL:      "",
			wantRoot: "",
			wantErr:  true,
		},
		{
			name:     "relative",
			URL:      "/hello",
			wantRoot: "",
			wantErr:  false,
		},
		{
			name:     "relative hashed",
			URL:      "/hello?a=b#1d56a4w",
			wantRoot: "",
			wantErr:  false,
		},
		{
			name:     "localhost",
			URL:      "http://localhost:8008/hello",
			wantRoot: "http://localhost:8008",
			wantErr:  false,
		},
		{
			name:     "no scheme",
			URL:      "localhost:8008/hello",
			wantRoot: "",
			wantErr:  true,
		},
		{
			name:     "outbound unsecure",
			URL:      "http://example.com/hello",
			wantRoot: "",
			wantErr:  true,
		},
		{
			name:     "outbound secure",
			URL:      "https://example.com/hello",
			wantRoot: "https://example.com",
			wantErr:  false,
		},
		{
			name:     "outbound secure but wrong port",
			URL:      "https://example.com:6666/hello",
			wantRoot: "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoot, err := m.IsInboundURL(tt.URL)
			if tt.wantErr != (err != nil) {
				t.Errorf("wantErr is %v, while error is %v", tt.wantErr, err)
				return
			}
			assert.Equalf(t, tt.wantRoot, gotRoot, "IsInboundURL(%v)", tt.URL)
		})
	}
}

func TestServerHost_API(t *testing.T) {
	m, err := New(Opt{
		API:     "https://example.com/",
		WebHook: "https://example.com:4567",
	})
	require.NoError(t, err)

	require.Equal(t, "https://example.com", m.API())
}

func TestServerHost_Webhook(t *testing.T) {
	m, err := New(Opt{
		API:     "https://example.com/",
		WebHook: "https://example.com:4567",
	})
	require.NoError(t, err)

	require.Equal(t, "https://example.com:4567", m.Webhook())
	require.Equal(t, "https://example.com:4567/webhook/1", m.WebhookFullURL("webhook/1"))
}

func TestServerHost_APIFullURL(t *testing.T) {
	m, err := New(Opt{
		API:     "https://example.com/",
		WebHook: "https://example.com:4567",
	})
	require.NoError(t, err)

	require.Equal(t, "https://example.com/a/b/c", m.APIFullURL("/a/b/c"))
}
