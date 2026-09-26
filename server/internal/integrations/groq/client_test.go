package groq

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientStreamsTextAndToolCalls(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"id":"chat-1","choices":[{"delta":{"content":"hello"},"finish_reason":""}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"id":"chat-1","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","function":{"name":"lookup","arguments":"{\"id\":"}}]},"finish_reason":"tool_calls"}]}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "data: [DONE]")
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
