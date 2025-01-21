package auth

import (
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

type TokenMeta struct {
	ExtraKeys []string `json:"extraKeys"`
}

// GetTokenMetaData returns extra metadata from the server when updating a token.
func GetTokenMetaData(token *oauth2.Token, credentialData *model.CredentialData) ([]byte, error) {
	tokenMetaData := TokenMeta{}
	err := json.Unmarshal(credentialData.MetaData, &tokenMetaData)
	if err != nil {
		return nil, fmt.Errorf("decoding credential token meta keys: %w", err)
	}

	if len(tokenMetaData.ExtraKeys) == 0 {
		return nil, nil
	}

	rawData := make(map[string]any)
	for _, key := range tokenMetaData.ExtraKeys {
		value := token.Extra(key)
		if value == nil {
			return nil, fmt.Errorf("querying token extra: no value of %s provided", key)
		}
		rawData[key] = value
	}

	b, err := json.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("json marshaling token meta data: %w", err)
	}
	return b, nil
}
