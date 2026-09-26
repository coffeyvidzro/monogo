package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Attacher interface {
	Attach(context.Context, session.Connection) error
}

type WebSocketConfig struct {
	HandshakeTimeout time.Duration
	ReadLimit        int64
}

func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{HandshakeTimeout: 5 * time.Second, ReadLimit: 64 << 10}
}

type WebSocketHandler struct {
	tokens   *TokenService
	attacher Attacher
	config   WebSocketConfig
}

func NewWebSocketHandler(tokens *TokenService, attacher Attacher, cfg WebSocketConfig) (*WebSocketHandler, error) {
	if tokens == nil {
		return nil, fmt.Errorf("media token service is required")
	}
	if attacher == nil {
		return nil, fmt.Errorf("media session attacher is required")
	}
	if cfg.HandshakeTimeout <= 0 {
		return nil, fmt.Errorf("media handshake timeout must be positive")
	}
	if cfg.ReadLimit <= 0 {
		return nil, fmt.Errorf("media WebSocket read limit must be positive")
	}
	return &WebSocketHandler{tokens: tokens, attacher: attacher, config: cfg}, nil
}

func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, err := h.tokens.VerifyAndConsume(r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		return
	}
	ws.SetReadLimit(h.config.ReadLimit)

	ctx, cancel := context.WithTimeout(r.Context(), h.config.HandshakeTimeout)
	messageType, payload, err := ws.Read(ctx)
	cancel()
	if err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "hello required")
		return
	}
	metadata, err := parseHello(messageType, payload, claims, r.RemoteAddr)
	if err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "invalid hello")
		return
	}

	connection := &webSocketConnection{connection: ws, metadata: metadata}
	if err := h.attacher.Attach(r.Context(), connection); err != nil {
		_ = ws.Close(websocket.StatusPolicyViolation, "session rejected")
		return
	}
	_ = connection.Close()
}

type audioForkHello struct {
	Type     string         `json:"type"`
	CallID   string         `json:"callSid"`
	Rate     int            `json:"rate"`
	Channels int            `json:"channels"`
	Encoding string         `json:"encoding"`
	Metadata map[string]any `json:"metadata"`
}

func parseHello(messageType websocket.MessageType, payload []byte, claims TokenClaims, remoteAddress string) (session.ConnectionMetadata, error) {
	if messageType != websocket.MessageText {
		return session.ConnectionMetadata{}, fmt.Errorf("first media frame must be text")
	}
	var hello audioForkHello
	if err := json.Unmarshal(payload, &hello); err != nil {
		return session.ConnectionMetadata{}, fmt.Errorf("decode media hello: %w", err)
	}
	if hello.Type != "hello" {
		return session.ConnectionMetadata{}, fmt.Errorf("first media message must be hello")
	}
	if hello.CallID != claims.ChannelID.String() {
		return session.ConnectionMetadata{}, fmt.Errorf("media hello channel id does not match token")
	}
	if !strings.EqualFold(hello.Encoding, "L16") {
		return session.ConnectionMetadata{}, fmt.Errorf("unsupported media encoding %q", hello.Encoding)
	}
	format := session.AudioFormat{SampleRateHz: hello.Rate, Channels: hello.Channels}
	if err := format.Validate(); err != nil {
		return session.ConnectionMetadata{}, err
	}
	return session.ConnectionMetadata{
		SessionID: claims.SessionID, CallID: claims.CallID, ChannelID: claims.ChannelID,
		OrganizationID: claims.OrganizationID,
		RemoteAddress:  remoteAddress, Format: format,
	}, nil
}

type webSocketConnection struct {
	connection *websocket.Conn
	metadata   session.ConnectionMetadata
	writeMu    sync.Mutex
	closeOnce  sync.Once
}

func (c *webSocketConnection) Metadata() session.ConnectionMetadata { return c.metadata }

func (c *webSocketConnection) ReceiveAudio(ctx context.Context) (session.AudioFrame, error) {
	messageType, payload, err := c.connection.Read(ctx)
	if err != nil {
		if status := websocket.CloseStatus(err); status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
			return session.AudioFrame{}, io.EOF
		}
		return session.AudioFrame{}, err
	}
	if messageType != websocket.MessageBinary {
		return session.AudioFrame{}, fmt.Errorf("unexpected media text frame")
	}
	frame := session.AudioFrame{Data: payload, Format: c.metadata.Format, CapturedAt: time.Now().UTC()}
	if err := frame.Validate(); err != nil {
		return session.AudioFrame{}, err
	}
	return frame, nil
}

func (c *webSocketConnection) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	if frame.Format != c.metadata.Format {
		return fmt.Errorf("output audio format does not match negotiated format")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.connection.Write(ctx, websocket.MessageBinary, frame.Data)
}

func (c *webSocketConnection) Close() error {
	var err error
	c.closeOnce.Do(func() { err = c.connection.CloseNow() })
	return err
}
