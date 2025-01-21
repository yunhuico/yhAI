package payload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/service"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/utils/validator"
)

type EditCredentialReq struct {
	model.EditableCredential

	// InputFields submitted by frontend, is not a column.
	// All values should be string.
	InputFields map[string]string `json:"inputFields"`
}

func (p *EditCredentialReq) Validate(ctx context.Context) (err error) {
	if p.Type == model.CredentialTypeOAuth {
		if p.Name == "" {
			err = errors.New("name can not be empty")
			return
		}

		return
	}

	// if using official credential, don't validate, trust credential config.
	if p.OfficialName == "" {
		var rawData []byte
		rawData, err = json.Marshal(p.InputFields)
		if err != nil {
			err = errors.New("credential input fields cannot marshal")
			return
		}
		_, err = service.ComposeCredentialData(p.AdapterClass, string(p.Type), rawData)
		if err != nil {
			err = fmt.Errorf("credential form input values invalid: %w", err)
			return
		}

		adapterManager := adapter.GetAdapterManager()
		meta := adapterManager.LookupAdapter(p.AdapterClass)
		var form *adapter.CredentialForm
		form, err = meta.GetCredentialForm(string(p.Type))
		if err != nil {
			err = fmt.Errorf("getting credential form: %w", err)
			return
		}
		err = form.ValidateDynamically(p.InputFields)
		if err != nil {
			err = fmt.Errorf("validating credential dynamically: %w", err)
			return
		}
	}

	return
}

// Normalize credential payload to model
func (p *EditCredentialReq) Normalize() (*model.Credential, error) {
	credential := &model.Credential{
		EditableCredential: model.EditableCredential{
			Name:         p.Name,
			AdapterClass: p.AdapterClass,
			Type:         p.Type,
		},
	}
	if !p.Type.IsOAuth2() {
		credential.Status = model.CredentialStatusAvailable
	}
	return credential, nil
}

type (
	RequestAuthURLReq struct {
		// CredentialID if mode is normal, credentialID keep empty
		CredentialID string `json:"credentialId" validate:"required"`
		// ForceRefresh specified this field if the credential expired or broken.
		ForceRefresh bool `json:"forceRefresh"`
		// If specified the redirect URL, will redirect to the url after user confirm.
		RedirectURL string `json:"redirectUrl"`
	}
)

func (r *RequestAuthURLReq) Validate() (err error) {
	if err = validator.Validate(r); err != nil {
		return fmt.Errorf("RequestAuthURLReq validation failed: %w", err)
	}

	return nil
}
