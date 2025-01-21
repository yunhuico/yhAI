package adapter

import (
	"context"
	"embed"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegisterAdapter(t *testing.T) {
	adapterManager := NewAdapterManager()
	t.Run("test panic when name empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		adapterManager.RegisterAdapter(&Meta{})
	})

	t.Run("test panic when class empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		adapterManager.RegisterAdapter(&Meta{
			BaseMeta: BaseMeta{
				Name: "test",
			},
		})
	})

	t.Run("test register successfully, but cannot register twice", func(t *testing.T) {
		adapterManager.RegisterAdapter(&Meta{
			BaseMeta: BaseMeta{
				Name:  "test",
				Class: "ultrafox/test",
			},
		})

		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		adapterManager.RegisterAdapter(&Meta{
			BaseMeta: BaseMeta{
				Name:  "test",
				Class: "ultrafox/test",
			},
		})
	})
}

func TestRegisterAdapterByRaw(t *testing.T) {
	adapterManager := NewAdapterManager()
	t.Run("test panic when raw is invalid json", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		adapterManager.RegisterAdapterByRaw([]byte(``))
	})

	t.Run("test successfully when no support credentials", func(t *testing.T) {
		adapterManager.RegisterAdapterByRaw([]byte(`{"name": "Test", "class": "ultrafox/test"}`))
	})

	t.Run("test successfully when has accessToken credential", func(t *testing.T) {
		adapterManager.RegisterAdapterByRaw([]byte(`{
			"name": "Test",
			"class": "ultrafox/test2",
			"supportCredentials": [
				"accessToken"
			],
			"inputSchemas": {
				"accessToken": {
					"fields": []
				}
			}
		}`))

		meta := adapterManager.LookupAdapter("ultrafox/test2")
		assert.True(t, meta.RequireAuth())
		accessTokenForm, err := meta.GetCredentialForm("accessToken")
		assert.NoError(t, err)
		assert.NotNil(t, accessTokenForm)

		_, err = meta.GetCredentialForm("notExists")
		assert.Error(t, err)
	})

	t.Run("test successfully when has accessToken credential, not don't provide inputSchema", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()

		adapterManager.RegisterAdapterByRaw([]byte(`{
			"name": "Test",
			"class": "ultrafox/test2",
			"supportCredentials": [
				"accessToken"
			],
			"inputSchemas": {
			}
		}`))
	})
}

//go:embed testdata/empty/*
var emptyDir embed.FS

//go:embed testdata
var adapterFS embed.FS

func TestRegisterSpecsByDir(t *testing.T) {
	adapterManager := NewAdapterManager()
	meta := &Meta{
		BaseMeta: BaseMeta{
			Name:  "test",
			Class: "ultrafox/test",
		},
	}
	adapterManager.RegisterAdapter(meta)
	t.Run("empty directory", func(t *testing.T) {
		meta.RegisterSpecsByDir(emptyDir)
	})

	t.Run("adapters directory", func(t *testing.T) {
		meta.RegisterSpecsByDir(adapterFS)
		assert.Len(t, adapterManager.LookupAdapter("ultrafox/test").Specs, 2)
	})
}

func TestRegisterSpecsByRaw(t *testing.T) {
	adapterManager := NewAdapterManager()
	meta := &Meta{
		BaseMeta: BaseMeta{
			Name:  "test",
			Class: "ultrafox/test",
		},
	}
	adapterManager.RegisterAdapter(meta)

	assert.Equal(t, 0, adapterManager.GetSpecCountByType(SpecTriggerType))
	assert.Equal(t, 1, adapterManager.GetMetaCount())
	adapterManager.Present(defaultLanguage)

	t.Run("spec name empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		meta.RegisterSpecByRaw([]byte(`{
			"name": ""
		}`))
	})

	t.Run("trigger don't set shortName", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		meta.RegisterSpecByRaw([]byte(`{
			"name": "trigger",
			"type": "trigger"
			"adapterClass": "ultrafox/test"
		}`))
	})

	t.Run("actor set triggerType", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		meta.RegisterSpecByRaw([]byte(`{
			"name": "actor",
			"type": "actor",
			"adapterClass": "ultrafox/test"
			"triggerType": "trigger"
		}`))
	})

	t.Run("trigger don't set triggerType", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("function should panic")
			}
		}()
		meta.RegisterSpecByRaw([]byte(`{
			"name": "trigger",
			"type": "trigger",
			"adapterClass": "ultrafox/test"
			"shortName": "Trigger"
		}`))
	})

	t.Run("register successfully", func(t *testing.T) {
		meta.RegisterSpecByRaw([]byte(`{
			"name": "actor",
			"type": "actor",
			"adapterClass": "ultrafox/test"
		}`))

		assert.Equal(t, 1, adapterManager.GetSpecCountByType(SpecActorType))
	})
}

func TestAdapterLocalization(t *testing.T) {
	adapterManager := NewAdapterManager()
	meta := adapterManager.RegisterAdapterByRaw([]byte(`{
    "name": "text",
    "class": "ultrafox/test",
	"supportCredentials": [
		"oauth2"
	],
    "inputSchemas": {
        "oauth2": {
            "fields": [
                {
                    "key": "server",
                    "label": {
                        "en-US": "Server address",
                        "zh-CN": "服务器地址"
                    },
                    "type": "string",
                    "default": "https://gitlab.com",
                    "required": true
                }
            ]
        }
    }
}`))
	meta.RegisterSpecByRaw([]byte(`{
    "name": "Test Action2",
    "type": "actor",
    "adapterClass": "ultrafox/test",
    "class": "ultrafox/test#action2",
    "inputSchema": {
        "fields": [
            {
                "key": "key",
                "label": {
					"en-US": "Key",
					"zh-CN": "关键字"
				},
                "type": "string",
                "style": "default",
                "required": true
            }
        ]
    },
    "outputSchema": {
        "fields": [
			{
				"key": "Name",
				"label": {
					"en-US": "Name",
					"zh-CN": "名字"
				}
			}
		]
    }
}`))

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Microsecond)
	defer cancel()
	assertLang := func(lang string, assertFn func(data ListPresentData)) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				data := adapterManager.Present(lang)
				assertFn(data)
			}
		}
	}
	go func() {
		assertLang("en-US", func(data ListPresentData) {
			assert.Len(t, data.AdapterList, 1)
			assert.Equal(t, "Key", data.AdapterList[0].Specs[0].InputSchema[0].Label)
			assert.Equal(t, "Name", data.AdapterList[0].Specs[0].OutputSchema[0].Label)
			assert.Equal(t, "Server address", data.AdapterList[0].CredentialForms["oauth2"].InputForm.Fields[0].Label)
		})
	}()
	go func() {
		assertLang("zh-CN", func(data ListPresentData) {
			assert.Len(t, data.AdapterList, 1)
			assert.Equal(t, "关键字", data.AdapterList[0].Specs[0].InputSchema[0].Label)
			assert.Equal(t, "名字", data.AdapterList[0].Specs[0].OutputSchema[0].Label)
			assert.Equal(t, "服务器地址", data.AdapterList[0].CredentialForms["oauth2"].InputForm.Fields[0].Label)
		})
	}()

	<-ctx.Done()
}
