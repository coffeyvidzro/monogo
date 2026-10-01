package providers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type STTStream interface {
	SendAudio(context.Context, session.AudioFrame) error
	Finalize(context.Context) error
	Events() <-chan STTEvent
	Close(context.Context) error
}

type STT interface {
	Descriptor() Descriptor
	StartSTT(
		context.Context,
		Runtime,
		session.AudioFormat,
		string,
	) (STTStream, error)
}

type LLMStream interface {
	Events() <-chan LLMEvent
	Close() error
}

type LLM interface {
	Descriptor() Descriptor
	Generate(context.Context, LLMRequest) (LLMStream, error)
}

type TTSStream interface {
	SendText(context.Context, string, bool) error
	Events() <-chan TTSEvent
	Close() error
}

type TTS interface {
	Descriptor() Descriptor
	StartTTS(context.Context, TTSRequest) (TTSStream, error)
}

type Realtime interface {
	Descriptor() Descriptor
	StartRealtime(
		context.Context,
		Runtime,
		session.Config,
	) (session.Stream, error)
}

type Registry struct {
	descriptors map[string]Descriptor
}

func NewRegistry(descriptors ...Descriptor) (*Registry, error) {
	registry := &Registry{
		descriptors: make(map[string]Descriptor, len(descriptors)),
	}
	for _, descriptor := range descriptors {
		if err := registry.Register(descriptor); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(descriptor Descriptor) error {
	if r == nil {
		return fmt.Errorf("provider registry is required")
	}
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

	copyDescriptor := descriptor
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
