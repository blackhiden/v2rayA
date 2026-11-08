package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	clog "github.com/xtls/xray-core/common/log"
)

func TestLiveFlowWrapperEmitsEvent(t *testing.T) {
	os.Setenv("V2RAYA_LIVE_FLOW", "1")
	defer os.Unsetenv("V2RAYA_LIVE_FLOW")

	buf := new(bytes.Buffer)
	wrapper := &liveFlowWrapper{output: buf}

	msg := &clog.AccessMessage{
		From:   "192.168.1.100:54321",
		To:     "example.com:443",
		Status: clog.AccessAccepted,
		Detour: "proxy_sg",
	}

	wrapper.Handle(msg)

	out := buf.String()
	if !strings.HasPrefix(out, liveFlowEventPrefix) {
		t.Fatalf("expected prefix %q, got %q", liveFlowEventPrefix, out)
	}

	payload := strings.TrimPrefix(out, liveFlowEventPrefix)
	payload = strings.TrimSpace(payload)

	var evt liveFlowEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		t.Fatalf("failed to unmarshal emitted JSON: %v", err)
	}

	if evt.Source != "192.168.1.100:54321" {
		t.Errorf("unexpected source: %s", evt.Source)
	}
	if evt.Destination != "example.com:443" {
		t.Errorf("unexpected destination: %s", evt.Destination)
	}
	if evt.Detour != "proxy_sg" {
		t.Errorf("unexpected detour: %s", evt.Detour)
	}
	if evt.Timestamp == 0 {
		t.Errorf("expected non-zero timestamp")
	}
}

func TestLiveFlowWrapperIgnoresNonAccepted(t *testing.T) {
	os.Setenv("V2RAYA_LIVE_FLOW", "1")
	defer os.Unsetenv("V2RAYA_LIVE_FLOW")

	buf := new(bytes.Buffer)
	wrapper := &liveFlowWrapper{output: buf}

	msg := &clog.AccessMessage{
		From:   "192.168.1.100:54321",
		To:     "blocked.com:443",
		Status: clog.AccessRejected,
		Detour: "direct",
	}

	wrapper.Handle(msg)

	if buf.Len() != 0 {
		t.Fatalf("expected no output for rejected access, got %q", buf.String())
	}
}

func TestLiveFlowWrapperDisabledWithoutEnv(t *testing.T) {
	os.Unsetenv("V2RAYA_LIVE_FLOW")

	buf := new(bytes.Buffer)
	wrapper := &liveFlowWrapper{output: buf}

	msg := &clog.AccessMessage{
		From:   "192.168.1.100:54321",
		To:     "example.com:443",
		Status: clog.AccessAccepted,
		Detour: "proxy_sg",
	}

	wrapper.Handle(msg)

	if buf.Len() != 0 {
		t.Fatalf("expected no output when env var not set, got %q", buf.String())
	}
}
