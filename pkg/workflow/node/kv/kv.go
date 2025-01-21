package kv

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapterMeta := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapterMeta.RegisterSpecsByDir(adapterDir)

	workflow.RegistryNodeMeta(&PersistentKVSet{})
	workflow.RegistryNodeMeta(&PersistentKVGet{})
}

type basePersistentKV struct {
	workflow.KVProcessor
}

func (b *basePersistentKV) Provision(ctx context.Context, dependencies workflow.ProvisionDeps) (err error) {
	b.KVProcessor = dependencies.KVProcessor
	return
}

type PersistentKVGet struct {
	basePersistentKV

	// key to the value
	Key string `json:"key"`
	// take missing key as error?
	MissingAsError bool `json:"missingAsError"`
}

func (p *PersistentKVGet) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/persistentKV#get")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(PersistentKVGet)
		},
		InputForm: spec.InputSchema,
	}
}

func (p *PersistentKVGet) Run(c *workflow.NodeContext) (any, error) {
	value, err := p.KVProcessor.Get(c.Context(), p.Key)
	if err == nil {
		return value, nil
	}
	if errors.Is(err, workflow.ErrKVKeyNotExisted) && !p.MissingAsError {
		return nil, nil
	}
	err = fmt.Errorf("kv get: %w", err)
	return nil, err
}

type PersistentKVSet struct {
	basePersistentKV

	// key to the value
	Key string
	// value to set
	Value any
	// across workflows?
	Shared bool
}

func (p *PersistentKVSet) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/persistentKV#set")
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(PersistentKVSet)
		},
		InputForm: spec.InputSchema,
	}
}

func (p *PersistentKVSet) Run(c *workflow.NodeContext) (any, error) {
	err := p.KVProcessor.Set(c.Context(), p.Key, p.Value)
	if err != nil {
		err = fmt.Errorf("setting kv: %w", err)
		return nil, err
	}

	return p.Value, nil
}
