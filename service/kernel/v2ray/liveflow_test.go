package v2ray

import (
	"testing"
	"time"
)

func TestParseLiveFlowEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		valid    bool
		ip       string
		port     int
		protocol string
	}{
		{name: "tcp ipv4", raw: "tcp:192.0.2.10:443", valid: true, ip: "192.0.2.10", port: 443, protocol: "tcp"},
		{name: "udp ipv6", raw: "udp:[2001:db8::1]:53", valid: true, ip: "2001:db8::1", port: 53, protocol: "udp"},
		{name: "http url", raw: "https://example.com:8443/path", valid: true, ip: "example.com", port: 8443},
		{name: "empty", raw: "", valid: false},
		{name: "bad port", raw: "tcp:example.com:70000", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoint, protocol, ok := parseLiveFlowEndpoint(test.raw)
			if ok != test.valid {
				t.Fatalf("validity = %v, want %v", ok, test.valid)
			}
			if !test.valid {
				return
			}
			if endpoint.IP != test.ip || endpoint.Port != test.port || protocol != test.protocol {
				t.Fatalf("endpoint = %#v, protocol = %q", endpoint, protocol)
			}
		})
	}
}

func TestSelectedDetourTag(t *testing.T) {
	tests := map[string]string{
		"":                      "",
		"proxy":                 "proxy",
		"socks -> 『Tokyo-01』":   "Tokyo-01",
		"socks ==> 『Tokyo-01』":  "Tokyo-01",
		"transparent >> direct": "direct",
	}
	for input, want := range tests {
		if got := selectedDetourTag(input); got != want {
			t.Errorf("selectedDetourTag(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestConsumeCoreLiveFlowEvent(t *testing.T) {
	if !ConsumeCoreLiveFlowEvent(`V2RAYA_LIVE_FLOW {"source":"tcp:192.0.2.1:1111","destination":"tcp:example.com:443","detour":"direct","timestamp":1}`) {
		t.Fatal("valid live flow event was not consumed")
	}
	if !ConsumeCoreLiveFlowEvent("V2RAYA_LIVE_FLOW not-json") {
		t.Fatal("malformed private event was not consumed")
	}
	if ConsumeCoreLiveFlowEvent("ordinary core log") {
		t.Fatal("ordinary core log was consumed")
	}
	// Standard Xray access log line should be recorded and return false (not swallowed from normal logs)
	if ConsumeCoreLiveFlowEvent("2026/11/08 11:03:55 from tcp:127.0.0.1:58946 accepted tcp:appsgenaiserver-pa.clients6.google.com:443 [socks_ipv4 -> proxy]") {
		t.Fatal("standard access log line should not be swallowed (must return false)")
	}
	if ConsumeCoreLiveFlowEvent("from tcp:127.0.0.1:58946 accepted tcp:appsgenaiserver-pa.clients6.google.com:443 [socks_ipv4 -> proxy]") {
		t.Fatal("standard access log line without timestamp should return false")
	}
}

func TestParseAccessLogLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantOk  bool
		wantSrc string
		wantDst string
		wantDet string
	}{
		{
			name:    "with subsecond timestamp",
			line:    "2026/11/08 11:03:55.736 from tcp:127.0.0.1:58946 accepted tcp:appsgenaiserver-pa.clients6.google.com:443 [socks_ipv4 -> proxy]",
			wantOk:  true,
			wantSrc: "tcp:127.0.0.1:58946",
			wantDst: "tcp:appsgenaiserver-pa.clients6.google.com:443",
			wantDet: "socks_ipv4 -> proxy",
		},
		{
			name:    "without timestamp",
			line:    "from tcp:127.0.0.1:58946 accepted tcp:claude.ai:443 [socks_ipv4 -> proxy]",
			wantOk:  true,
			wantSrc: "tcp:127.0.0.1:58946",
			wantDst: "tcp:claude.ai:443",
			wantDet: "socks_ipv4 -> proxy",
		},
		{
			name:    "no detour",
			line:    "from 192.168.1.1:1234 accepted tcp:example.com:80",
			wantOk:  true,
			wantSrc: "192.168.1.1:1234",
			wantDst: "tcp:example.com:80",
			wantDet: "",
		},
		{
			name:   "rejected connection",
			line:   "from 192.168.1.1:1234 rejected tcp:example.com:80",
			wantOk: false,
		},
		{
			name:   "non access log",
			line:   "2026/11/08 11:03:55 [Warning] some core warning",
			wantOk: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evt, ok := parseAccessLogLine(tc.line)
			if ok != tc.wantOk {
				t.Fatalf("parseAccessLogLine() ok = %v, want %v", ok, tc.wantOk)
			}
			if tc.wantOk {
				if evt.Source != tc.wantSrc || evt.Destination != tc.wantDst || evt.Detour != tc.wantDet {
					t.Fatalf("parsed = %+v, want src=%q dst=%q det=%q", evt, tc.wantSrc, tc.wantDst, tc.wantDet)
				}
			}
		})
	}
}

func TestRecordCoreAccessExcludesGstaticProbe(t *testing.T) {
	feed := NewSubscriptions(2)
	feed.RegisterProduct(LiveFlowProduct)
	producer := &LiveFlowProducer{
		feed:     feed,
		sessions: make(map[string]*LiveFlowSession),
		done:     make(chan struct{}),
	}
	defer producer.Stop()

	box := feed.SubscribeMessage(LiveFlowProduct)
	defer box.Cancel()

	producer.RecordCoreAccess(CoreLiveFlowEvent{
		Source:      "tcp:192.0.2.1:1111",
		Destination: "tcp:gstatic.com:443",
		Detour:      "direct",
		Timestamp:   time.Now().UnixMilli(),
	})
	if sessions := producer.Snapshot(); len(sessions) != 0 {
		t.Fatalf("probe created %d Live Flow session(s), want 0", len(sessions))
	}
	select {
	case message := <-box.Messages:
		t.Fatalf("probe published Live Flow event %#v", message)
	default:
	}

	producer.RecordCoreAccess(CoreLiveFlowEvent{
		Source:      "tcp:192.0.2.1:1111",
		Destination: "tcp:gstatic.com:80",
		Detour:      "direct",
		Timestamp:   time.Now().UnixMilli(),
	})
	select {
	case message := <-box.Messages:
		event, ok := message.Body.(LiveFlowEvent)
		if !ok || event.Type != "flow_start" {
			t.Fatalf("normal route message = %#v, want flow_start", message)
		}
	case <-time.After(time.Second):
		t.Fatal("normal route did not publish a Live Flow event")
	}
}
