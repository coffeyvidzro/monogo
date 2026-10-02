package providers

import (
	"fmt"
	"sort"
	"strings"
)

type Registry struct {
	descriptors map[string]Descriptor
	stt         map[string]STT
	llm         map[string]LLM
	tts         map[string]TTS
	realtime    map[string]Realtime
}

func NewRegistry(providers ...Provider) (*Registry, error) {
	registry := &Registry{
		descriptors: make(map[string]Descriptor, len(providers)),
		stt:         make(map[string]STT),
		llm:         make(map[string]LLM),
		tts:         make(map[string]TTS),
		realtime:    make(map[string]Realtime),
	}
	for _, provider := range providers {
		if err := registry.Register(provider); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(provider Provider) error {
	if r == nil {
		return fmt.Errorf("provider registry is required")
	}
	if provider == nil {
		return fmt.Errorf("provider implementation is required")
	}

	descriptor := provider.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return err
	}

	key := registryKey(descriptor.Kind, descriptor.ID)
	if _, exists := r.descriptors[key]; exists {
		return fmt.Errorf(
			"provider %q already registered for %q",
			descriptor.ID,
			descriptor.Kind,
		)
	}

	id := strings.TrimSpace(descriptor.ID)
	switch descriptor.Kind {
	case KindSTT:
		implementation, ok := provider.(STT)
		if !ok {
			return fmt.Errorf("provider %q does not implement STT", id)
		}
		r.stt[id] = implementation
	case KindLLM:
		implementation, ok := provider.(LLM)
		if !ok {
			return fmt.Errorf("provider %q does not implement LLM", id)
		}
		r.llm[id] = implementation
	case KindTTS:
		implementation, ok := provider.(TTS)
		if !ok {
			return fmt.Errorf("provider %q does not implement TTS", id)
		}
		r.tts[id] = implementation
	case KindRealtime:
		implementation, ok := provider.(Realtime)
		if !ok {
			return fmt.Errorf("provider %q does not implement realtime", id)
		}
		r.realtime[id] = implementation
	default:
		return fmt.Errorf("unsupported provider kind %q", descriptor.Kind)
	}

	copyDescriptor := descriptor
	copyDescriptor.ID = id
	copyDescriptor.Capabilities = append(
		[]Capability(nil),
		descriptor.Capabilities...,
	)
	r.descriptors[key] = copyDescriptor
	return nil
}

func (r *Registry) Get(kind Kind, id string) (Descriptor, bool) {
	if r == nil {
		return Descriptor{}, false
	}

	descriptor, ok := r.descriptors[registryKey(kind, id)]
	if !ok {
		return Descriptor{}, false
	}

	descriptor.Capabilities = append(
		[]Capability(nil),
		descriptor.Capabilities...,
	)
	return descriptor, true
}

func (r *Registry) STT(id string) (STT, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.stt[strings.TrimSpace(id)]
	return provider, ok
}

func (r *Registry) LLM(id string) (LLM, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.llm[strings.TrimSpace(id)]
	return provider, ok
}

func (r *Registry) TTS(id string) (TTS, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.tts[strings.TrimSpace(id)]
	return provider, ok
}

func (r *Registry) Realtime(id string) (Realtime, bool) {
	if r == nil {
		return nil, false
	}
	provider, ok := r.realtime[strings.TrimSpace(id)]
	return provider, ok
}

func (r *Registry) List(kind Kind) []Descriptor {
	if r == nil {
		return nil
	}

	result := make([]Descriptor, 0)
	for _, descriptor := range r.descriptors {
		if descriptor.Kind != kind {
			continue
		}

		copyDescriptor := descriptor
		copyDescriptor.Capabilities = append(
			[]Capability(nil),
			descriptor.Capabilities...,
		)
		result = append(result, copyDescriptor)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

func registryKey(kind Kind, id string) string {
	return string(kind) + ":" + strings.TrimSpace(id)
}
