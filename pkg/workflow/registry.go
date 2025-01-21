package workflow

import (
	"fmt"
	"sync"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/trigger"
)

var registry *adapterRegistry

type adapterRegistry struct {
	sync.Mutex

	adapterManager   *adapter.AdapterManager
	nodeMetas        map[string]NodeMeta
	webhookProviders map[string]trigger.TriggerProvider
	sampleProviders  map[string]trigger.SampleProvider
}

func init() {
	registry = &adapterRegistry{
		adapterManager:   adapter.GetAdapterManager(),
		nodeMetas:        map[string]NodeMeta{},
		webhookProviders: map[string]trigger.TriggerProvider{},
		sampleProviders:  map[string]trigger.SampleProvider{},
	}

	registerInternalAdapters()
}

// WebhookProviders returns a shallow copy of
// registered webhook providers.
//
// The key is the adapter class.
func WebhookProviders() map[string]trigger.TriggerProvider {
	registry.Lock()
	defer registry.Unlock()

	m := make(map[string]trigger.TriggerProvider, len(registry.webhookProviders))

	for k, v := range registry.webhookProviders {
		m[k] = v
	}

	return m
}

// SampleProviders returns a shallow copy of
// registered webhook providers.
//
// The key is the adapter class.
func SampleProviders() map[string]trigger.SampleProvider {
	registry.Lock()
	defer registry.Unlock()

	m := make(map[string]trigger.SampleProvider, len(registry.sampleProviders))

	for k, v := range registry.sampleProviders {
		m[k] = v
	}

	return m
}

func GetNodeMeta(class string) (meta NodeMeta, ok bool) {
	registry.Lock()
	defer registry.Unlock()
	m, ok := registry.nodeMetas[class]
	return m, ok
}

func getInputSchema(class string) adapter.InputFormFields {
	registry.Lock()
	defer registry.Unlock()
	m, ok := registry.nodeMetas[class]
	if !ok {
		return nil
	}
	return m.InputForm
}

func RegistryNodeMeta(instance Node) {
	nodeMeta := instance.UltrafoxNode()

	spec := registry.adapterManager.LookupSpec(nodeMeta.Class)
	if spec == nil {
		panic(fmt.Errorf("class %v not registered", nodeMeta.Class))
	}

	if nodeMeta.New == nil {
		panic("missing node constructor")
	}
	if value := nodeMeta.New(); value == nil {
		panic("node constructor must return a non-nil instance")
	}

	node := nodeMeta.New()

	if spec.TriggerType == string(model.TriggerTypeWebhook) || spec.TriggerType == string(model.TriggerTypePoll) {
		_, ok := node.(trigger.TriggerProvider)
		if !ok {
			panic(fmt.Sprintf("node %s is not a trigger, trigger node should implement trigger.TriggerProvider", nodeMeta.Class))
		}
		registry.webhookProviders[nodeMeta.Class] = nodeMeta.New().(trigger.TriggerProvider)
	} else {
		if nodeMeta.InputForm == nil {
			panic("actor's inputForm required")
		}
	}

	if _, ok := node.(trigger.SampleProvider); ok {
		registry.sampleProviders[nodeMeta.Class] = nodeMeta.New().(trigger.SampleProvider)
	}

	registry.nodeMetas[nodeMeta.Class] = nodeMeta
}
