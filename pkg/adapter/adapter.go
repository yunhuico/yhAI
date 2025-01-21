// Package adapter implements ultrafox/spec: Adapter Design described at
// https://jihulab.com/jihulab/ultrafox/spec/-/tree/main/design/adapter).
//
// Much of the implementation is borrowed directly from the above spec.
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"

	"github.com/Masterminds/sprig/v3"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
)

var (
	CustomCredentialType      = "custom"
	AccessTokenCredentialType = "accessToken"
	// OAuth2CredentialType legacy OAuth credential type
	// DEPRECATED: use OAuthCredentialType instead.
	OAuth2CredentialType = "oauth2"
	OAuthCredentialType  = "OAuth"

	// the input data is the credential data map, like:
	// 	{
	// 		"accessToken": "xxxxxxxxxxxxxx"
	// 		"server": "xxxxxxxxxxxxxx"
	// }
	defaultAccessTokenTemplate = `{
	"accessToken": "{{ .accessToken }}"
}`
	defaultOAuth2Template = `{
	"oauth2Config": {{ . | toJson }}
}`
	defaultCustomTemplate = `{
	"metaData": {{ . | toJson }}
}`

	defaultAccessTokenTpl *template.Template
	defaultOAuth2Tpl      *template.Template
	defaultCustomTpl      *template.Template
)

func init() {
	var err error
	defaultCustomTpl, err = template.New("custom-template").Funcs(sprig.TxtFuncMap()).Parse(defaultCustomTemplate)
	if err != nil {
		panic(err)
	}
	defaultOAuth2Tpl, err = template.New("oauth2-template").Funcs(sprig.TxtFuncMap()).Parse(defaultOAuth2Template)
	if err != nil {
		panic(err)
	}
	defaultAccessTokenTpl, err = template.New("access-token-template").Funcs(sprig.TxtFuncMap()).Parse(defaultAccessTokenTemplate)
	if err != nil {
		panic(err)
	}
}

type jsonSpec struct {
	BaseSpec

	Name             Lang `json:"name"`
	Desc             Lang `json:"desc"`
	NoMoreSamplesMsg Lang `json:"noMoreSamplesMsg"`

	InputSchema struct {
		Fields internalInputFormFields `json:"fields"`
	} `json:"inputSchema"`

	OutputSchema struct {
		Fields internalOutputFields `json:"fields"`
	} `json:"outputSchema"`

	TestPageSchema struct {
		Fields internalInputFormFields `json:"fields"`
	} `json:"testPageSchema"`
}

func (s jsonSpec) Validate() error {
	if !s.Name.Defined() {
		return fmt.Errorf("spec name cannot be empty")
	}

	if s.Type == model.NodeTypeTrigger {
		switch model.TriggerType(s.TriggerType) {
		case model.TriggerTypeWebhook, model.TriggerTypeCron, model.TriggerTypePoll:
			// relax
		default:
			return fmt.Errorf("trigger type should be one of webhook, cron, poll. Got %q", s.TriggerType)
		}
	} else {
		if s.TriggerType != "" {
			return errors.New("only trigger can defines triggerType")
		}
	}

	return nil
}

func (s jsonSpec) ToDefault() BaseSpec {
	return s.ToLang("en-US")
}

func (s jsonSpec) ToLang(lang string) BaseSpec {
	spec := s.BaseSpec

	if spec.DocsPath == "" {
		spec.DocsPath = buildSpecDocsPath(spec.Class, spec.Type)
	}

	spec.Name = s.Name.GetLang(lang)
	spec.Desc = s.Desc.GetLang(lang)
	spec.NoMoreSamplesMsg = s.NoMoreSamplesMsg.GetLang(lang)
	return spec
}

func buildSpecDocsPath(class string, specType AdapterSpecType) string {
	// class format: .*/${adapterName}#${specName}
	tmp := strings.Split(class, "#")
	specName := tmp[len(tmp)-1]
	tmp = strings.Split(tmp[0], "/")
	adapterName := tmp[len(tmp)-1]

	// adapterName + specType plural + specName
	return "/docs/applications/" + adapterName + "/" + string(specType) + "s" + "/#" + specName
}

type BaseSpec struct {
	// Name, defined in definition.cue if adapter is driver by cue,
	// set by external if driver by custom node.
	Name string `json:"name"`

	Desc string `json:"desc"`

	Class string `json:"class"`

	DocsPath string `json:"docsPath"`

	// Driver the driver of this adapter. if driver by custom node, the `definition.cue` not exists.
	// the frontend should not care about this field.
	Driver AdapterDriver `json:"-"`

	// Type set by node UltrafoxNode.
	Type AdapterSpecType `json:"type"`
	// AdapterClass the adapter app name.
	// frontend use this field submit the request when select credential connection or create node.
	AdapterClass string `json:"adapterClass"`
	// TriggerType the trigger type of spec, trigger type keep empty if is actor.
	// UltraFox supports cron, webhook trigger.
	TriggerType string `json:"triggerType,omitempty"`

	// ShortName for displaying the trigger sample name, like: Issue - 1, Issue - 2, etc.
	ShortName string `json:"shortName,omitempty"`

	// Hidden if set to true, this spec does not expose.
	Hidden bool `json:"hidden,omitempty"`

	// NoMoreSamplesMsg displays when user click "Load more samples" in trigger node, but no sample is loaded.
	NoMoreSamplesMsg string `json:"noMoreSamplesMsg,omitempty"`

	// When to set EnableTriggerAtFirst as true?
	// If trigger type is webhook(not custom-webhook), and the trigger can't implement trigger.SampleProvider.
	// Then worker can transform callback to sample(workflow_instance_nodes)
	EnableTriggerAtFirst bool `json:"enableTriggerAtFirst,omitempty"`
}

// Spec contains information of a function of one adapter,
// which corresponds with a hash in class URL.
type Spec struct {
	BaseSpec

	// This output schema is default, echo value field (like label, desc) written by english.
	// other languages defined in langOutputSchema.
	OutputSchema OutputFields `json:"outputSchema,omitempty"`

	// This input schema is default, echo value field (like label, desc) written by english.
	// other languages defined in langInputSchema.
	InputSchema InputFormFields `json:"inputSchema,omitempty"`

	// AdapterMeta reference to the adapter meta.
	AdapterMeta *Meta `json:"-"`

	// TestPageSchema will be nil if don't define test page schema.
	TestPageSchema InputFormFields `json:"testPageSchema,omitempty"`

	// basicValidator will be not nil when exists required field.
	basicValidator *basicValidator `json:"-"`

	langBase           map[string]BaseSpec
	langOutputSchema   map[string]OutputFields
	langInputSchema    map[string]InputFormFields
	langTestPageSchema map[string]InputFormFields
}

func (s Spec) IsTrigger() bool {
	return s.Type == SpecTriggerType
}

func (s Spec) GenerateNodeMetaData() model.NodeMetaData {
	return model.NodeMetaData{
		AdapterClass:         s.AdapterClass,
		TriggerType:          model.TriggerType(s.TriggerType),
		EnableTriggerAtFirst: s.EnableTriggerAtFirst,
	}
}

func (s Spec) ValidateDynamically(inputFields map[string]any) (err error) {
	if s.basicValidator == nil {
		return
	}

	return s.basicValidator.validate(inputFields)
}

// AdapterDriver the adapter driver by what's type of node.
type AdapterDriver string

const (
	// DriverCUE is the adapter driver for pure CUE.
	DriverCUE = AdapterDriver("cue")
	// DriverCustomNode is the adapter driver by custom node
	DriverCustomNode = AdapterDriver("customNode")
)

// AdapterSpecType equals model.NodeType
type AdapterSpecType = model.NodeType

const (
	SpecTriggerType AdapterSpecType = "trigger"
	SpecActorType   AdapterSpecType = "actor"
	SpecLogicType   AdapterSpecType = "logic"
)

type BaseMeta struct {
	// User-facing name
	Name string `json:"name" yaml:"name" validate:"required"`
	// Must match its implementation's class,
	// Implicit declaration, automatically generated by dir path.
	Class string `json:"class" yaml:"class" validate:"required"`
	// Semver
	Version string `json:"version" yaml:"version" validate:"required"`
	// User-facing description
	Description string `json:"description" yaml:"description"`
	// URL pointing to icon file
	Icon string `json:"icon" yaml:"icon"`
	// URL to related service, like: https://gitlab.com
	URL string `json:"url,omitempty" yaml:"url"`
	// for classification
	Tags []string `json:"tags,omitempty" yaml:"tags"`
	// SupportCredentials defines in metadata
	SupportCredentials []string `json:"supportCredentials,omitempty" yaml:"supportCredentials"`
	// IsInternal indicates whether this adapter is internal.
	IsInternal bool `json:"isInternal,omitempty" yaml:"isInternal"`
}

type jsonMeta struct {
	BaseMeta

	Name        Lang `json:"name" yaml:"name" validate:"required"`
	Description Lang `json:"description" yaml:"description"`

	InputSchemas map[string]struct {
		Fields internalInputFormFields `json:"fields"`
	} `json:"inputSchemas"`
}

// Meta metadata of one adapter, which is consisted of multiple spec
type Meta struct {
	BaseMeta

	// Specs define every actor or trigger
	Specs []*Spec `json:"specs"`
	// CredentialForms parse credential.cue if exists SupportCredentials
	CredentialForms map[string]*CredentialForm `json:"credentialForms,omitempty"`

	adapterManager  *AdapterManager
	langName        map[string]string
	langDescription map[string]string

	// credentialTestingFunc can be nil when don't register credential validation function.
	credentialTestingFunc CredentialTestingFunc
}

type CredentialTestingFunc func(ctx context.Context, credentialType model.CredentialType, inputFields model.InputFields) (err error)

// CredentialForm defines credential form
type CredentialForm struct {
	InputForm *InputFormField `json:"inputForm"`

	// This template registered by node.
	// It will be very ugly if defined in json, because it is raw json string.
	Template *template.Template `json:"-"`

	// Masker each credential form should bind a Masker.
	Masker Masker `json:"-"`

	langInputForm map[string]*InputFormField

	// basicValidator will be not nil when exists required field.
	basicValidator *basicValidator `json:"-"`
}

func (m *Meta) RequireAuth() bool {
	return len(m.SupportCredentials) > 0
}

// GetCredentialForm by credential type.
// If type is model.CredentialTypeOfficialOAuth2, use same form as oauth2.
func (m *Meta) GetCredentialForm(credentialType string) (*CredentialForm, error) {
	credentialForm, ok := m.CredentialForms[credentialType]
	if !ok {
		return nil, fmt.Errorf("adapter %q not define form %q", m.Class, credentialType)
	}
	if credentialForm.InputForm == nil {
		return nil, fmt.Errorf("credential form %q not defined", credentialType)
	}
	return credentialForm, nil
}

func (f *CredentialForm) ValidateDynamically(inputFields map[string]string) (err error) {
	if f == nil || f.basicValidator == nil {
		return
	}
	newInputFields := map[string]any{}
	for k, v := range inputFields {
		newInputFields[k] = v
	}
	return f.basicValidator.validate(newInputFields)
}

// RegisterCredentialTemplate call in initialize, so panics if any error occurs.
func (m *Meta) RegisterCredentialTemplate(credentialType string, content string) {
	if _, ok := m.CredentialForms[credentialType]; !ok {
		panic(fmt.Sprintf("credential form %q not defined", credentialType))
	}

	tpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(content)
	if err != nil {
		panic(fmt.Errorf("template parse error: %v", err))
	}

	m.CredentialForms[credentialType].Template = tpl
}

// TestCredentialTemplate for user's testing, if something wrong happened, panics directly.
func (m *Meta) TestCredentialTemplate(credentialType string, data map[string]any) {
	form, ok := m.CredentialForms[credentialType]
	if !ok {
		panic(fmt.Sprintf("credential form %q not defined", credentialType))
	}
	var buf bytes.Buffer
	err := form.Template.Execute(&buf, data)
	if err != nil {
		panic(fmt.Errorf("executing template: %w", err))
	}

	raw := buf.Bytes()
	var result map[string]any
	err = json.Unmarshal(raw, &result)
	if err != nil {
		fmt.Println(string(raw))
		panic(fmt.Errorf("decoding json: %w", err))
	}
}

// AdapterManager is the adapter manager.
// TODO: make a global registry to register external adapters.
type AdapterManager struct {
	metas   []*Meta
	metaMap map[string]*Meta
	specMap map[string]*Spec
}

func (a *AdapterManager) GetSpecCountByType(t AdapterSpecType) (count int) {
	for _, meta := range a.metas {
		for _, spec := range meta.Specs {
			if spec.Type == t {
				count++
			}
		}
	}
	return
}

func (a *AdapterManager) GetIconsByClass(adapterClasses []string) []string {
	result := make([]string, 0, 8) // one size to fit them all
	for i, class := range adapterClasses {
		adapter := a.LookupAdapter(class)
		if adapter != nil {
			result = append(result, adapter.Icon)
			if i == 0 {
				result = append(result, "official://trend-static")
			}
		}
	}
	return result
}

// GetMetaCount returns the count of meta.
func (a *AdapterManager) GetMetaCount() int {
	return len(a.metas)
}

// GetMetas returns all adapters.
func (a *AdapterManager) GetMetas() []*Meta {
	return a.metas
}

// ListPresentData is the api response of list adapter.
type ListPresentData struct {
	AdapterList []Meta `json:"adapters"`
}

// FillDynamicFields fill the dynamic fields to the adapter.
func (a *AdapterManager) FillDynamicFields(oauth2CallbackURL string) {
	for _, meta := range a.metas {
		for supportCredential, form := range meta.CredentialForms {
			if supportCredential == OAuth2CredentialType {
				for i := range form.InputForm.Fields {
					if form.InputForm.Fields[i].Key == "redirectUrl" {
						form.InputForm.Fields[i].Default = oauth2CallbackURL
					}
				}

				for _, langForm := range form.langInputForm {
					for i := range langForm.Fields {
						if langForm.Fields[i].Key == "redirectUrl" {
							langForm.Fields[i].Default = oauth2CallbackURL
						}
					}
				}
			}
		}
	}
}

// Present for api render.
func (a *AdapterManager) Present(lang string) ListPresentData {
	metas := make([]Meta, 0, len(a.metas))
	for _, meta := range a.metas {
		newMeta := meta.copy()

		if lang != defaultLanguage {
			if _, ok := meta.langName[lang]; ok {
				newMeta.Name = meta.langName[lang]
			}
			if _, ok := meta.langDescription[lang]; ok {
				newMeta.Description = meta.langDescription[lang]
			}
		}

		var newSpecs []*Spec
		for _, spec := range meta.Specs {
			if spec.Hidden {
				continue
			}
			switch spec.Class {
			case "ultrafox/jira#createIssue", "ultrafox/jira#updateIssue":
				if !featureflag.IsEnabled(context.Background(), featureflag.AddJiraIssueActors, featureflag.ContextData{}) {
					continue
				}
			}

			newSpec := *spec

			if lang != defaultLanguage {
				if _, ok := newSpec.langInputSchema[lang]; ok {
					newSpec.InputSchema = newSpec.langInputSchema[lang]
				}
				if _, ok := newSpec.langOutputSchema[lang]; ok {
					newSpec.OutputSchema = newSpec.langOutputSchema[lang]
				}
				if _, ok := newSpec.langTestPageSchema[lang]; ok {
					newSpec.TestPageSchema = newSpec.langTestPageSchema[lang]
				}
			}

			if _, ok := newSpec.langBase[lang]; ok {
				newSpec.BaseSpec = newSpec.langBase[lang]
			}

			newSpecs = append(newSpecs, &newSpec)
		}

		// newMeta all fields same as original meta except Specs,
		// After filter out Hidden spec, should use new memory to store Specs,
		// The Internal spec is visible in UltraFox backend.
		newMeta.Specs = newSpecs
		if len(newMeta.Specs) == 0 {
			continue
		}

		if lang != defaultLanguage {
			for form, credentialForm := range newMeta.CredentialForms {
				if _, ok := credentialForm.langInputForm[lang]; !ok {
					continue
				}

				newMeta.CredentialForms[form] = &CredentialForm{
					InputForm: credentialForm.langInputForm[lang],
				}
			}
		}

		metas = append(metas, newMeta)
	}

	return ListPresentData{
		AdapterList: metas,
	}
}

// LookupAdapter returns the meta of the adapter by class name.
func (a *AdapterManager) LookupAdapter(class string) *Meta {
	return a.metaMap[class]
}

func (a *AdapterManager) LookupSpec(class string) *Spec {
	return a.specMap[class]
}

// SpecRequireAuth reports whether the specified class supports credential usage.
// The caller should ensure the provided class exists. if spec not defined,
// will return false.
func (a *AdapterManager) SpecRequireAuth(class string) bool {
	spec := a.LookupSpec(class)
	if spec == nil {
		return false
	}
	meta := a.LookupAdapter(spec.AdapterClass)
	if meta == nil {
		return false
	}
	return meta.RequireAuth()
}

// RegisterAdapter panics if an error is encountered.
func RegisterAdapter(m *Meta) {
	adapterManager.RegisterAdapter(m)
}

func (a *AdapterManager) RegisterAdapter(m *Meta) {
	if err := m.Validate(); err != nil {
		panic(err)
	}

	m.adapterManager = a
	m.adapterManager.metas = append(m.adapterManager.metas, m)

	if _, ok := m.adapterManager.metaMap[m.Class]; ok {
		panic(fmt.Errorf("adapter %q already registered", m.Class))
	}

	m.adapterManager.metaMap[m.Class] = m
}

func RegisterAdapterByRaw(raw []byte) *Meta {
	return adapterManager.RegisterAdapterByRaw(raw)
}

// RegisterAdapterByRaw panics if an error is encountered.
func (a *AdapterManager) RegisterAdapterByRaw(raw []byte) *Meta {
	internalMeta := &jsonMeta{}
	err := json.Unmarshal(raw, internalMeta)
	if err != nil {
		panic(fmt.Errorf("parse adapter by raw: %w", err))
	}

	forms := map[string]*CredentialForm{}
	for _, supportCredential := range internalMeta.SupportCredentials {
		var form *InputFormField

		v, ok := internalMeta.InputSchemas[supportCredential]
		if !ok {
			panic(fmt.Errorf("must define credential %q's input schema", supportCredential))
		}

		form = &InputFormField{
			AdvancedField: AdvancedField{
				Fields: v.Fields.ToDefault(),
			},
		}

		forms[supportCredential] = &CredentialForm{
			InputForm: form,
			Template:  getDefaultCredentialTemplate(supportCredential),
			Masker:    NewMasker(form.Fields),
			langInputForm: map[string]*InputFormField{
				defaultLanguage: {
					AdvancedField: AdvancedField{
						Fields: v.Fields.ToDefault(),
					},
				},
				"zh-CN": {
					AdvancedField: AdvancedField{
						Fields: v.Fields.ToLang("zh-CN"),
					},
				},
			},
			basicValidator: buildBasicValidator(form.Fields),
		}
	}

	meta := &Meta{
		BaseMeta:        internalMeta.BaseMeta,
		CredentialForms: forms,
		langName: map[string]string{
			"en-US": internalMeta.Name.GetLang("en-US"),
			"zh-CN": internalMeta.Name.GetLang("zh-CN"),
		},
		langDescription: map[string]string{
			"en-US": internalMeta.Description.GetLang("en-US"),
			"zh-CN": internalMeta.Description.GetLang("zh-CN"),
		},
	}
	meta.Name = internalMeta.Name.GetLang(defaultLanguage)
	meta.Description = internalMeta.Description.GetLang(defaultLanguage)

	a.RegisterAdapter(meta)

	return meta
}

// RegisterSpecsByDir register all json specsMap in the given directory,
// panic if an error is encountered.
func (m *Meta) RegisterSpecsByDir(dir fs.FS) {
	err := fs.WalkDir(dir, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if p == "." {
			return nil
		}

		if d.Type().IsDir() {
			return nil
		}

		p = path.Clean(p)
		if strings.HasSuffix(p, ".json") {
			err = m.registerSpec(dir, p)
			if err != nil {
				return fmt.Errorf("register spec %q: %w", p, err)
			}
		}

		return nil
	})

	if err != nil {
		panic(fmt.Errorf("register specsMap by dir: %w", err))
	}
}

func (m *Meta) registerSpec(dir fs.FS, p string) error {
	fileData, err := fs.ReadFile(dir, p)
	if err != nil {
		return fmt.Errorf("fs read file %s: %w", p, err)
	}

	return m.registerSpecByRaw(fileData)
}

func (m *Meta) RegisterSpecByRaw(raw []byte) {
	err := m.registerSpecByRaw(raw)
	if err != nil {
		panic(fmt.Errorf("register spec by raw: %w", err))
	}
}

func (m *Meta) registerSpecByRaw(raw []byte) (err error) {
	var internalSpec jsonSpec
	err = json.Unmarshal(raw, &internalSpec)
	if err != nil {
		return fmt.Errorf("parse internalSpec: %w", err)
	}

	if err = internalSpec.Validate(); err != nil {
		return fmt.Errorf("validate spec %q error: %w", internalSpec.Class, err)
	}

	if _, ok := m.adapterManager.specMap[internalSpec.Class]; ok {
		return fmt.Errorf("internalSpec %q already registered", internalSpec.Class)
	}

	if internalSpec.AdapterClass != m.Class {
		return fmt.Errorf("class %q adapterClass is wrong", internalSpec.Class)
	}

	spec := &Spec{
		BaseSpec:     internalSpec.ToDefault(),
		OutputSchema: internalSpec.OutputSchema.Fields.ToDefault(),
		InputSchema:  internalSpec.InputSchema.Fields.ToDefault(),
		AdapterMeta:  m,
		// multiple languages...
		langBase: map[string]BaseSpec{
			"en-US": internalSpec.ToDefault(),
			"zh-CN": internalSpec.ToLang("zh-CN"),
		},
		langOutputSchema: map[string]OutputFields{
			"en-US": internalSpec.OutputSchema.Fields.ToDefault(),
			"zh-CN": internalSpec.OutputSchema.Fields.ToLang("zh-CN"),
		},
		langInputSchema: map[string]InputFormFields{
			"en-US": internalSpec.InputSchema.Fields.ToDefault(),
			"zh-CN": internalSpec.InputSchema.Fields.ToLang("zh-CN"),
		},
		langTestPageSchema: map[string]InputFormFields{},
	}

	if len(internalSpec.TestPageSchema.Fields) > 0 {
		spec.TestPageSchema = internalSpec.TestPageSchema.Fields.ToDefault()
		spec.langTestPageSchema = map[string]InputFormFields{
			"en-US": internalSpec.TestPageSchema.Fields.ToDefault(),
			"zh-CN": internalSpec.TestPageSchema.Fields.ToLang("zh-CN"),
		}
	}

	// build validator
	spec.basicValidator = buildBasicValidator(spec.InputSchema)

	m.Specs = append(m.Specs, spec)
	m.adapterManager.specMap[internalSpec.Class] = spec

	return nil
}

func (m *Meta) Validate() (err error) {
	if m.Name == "" {
		return fmt.Errorf("adapter name cannot be empty")
	}
	if m.Class == "" {
		return fmt.Errorf("adapter class cannot be empty")
	}
	return
}

func (m *Meta) GetCredentialTemplate(credentialType string) (tpl *template.Template, err error) {
	form, ok := m.CredentialForms[credentialType]
	if !ok {
		err = fmt.Errorf("credential type %v not defined", credentialType)
		return
	}
	tpl = form.Template
	return
}

func (m *Meta) copy() (newMeta Meta) {
	newMeta = *m
	newMeta.CredentialForms = map[string]*CredentialForm{}
	for t, form := range m.CredentialForms {
		newMeta.CredentialForms[t] = form
	}
	return
}

func (m *Meta) RegisterCredentialTestingFunc(fn CredentialTestingFunc) {
	m.credentialTestingFunc = fn
}

func (m *Meta) TestCredential(ctx context.Context, credentialType model.CredentialType, fields map[string]string) (err error) {
	if m.credentialTestingFunc == nil {
		return
	}

	inputFields := make(model.InputFields, len(fields))
	for k, v := range fields {
		inputFields[k] = v
	}

	return m.credentialTestingFunc(ctx, credentialType, inputFields)
}

func getDefaultCredentialTemplate(t string) *template.Template {
	switch t {
	case OAuth2CredentialType:
		return defaultOAuth2Tpl
	case AccessTokenCredentialType:
		return defaultAccessTokenTpl
	case OAuthCredentialType:
		return defaultCustomTpl
	default:
		return defaultCustomTpl
	}
}
