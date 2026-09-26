package freeswitch

import "testing"

func TestAudioForkRequestValidate(t *testing.T) {
	valid := AudioForkRequest{
		ChannelID: "00000000-0000-0000-0000-000000000001", WebSocketURL: "ws://media:8090/v1/audio-forks?token=value",
		MixType: "mono", SampleRate: "16k",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*AudioForkRequest)
	}{
		{name: "channel", mutate: func(r *AudioForkRequest) { r.ChannelID = "" }},
		{name: "URL", mutate: func(r *AudioForkRequest) { r.WebSocketURL = "https://media.example.com" }},
		{name: "mix", mutate: func(r *AudioForkRequest) { r.MixType = "caller" }},
		{name: "rate", mutate: func(r *AudioForkRequest) { r.SampleRate = "32k" }},
		{name: "metadata", mutate: func(r *AudioForkRequest) { r.Metadata = "{" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := valid
			tt.mutate(&request)
			if err := request.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
