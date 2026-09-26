package groq

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientStreamsTextAndToolCalls(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintln(w, `data: {"id":"chat-1","choices":[{"delta":{"content":"hello"},"finish_reason":""}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, `data: {"id":"chat-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","function":{"name":"lookup","arguments":"{\"id\":"}}]},"finish_reason":"tool_calls"}]}`)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "data: [DONE]")
	}))
	defer server.Close()
	stream, err := NewClient(server.Client()).Generate(context.Background(), Config{APIKey: "secret", Endpoint: server.URL}, []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	var events []Event
	for event := range stream.Events() {
		events = append(events, event)
	}
	if len(events) != 3 || events[0].TextDelta != "hello" || events[1].ToolName != "lookup" || !events[2].Done {
		t.Fatalf("events = %+v", events)
	}
}


func TestStreamCloseUnblocksFullEventBuffer(t *testing.T) {
	reader, writer := io.Pipe()
	stream := newStream(context.Background(), reader)
	go stream.readLoop()

	go func() {
		defer func() { _ = writer.Close() }()
		for i := 0; i < 64; i++ {
			_, _ = fmt.Fprintf(writer, "data: {\"id\":\"chat-%d\",\"choices\":[{\"delta\":{\"content\":\"x\"},\"finish_reason\":\"\"}]}\n\n", i)
		}
	}()

	deadline := time.Now().Add(time.Second)
	for len(stream.events) < cap(stream.events) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(stream.events) != cap(stream.events) {
		t.Fatalf("event buffer = %d, want %d", len(stream.events), cap(stream.events))
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case <-stream.done:
	case <-time.After(time.Second):
		t.Fatal("Groq read loop did not stop after cancellation")
	}
}
