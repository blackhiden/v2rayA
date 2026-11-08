package service

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/v2rayA/v2rayA/kernel/v2ray"
	"github.com/v2rayA/v2rayA/pkg/util/log"
)

// FlowEndpoint identifies an endpoint in the live-flow WebSocket schema.
type FlowEndpoint struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Domain string `json:"domain,omitempty"`
}

// ProxyNode identifies a selected server or another proxy-chain hop.
type ProxyNode struct {
	ProxyID string `json:"proxy_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Server  string `json:"server"`
}

// User represents user information when a core event makes it available.
type User struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// LiveFlowSession represents a live proxy session.
type LiveFlowSession struct {
	SessionID   string            `json:"session_id"`
	Source      FlowEndpoint      `json:"source"`
	ProxyChain  []ProxyNode       `json:"proxy_chain"`
	Destination FlowEndpoint      `json:"destination"`
	Protocol    string            `json:"protocol"`
	StartTime   time.Time         `json:"start_time"`
	LastUpdate  time.Time         `json:"last_update"`
	User        *User             `json:"user,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	BytesSent   uint64            `json:"bytes_sent"`
	BytesRecv   uint64            `json:"bytes_recv"`
	SpeedBPS    uint64            `json:"speed_bps"`
	Status      string            `json:"status"`
}

// LiveFlowMessage is the WebSocket protocol envelope.
type LiveFlowMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// FlowStartData represents flow_start data.
type FlowStartData struct {
	SessionID   string            `json:"session_id"`
	Source      FlowEndpoint      `json:"source"`
	ProxyChain  []ProxyNode       `json:"proxy_chain"`
	Destination FlowEndpoint      `json:"destination"`
	Protocol    string            `json:"protocol"`
	StartTime   time.Time         `json:"start_time"`
	User        *User             `json:"user,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FlowUpdateData represents flow_update data.
type FlowUpdateData struct {
	SessionID    string `json:"session_id"`
	SpeedBPS     uint64 `json:"speed_bps"`
	BytesSent    uint64 `json:"bytes_sent"`
	BytesRecv    uint64 `json:"bytes_recv"`
	LastActivity string `json:"last_activity"`
	Status       string `json:"status"`
}

// FlowEndData represents flow_end data.
type FlowEndData struct {
	SessionID      string  `json:"session_id"`
	EndTime        string  `json:"end_time"`
	TotalBytesSent uint64  `json:"total_bytes_sent"`
	TotalBytesRecv uint64  `json:"total_bytes_recv"`
	DurationSecs   float64 `json:"duration_seconds"`
	EndReason      string  `json:"end_reason"`
}

// BatchStateData represents the initial state sent to a connected client.
type BatchStateData struct {
	Sessions    []FlowStartData `json:"sessions"`
	Timestamp   string          `json:"timestamp"`
	TotalActive int             `json:"total_active"`
}

const (
	liveFlowWriteWait = 10 * time.Second
	liveFlowPongWait  = 60 * time.Second
	liveFlowPingEvery = 30 * time.Second
	liveFlowMaxRead   = 1024
)

// LiveFlowHandler bridges the shared kernel live_flow feed to one WebSocket.
type LiveFlowHandler struct {
	conn       *websocket.Conn
	send       chan interface{}
	Done       chan struct{}
	stopOnce   sync.Once
	feedBox    *v2ray.Box
	feedCancel func()
}

// NewLiveFlowHandler creates a handler for one authenticated client.
func NewLiveFlowHandler(conn *websocket.Conn) *LiveFlowHandler {
	return &LiveFlowHandler{
		conn: conn,
		send: make(chan interface{}, 256),
		Done: make(chan struct{}),
	}
}

// Start subscribes to real route-session events and starts the read/write pumps.
func (h *LiveFlowHandler) Start() {
	if h.subscribeToLiveFlowFeed() {
		// Queue the snapshot before forwarding the subscription. A route that
		// arrives between subscription and snapshot may be duplicated, but it
		// cannot be lost when a later stale batch overwrites client state.
		h.SendBatchState()
		h.forwardLiveFlowFeed()
	}
	go h.writePump()
	go h.readPump()
}

// SendBatchState sends active route sessions accumulated before connection.
func (h *LiveFlowHandler) SendBatchState() {
	producer := v2ray.GetLiveFlowProducer()
	producer.Start()

	snapshot := producer.Snapshot()
	sessions := make([]FlowStartData, 0, len(snapshot))
	for _, session := range snapshot {
		sessions = append(sessions, flowStartDataFromKernel(session))
	}

	h.enqueue(LiveFlowMessage{
		Type: "batch_state",
		Data: BatchStateData{
			Sessions:    sessions,
			Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
			TotalActive: len(sessions),
		},
	})
}

// AddSession remains available for services that originate a complete flow
// directly. Kernel route events use the shared producer path above.
func (h *LiveFlowHandler) AddSession(session *LiveFlowSession) {
	if session == nil {
		return
	}
	h.enqueue(LiveFlowMessage{
		Type: "flow_start",
		Data: FlowStartData{
			SessionID:   session.SessionID,
			Source:      session.Source,
			ProxyChain:  session.ProxyChain,
			Destination: session.Destination,
			Protocol:    session.Protocol,
			StartTime:   session.StartTime,
			User:        session.User,
			Metadata:    session.Metadata,
		},
	})
}

// UpdateSession emits a session update. The caller is responsible for deriving
// counters from a source that actually exposes per-session traffic statistics.
func (h *LiveFlowHandler) UpdateSession(sessionID string, bytesSent, bytesRecv, speedBPS uint64, status string) {
	h.enqueue(LiveFlowMessage{
		Type: "flow_update",
		Data: FlowUpdateData{
			SessionID:    sessionID,
			SpeedBPS:     speedBPS,
			BytesSent:    bytesSent,
			BytesRecv:    bytesRecv,
			LastActivity: time.Now().UTC().Format(time.RFC3339Nano),
			Status:       status,
		},
	})
}

// EndSession emits a session end event.
func (h *LiveFlowHandler) EndSession(sessionID string, reason string) {
	h.enqueue(LiveFlowMessage{
		Type: "flow_end",
		Data: FlowEndData{
			SessionID: sessionID,
			EndTime:   time.Now().UTC().Format(time.RFC3339Nano),
			EndReason: reason,
		},
	})
}

// GenerateSessionID generates a unique session ID for direct producers.
func GenerateSessionID() string {
	return uuid.New().String()
}

func (h *LiveFlowHandler) subscribeToLiveFlowFeed() bool {
	producer := v2ray.GetLiveFlowProducer()
	producer.Start()

	if v2ray.ApiFeed != nil {
		v2ray.ApiFeed.RegisterProduct(v2ray.LiveFlowProduct)
	}

	box := v2ray.ApiFeed.SubscribeMessage(v2ray.LiveFlowProduct)
	if box == nil {
		log.Error("LiveFlow: failed to subscribe to live-flow feed")
		return false
	}
	h.feedBox = box
	h.feedCancel = box.Cancel
	return true
}

func (h *LiveFlowHandler) forwardLiveFlowFeed() {
	if h.feedBox == nil {
		return
	}
	go func() {
		for message := range h.feedBox.Messages {
			h.enqueue(message.Body)
		}
	}()
}

func (h *LiveFlowHandler) enqueue(message interface{}) {
	select {
	case <-h.Done:
		return
	case h.send <- message:
	default:
		// Dropping a stale update is preferable to blocking the shared feed or
		// retaining unbounded traffic metadata for a slow browser.
	}
}

func (h *LiveFlowHandler) readPump() {
	defer h.Stop()
	h.conn.SetReadLimit(liveFlowMaxRead)
	_ = h.conn.SetReadDeadline(time.Now().Add(liveFlowPongWait))
	h.conn.SetPongHandler(func(string) error {
		return h.conn.SetReadDeadline(time.Now().Add(liveFlowPongWait))
	})
	for {
		if _, _, err := h.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *LiveFlowHandler) writePump() {
	ticker := time.NewTicker(liveFlowPingEvery)
	defer func() {
		ticker.Stop()
		h.Stop()
	}()

	for {
		select {
		case <-h.Done:
			return
		case message := <-h.send:
			_ = h.conn.SetWriteDeadline(time.Now().Add(liveFlowWriteWait))
			if err := h.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			_ = h.conn.SetWriteDeadline(time.Now().Add(liveFlowWriteWait))
			if err := h.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Stop releases the feed subscription and closes the WebSocket exactly once.
func (h *LiveFlowHandler) Stop() {
	h.stopOnce.Do(func() {
		close(h.Done)
		if h.feedCancel != nil {
			h.feedCancel()
		}
		_ = h.conn.Close()
	})
}

func flowStartDataFromKernel(session v2ray.LiveFlowStartData) FlowStartData {
	chain := make([]ProxyNode, 0, len(session.ProxyChain))
	for _, proxy := range session.ProxyChain {
		chain = append(chain, ProxyNode{
			ProxyID: proxy.ProxyID,
			Name:    proxy.Name,
			Type:    proxy.Type,
			Server:  proxy.Server,
		})
	}
	return FlowStartData{
		SessionID: session.SessionID,
		Source: FlowEndpoint{
			IP:     session.Source.IP,
			Port:   session.Source.Port,
			Domain: session.Source.Domain,
		},
		ProxyChain: chain,
		Destination: FlowEndpoint{
			IP:     session.Destination.IP,
			Port:   session.Destination.Port,
			Domain: session.Destination.Domain,
		},
		Protocol:  session.Protocol,
		StartTime: session.StartTime,
	}
}
