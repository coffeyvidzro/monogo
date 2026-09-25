package calling

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestClassifyClientOriginateError(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		class OriginateFailureClass
	}{
		{name: "deadline", err: context.DeadlineExceeded, class: OriginateFailureTimeout},
		{name: "transport", err: &net.OpError{Op: "write", Net: "tcp", Err: errors.New("reset")}, class: OriginateFailureTransport},
		{name: "canceled", err: context.Canceled, class: OriginateFailureInternal},
		{name: "internal", err: errors.New("rejected"), class: OriginateFailureInternal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got *OriginateError
			if !errors.As(classifyClientOriginateError(test.err), &got) || got.Class != test.class {
				t.Fatalf("classifyClientOriginateError() = %+v, want %s", got, test.class)
			}
		})
	}
}
