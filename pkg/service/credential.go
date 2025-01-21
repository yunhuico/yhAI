package service

import (
	"bytes"
	"encoding/json"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

// ComposeCredentialData compose CredentialData by given rawData.
// rawData is a decrypted data (json []byte, from Credentials.Data),
// rawData is a map before encrypted.
func ComposeCredentialData(adapterClass string, credentialType string, rawData []byte) (credData *model.CredentialData, err error) {
	rawMap := map[string]any{}
	err = json.Unmarshal(rawData, &rawMap)
	if err != nil {
		err = fmt.Errorf("unmarshaling raw data to map: %w", err)
		return
	}

	adapterManager := adapter.GetAdapterManager()
	meta := adapterManager.LookupAdapter(adapterClass)
	if meta == nil {
		err = fmt.Errorf("unknown adapter %q", adapterClass)
		return
	}

	tpl, err := meta.GetCredentialTemplate(credentialType)
	if err != nil {
		err = fmt.Errorf("getting credential template: %w", err)
		return
	}

	var buf bytes.Buffer
	err = tpl.Execute(&buf, rawMap)
	if err != nil {
		err = fmt.Errorf("executing credential template: %w", err)
		return
	}

	credData = &model.CredentialData{}
	decoder := json.NewDecoder(&buf)
	err = decoder.Decode(credData)
	if err != nil {
		err = fmt.Errorf("decoding credential data: %w", err)
		return
	}

	return
}
