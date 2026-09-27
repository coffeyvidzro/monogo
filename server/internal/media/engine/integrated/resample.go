package integrated

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type rateAdaptedStream struct {
	ctx            context.Context
	cancel         context.CancelFunc
	provider       session.Stream
	inputFormat    session.AudioFormat
	outputFormat   session.AudioFormat
	providerFormat session.AudioFormat
	input          *pcm16RateConverter
	output         *pcm16RateConverter
	audio          chan session.AudioFrame
	done           chan struct{}
	inputMu        sync.Mutex
	closeOnce      sync.Once
}

func newRateAdaptedStream(
	ctx context.Context,
	provider session.Stream,
	inputFormat, outputFormat, providerFormat session.AudioFormat,
) (*rateAdaptedStream, error) {
	if provider == nil {
		return nil, fmt.Errorf("integrated provider stream is required")
	}
	input, err := newPCM16RateConverter(inputFormat.SampleRateHz, providerFormat.SampleRateHz)
	if err != nil {
		return nil, err
	}
	output, err := newPCM16RateConverter(providerFormat.SampleRateHz, outputFormat.SampleRateHz)
	if err != nil {
		return nil, err
	}
	streamCtx, cancel := context.WithCancel(ctx)
	s := &rateAdaptedStream{
		ctx:            streamCtx,
		cancel:         cancel,
		provider:       provider,
		inputFormat:    inputFormat,
		outputFormat:   outputFormat,
		providerFormat: providerFormat,
		input:          input,
		output:         output,
		audio:          make(chan session.AudioFrame, 32),
		done:           make(chan struct{}),
	}
	go s.relayAudio()
	return s, nil
}

func (s *rateAdaptedStream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	if frame.Format != s.inputFormat {
		return fmt.Errorf("integrated transport input format changed during stream")
	}

	s.inputMu.Lock()
	data, err := s.input.Convert(frame.Data)
	s.inputMu.Unlock()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return s.provider.SendAudio(ctx, session.AudioFrame{
		Data:       data,
		Format:     s.providerFormat,
		CapturedAt: frame.CapturedAt,
	})
}

func (s *rateAdaptedStream) Interrupt(ctx context.Context) error {
	return s.provider.Interrupt(ctx)
}

func (s *rateAdaptedStream) Audio() <-chan session.AudioFrame { return s.audio }
func (s *rateAdaptedStream) Events() <-chan session.Event     { return s.provider.Events() }

func (s *rateAdaptedStream) Close(ctx context.Context) error {
	s.closeOnce.Do(s.cancel)
	providerErr := s.provider.Close(ctx)
	select {
	case <-s.done:
		return providerErr
	case <-ctx.Done():
		if providerErr != nil {
			return providerErr
		}
		return ctx.Err()
	}
}

func (s *rateAdaptedStream) relayAudio() {
	defer close(s.done)
	defer close(s.audio)
	for {
		select {
		case frame, ok := <-s.provider.Audio():
			if !ok {
				return
			}
			if frame.Format != s.providerFormat {
				return
			}
			data, err := s.output.Convert(frame.Data)
			if err != nil {
				return
			}
			if len(data) == 0 {
				continue
			}
			frame.Data = data
			frame.Format = s.outputFormat
			select {
			case s.audio <- frame:
			case <-s.ctx.Done():
				return
			}
		case <-s.ctx.Done():
			return
		}
	}
}

type pcm16RateConverter struct {
	inputRate   int64
	outputRate  int64
	phase       int64
	previous    int16
	initialized bool
}

func newPCM16RateConverter(inputRate, outputRate int) (*pcm16RateConverter, error) {
	if inputRate <= 0 || outputRate <= 0 {
		return nil, fmt.Errorf("PCM16 sample rates must be positive")
	}
	return &pcm16RateConverter{inputRate: int64(inputRate), outputRate: int64(outputRate)}, nil
}

func (c *pcm16RateConverter) Convert(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("PCM16 audio is empty")
	}
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("PCM16 audio has odd byte length %d", len(data))
	}
	if c.inputRate == c.outputRate {
		return append([]byte(nil), data...), nil
	}

	samples := len(data) / 2
	estimated := (int64(samples)*c.outputRate + c.inputRate - 1) / c.inputRate
	output := make([]byte, 0, int(estimated)*2)
	appendSample := func(sample int16) {
		var encoded [2]byte
		binary.LittleEndian.PutUint16(encoded[:], uint16(sample))
		output = append(output, encoded[:]...)
	}
	readSample := func(index int) int16 {
		return int16(binary.LittleEndian.Uint16(data[index*2 : index*2+2]))
	}

	index := 0
	if !c.initialized {
		c.previous = readSample(0)
		c.initialized = true
		index = 1
	}
	for ; index < samples; index++ {
		current := readSample(index)
		for c.phase <= c.outputRate {
			delta := int64(current) - int64(c.previous)
			value := int64(c.previous) + delta*c.phase/c.outputRate
			appendSample(int16(value))
			c.phase += c.inputRate
		}
		c.phase -= c.outputRate
		c.previous = current
	}
	return output, nil
}
