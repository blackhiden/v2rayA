# Bug Fix Plan: Live Flow WebSocket Connected But No Data Received

## 1. Bug Report Summary

**Reported Issue:** Live Flow feature WebSocket connection succeeds but frontend never receives any data.

**Symptoms:**
- WebSocket upgrade succeeds (HTTP 101 Switching Protocols)
- Connection remains open
- No flow events received
- Dashboard remains empty

**User Evidence:**
```
Request URL: ws://localhost:2017/api/live-flow?token=...
Status code: 101 Switching Protocols
Result: No messages received
```

**Environment:** v2rayA on localhost:2017, Chrome on Linux

---

## 2. Root Cause

**Missing environment variable in core process startup.**

The v2raya_core binary emits Live Flow events ONLY when `V2RAYA_LIVE_FLOW=1` is set.
This variable is **never added** to the environment in `StartCoreProcess()`.

### Code Evidence

**File:** `core/xray/app/log/log.go` (line 126)
```go
func emitLiveFlowEvent(msg *log.AccessMessage) {
    if os.Getenv("V2RAYA_LIVE_FLOW") != "1" || msg == nil || msg.Status != log.AccessAccepted {
        return  // ← EXITS when env var not set
    }
    // ... emit event to stdout
}
```

**File:** `service/kernel/v2ray/process.go` (lines 250-305)
```go
func StartCoreProcess(ctx context.Context) (*os.Process, error) {
    // ... setup ...
    env = append(env, "XRAY_LOCATION_ASSET="+assetDir)
    if memMiB < 2048 {
        env = append(env, "V2RAY_CONF_GEOLOADER=memconservative")
    }
    // ← MISSING: env = append(env, "V2RAYA_LIVE_FLOW=1")
    proc, err := RunWithLog(ctx, v2rayBinPath, arguments, dir, env)
}
```

### What Happens

**Expected Pipeline:**
1. Core starts with V2RAYA_LIVE_FLOW=1
2. Core emits route events to stdout
3. Service parses events → LiveFlowProducer
4. WebSocket forwards to frontend

**Actual (Broken) Pipeline:**
1. Core starts WITHOUT V2RAYA_LIVE_FLOW
2. Core NEVER emits events (env check fails)
3. Service has no data
4. WebSocket connected but silent

---

## 3. Solution

**Add missing environment variable.**

### Implementation

**File:** `/home/denny/Project/v2rayA/service/kernel/v2ray/process.go`
**Location:** After line 297
**Add:**
```go
// Enable Live Flow route event emission
env = append(env, "V2RAYA_LIVE_FLOW=1")
```

**Complete context:**
```go
    if memMiB := memstat.Available / 1024 / 1024; memMiB < 2048 {
        env = append(env, "V2RAY_CONF_GEOLOADER=memconservative")
        log.Info("low memory: %vMiB, set V2RAY_CONF_GEOLOADER=memconservative", memMiB)
    }
}

// Enable Live Flow route event emission
env = append(env, "V2RAYA_LIVE_FLOW=1")

log.Debug(strings.Join(arguments, " "))
proc, err := RunWithLog(ctx, v2rayBinPath, arguments, dir, env)
```

---

## 4. Implementation Steps

### Step 1: Apply Fix
1. Edit `service/kernel/v2ray/process.go`
2. Add env variable after line 297
3. Save file

### Step 2: Build
```bash
cd /home/denny/Project/v2rayA/service
go build ./...
go vet ./...
```

### Step 3: Verify Environment
Check core process environment:
```bash
cat /proc/<PID>/environ | tr '\0' '\n' | grep V2RAYA_LIVE_FLOW
```
Expected: `V2RAYA_LIVE_FLOW=1`

### Step 4: Functional Test
1. Start service
2. Connect to proxy
3. Open Live Flow dashboard
4. Generate traffic: `curl https://example.com`
5. **Expected:**
   - batch_state on connection
   - flow_start for traffic
   - Dashboard shows topology

### Step 5: Run Tests
```bash
cd service
go test ./kernel/v2ray
```
All tests must pass.

---

## 5. Acceptance Criteria

- ✅ Environment variable set in core process
- ✅ Code compiles without errors
- ✅ Core emits event lines to stdout during traffic
- ✅ WebSocket receives batch_state, flow_start, flow_update messages
- ✅ Dashboard renders flow topology
- ✅ gstatic.com:443 probe excluded
- ✅ Sessions expire after 30s (flow_end sent)
- ✅ All unit tests pass
- ✅ No regression in proxy/load balancing/observatory
- ✅ Documentation updated

---

## 6. Why This Fix Is Correct

1. **Matches Design**: Core explicitly checks this variable
2. **Documented**: Multiple docs reference V2RAYA_LIVE_FLOW=1
   - docs/LIVE_FLOW_IMPLEMENTATION.md:38
   - docs/LIVE_FLOW_SESSION_RESUME.md:11,30
   - docs/MERGE-PRESERVATION.md:187
3. **Minimal**: Single line addition
4. **Safe**: No side effects on proxy behavior
5. **Compatible**: External cores ignore unknown variables

---

## 7. Risks & Mitigations

### Performance Impact
- **Risk:** Event emission overhead
- **Mitigation:** Events only on route selection, fast JSON, buffered output, probe exclusion

### Memory
- **Risk:** Session accumulation
- **Mitigation:** 30s TTL enforced, bounded map size

### Compatibility
- **Risk:** External cores don't support it
- **Mitigation:** Graceful degradation (empty feed, no crash)

---

## 8. Testing Plan

### Unit Tests
```bash
go test -v -run "TestParseLiveFlow|TestSelectedDetour|TestConsumeCoreL|TestRecordCore" ./kernel/v2ray
```

### Integration Tests
1. Basic flow reception
2. Multiple concurrent flows
3. Direct vs proxied routes
4. Session lifecycle (refresh/expiry)
5. WebSocket reconnection
6. No traffic scenario
7. High volume traffic
8. Core restart
9. External core (xray-core)
10. Performance impact

---

## 9. References

### Primary Files
- `service/kernel/v2ray/process.go` - Fix location
- `core/xray/app/log/log.go` - Event emission

### Pipeline Files
- `service/kernel/v2ray/liveflow.go` - Producer
- `service/server/service/liveflow.go` - WebSocket handler
- `service/server/controller/liveflow.go` - Controller
- `gui/src/plugins/liveflow.js` - Frontend client

### Documentation
- `docs/LIVE_FLOW_IMPLEMENTATION.md`
- `docs/ARCHITECTURE.md`
- `docs/MERGE-PRESERVATION.md`
- `CODE_GRAPH.md`
- `session.md`

---

## 10. Conclusion

**Root Cause:** Missing `V2RAYA_LIVE_FLOW=1` environment variable

**Solution:** Add one line: `env = append(env, "V2RAYA_LIVE_FLOW=1")`

**Risk:** Minimal (designed behavior, documented, tested)

**Impact:** Enables broken feature, no change to existing functionality

**Confidence:** Very High

**Recommendation:** Apply fix immediately.
