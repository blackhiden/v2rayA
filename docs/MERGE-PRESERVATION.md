# MERGE-PRESERVATION.md: Local Feature Preservation Guide

## 1. Purpose & Scope

This document serves as the authoritative protection manifest for all features, architectural enhancements, and safeguards developed locally in this repository that **do not exist in the upstream project** (`https://github.com/v2rayA/v2rayA`).

During synchronization, merging, or rebasing with `upstream`, upstream changes will frequently touch common files (such as template generators, route handlers, DB config, and GUI components). Without strict preservation, upstream merges can silently overwrite, disable, or degrade these local enhancements.

**Binding Directive:**  
Anyone performing an upstream sync (human developer, automation script, or AI agent) **must consult this document before, during, and after merging**. No upstream merge is complete until every feature cataloged below is verified to be fully intact and functional.

---

## 2. Upstream Merge Protocol & Standard Operating Procedure (SOP)

Follow this rigorous sequence whenever synchronizing upstream changes:

```text
               ┌───────────────────────────────┐
               │    1. Pre-Merge Preparation    │
               │  - Check git status & branch  │
               │  - Backup local dev database  │
               └──────────────┬────────────────┘
                              │
                              ▼
               ┌───────────────────────────────┐
               │    2. Fetch & Inspect Diff    │
               │  - git fetch upstream         │
               │  - git log / git diff inspect │
               └──────────────┬────────────────┘
                              │
                              ▼
               ┌───────────────────────────────┐
               │    3. Execute Merge/Rebase    │
               │  - Merge into develop branch  │
               │  - Identify conflict files    │
               └──────────────┬────────────────┘
                              │
                              ▼
               ┌───────────────────────────────┐
               │ 4. Reconcile Conflicts & Port │
               │  - Apply preservation rules   │
               │  - Restore local signatures   │
               └──────────────┬────────────────┘
                              │
                              ▼
               ┌───────────────────────────────┐
               │    5. Full Verification Run   │
               │  - Unit tests & builds        │
               │  - Verify all local features  │
               └──────────────┬────────────────┘
                              │
                              ▼
               ┌───────────────────────────────┐
               │ 6. Update Docs & session.md   │
               └───────────────────────────────┘
```

### Step 1: Pre-Merge Preparation
1. Ensure the working tree is clean: `git status`.
2. Ensure you are on the integration branch (typically `develop`).
3. Ensure private development databases in `data/` or `~/.config/v2raya/` are backed up.

### Step 2: Fetch and Inspect Upstream Changes
```bash
git fetch upstream
git log --oneline HEAD..upstream/main
git diff --stat HEAD..upstream/main
```
Identify which files touched upstream overlap with the **Key Implementation Files** listed in Section 3.

### Step 3: Perform Merge
```bash
git merge upstream/main --no-commit
```
If conflicts arise, proceed to Section 3 and follow the **Conflict Resolution Guidance** for each affected feature.

### Step 4: Verification Suite
Execute the mandatory verification commands:
```bash
# Backend validation
cd service && go build ./... && go vet ./...
go test ./db/configure ./kernel/v2ray ./kernel/touch ./kernel/serverObj ./server/service ./server/controller

# Frontend validation
cd ../gui && npm run build
```

### Step 5: Documentation & Session Update
Record the merge commit SHA, conflicts encountered, resolution decisions, and test outputs in `session.md` and `CHANGELOG.md`.

---

## 3. Inventory of Locally Built Features to Preserve

---

### Feature 1: Per-Proxy-Group Round-Robin Load Balancing

#### Description & Rationale
Upstream v2rayA assigns a single active server per proxy group or uses a global setting. This local feature allows **each individual proxy group** to independently enable round-robin load balancing across all connected servers in that group. New outgoing proxy connections cycle sequentially through the group's servers, while long-lived connections remain stable on their assigned server.

#### Key Implementation Files
- `service/db/configure/outbound.go`:
  - Struct `OutboundSetting` with `LoadBalanceEnabled bool` and `LoadBalanceStrategy string`.
  - Constants `LoadBalanceStrategyRoundRobin = "roundrobin"`.
  - Helpers `DefaultOutboundSetting()` and `NormalizeLoadBalanceStrategy()`.
- `service/server/service/connect.go`:
  - In `SetConnect()` / connection handling: skips dropping existing connections in a group when `LoadBalanceEnabled` is active, allowing multi-server attachment.
- `service/kernel/v2ray/template_outbound.go`:
  - Generates individual server outbounds tagged as `『groupName』_0`, `『groupName』_1`, ..., `『groupName』_N`.
  - Generates selector outbound `『groupName』` with protocol `"selector"`.
- `service/kernel/v2ray/template_routing.go`:
  - `setGroupRouting()` redirects routing rules targeting the group to use balancer tags.
- `service/kernel/v2ray/template_api.go`:
  - `SetAPI()` iterates `t.Outbounds`, identifies balancers via `len(o.Balancers) > 0`, copies `o.Balancers` into the selector, skips empty selector lists (preventing Xray crashes), and resolves outbound settings using `serverInfoMap` and `extractGroupName()`.
- `service/kernel/v2ray/template_loadbalance_test.go`:
  - Regression and unit tests for balancer templates.
- `gui/src/components/modalOutboundSetting.vue`:
  - UI switch for **Load balancing** and selection dropdown for **Round robin**.
- `gui/src/components/outboundGroupPanel.vue`:
  - Displays load balancing status badges on proxy groups.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream changes to `template_outbound.go` or `template_api.go` often assume one outbound per group tag and will overwrite balancer generation, or upstream might introduce a conflicting global balancer structure.
- **Resolution Strategy:**
  1. Retain the fields `LoadBalanceEnabled` and `LoadBalanceStrategy` in `OutboundSetting`.
  2. Preserve `setGroupRouting()` in `template_routing.go`.
  3. Ensure `SetAPI()` in `template_api.go` retains empty selector protection (`continue` if selector list is empty) and fallback name extraction via `extractGroupName()`.
  4. Ensure `SetConnect()` in `connect.go` does not disconnect existing servers if load balancing is enabled.

#### Verification
```bash
cd service && go test -v -run ".*LoadBalance.*" ./kernel/v2ray ./db/configure
cd service && go build ./kernel/v2ray/...
```

---

### Feature 2: Active Proxy Group Selection

#### Description & Rationale
Upstream always routes default proxy traffic and transparent proxy traffic to the hardcoded `proxy` group. This local feature allows the user to designate **any configured proxy group as the active default group** directly from the GUI (via the **PROXY GROUP** tab). Missing, deleted, or empty groups self-heal back to `proxy`.

#### Key Implementation Files
- `service/db/configure/outbound.go`:
  - `GetActiveProxyOutboundName()`: Retrieves active outbound from `system_config:active_outbound`, defaulting to `"proxy"`.
  - `SetActiveProxyOutboundName(name string)`: Persists active outbound.
- `service/server/controller/outbound.go`:
  - `GetOutbounds`: Injects `activeOutboundName` into the JSON response.
  - `PutActiveOutbound`: Validates request, sets active outbound in DB, reloads running core, and reverts on failure.
- `service/server/router/index.go`:
  - Registers route `PUT /api/outbound/active`.
- `service/kernel/v2ray/template_routing.go`:
  - Resolves default proxy routing rules, transparent proxy rules, and reserved `proxy` targets to the currently configured active outbound group name instead of a hardcoded string.
- `gui/src/components/ProxyGroupPanel.vue`:
  - Table of proxy groups with radio button for active group selection, member count badge, and active status display.
- `gui/src/node.vue`:
  - Node navigation tab for **PROXY GROUP**.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream router changes in `index.go` may omit `PUT /api/outbound/active`. Upstream changes in `template_routing.go` often hardcode `"proxy"` as the outbound tag.
- **Resolution Strategy:**
  1. Ensure `system_config:active_outbound` DB operations remain intact.
  2. In `template_routing.go`, ensure rule generation queries `GetActiveProxyOutboundName()`.
  3. Re-register `auth.PUT("outbound/active", controller.PutActiveOutbound)` in `service/server/router/index.go`.
  4. Preserve `ProxyGroupPanel.vue` and its tab link in `gui/src/node.vue`.

#### Verification
```bash
cd service && go test -v -run ".*Active.*|.*Outbound.*" ./db/configure ./kernel/v2ray ./server/controller
```



---

### Feature 3: Live Flow Real-Time Topology Visualization & Event Pipeline

#### Description & Rationale
Live Flow provides an interactive, animated SVG diagram displaying active traffic paths:
`source → selected proxy / proxy chain → destination`.
It operates via real-time WebSocket feeds and line-buffered stdout route events from the core process without adding external runtimes (such as React Flow). Subdomains are grouped under root registrable domains to prevent canvas clutter, and internal latency probes (`gstatic.com:443`) are filtered out.

The pipeline is dual-channel and resilient:
1. **Core Emitter (`core/main/liveflow.go`):** When `V2RAYA_LIVE_FLOW=1` is passed to `v2raya_core`, logger handler creators intercept accepted `AccessMessage` instances from Xray and output structured JSON lines (`V2RAYA_LIVE_FLOW {...}`) to stdout.
2. **Universal Access-Log Fallback (`service/kernel/v2ray/liveflow.go`):** In addition to private IPC events, `ConsumeCoreLiveFlowEvent()` parses standard Xray access logs (`from <source> accepted <destination> [<detour>]`). This guarantees real-time traffic tracking across all core deployments (bundled, external, Docker) even if private IPC hooks are absent.
3. **Resilient Pub-Sub Broker (`service/kernel/v2ray/feed.go` & `api.go`):** `ApiFeed` registers `LiveFlowProduct` (`"live_flow"`), and `Feed.SubscribeMessage()` / `Feed.ProductMessage()` automatically register unknown products on first use instead of returning `nil`, preventing dropped subscriptions.
4. **Interactive SVG Graphic Canvas (`gui/src/views/LiveFlowView.vue`):** Interactive SVG graphic topology canvas (`viewBox="0 0 1200 height"`) rendered with cubic Bezier paths, animated dashes, route status markers (`active`, `idle`, `error`), registrable root-domain aggregation, and inspection modals (Route Details, Subdomains, Ports), plus a view-toggle between Topology Canvas and Session Table.
5. **Robust Endpoint Normalization (`gui/src/plugins/liveflow.ts`):** Client normalizes endpoints having either IP or domain without discarding valid routes.

#### Key Implementation Files
- `core/main/liveflow.go` & `core/main/liveflow_test.go`:
  - Structured route event emitter using Xray's `xlog.RegisterHandlerCreator` for `LogType_Console`, `LogType_File`, and `LogType_None`, activated via `V2RAYA_LIVE_FLOW=1`.
- `service/kernel/v2ray/process.go`:
  - Sets `V2RAYA_LIVE_FLOW=1` in `StartCoreProcess()`.
  - Line-buffered stdout scanner inspecting core output for route events, invoking `ConsumeCoreLiveFlowEvent()`.
- `service/kernel/v2ray/liveflow.go` & `service/kernel/v2ray/liveflow_test.go`:
  - Parses private route events (`V2RAYA_LIVE_FLOW `).
  - Fallback parser `parseAccessLogLine()` extracting routes from standard Xray access logs.
  - Strips port/trailing dot, filters probe `gstatic.com:443`, aggregates subdomains by registrable domain, resolves outbound server tags (trimming brackets `[socks_ipv4 -> proxy]`), and publishes events to `ApiFeed("live_flow")`.
- `service/kernel/v2ray/api.go`:
  - Registers `LiveFlowProduct` on `ApiFeed` during `init()`.
- `service/kernel/v2ray/feed.go` & `service/kernel/v2ray/feed_test.go`:
  - Auto-registers products in `SubscribeMessage()` and `ProductMessage()` to prevent `nil` subscription boxes.
- `service/server/service/liveflow.go` & `service/server/controller/liveflow.go`:
  - Subscribes to `ApiFeed("live_flow")`, sends initial `batch_state` snapshot, and bridges events to `/api/live-flow` WebSockets with token-bucket rate limiting and JWT validation.
- `service/server/router/index.go`:
  - Registers `/api/live-flow` WebSocket handler.
- `gui/src/views/LiveFlowView.vue`:
  - Interactive SVG topology canvas with directed Bezier paths, root domain aggregation, inspector dialogs, and Topology/Table toggle, styled with Material 3 CSS variables (`rgb(var(--v-theme-*))`).
- `gui/src/plugins/liveflow.ts`:
  - Client WebSocket protocol, exponential-backoff reconnect, and endpoint normalization (supporting IP and domain).

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream does not have Live Flow. Upstream upgrades to `core/` (Xray) or `service/kernel/v2ray/process.go` can overwrite the stdout scanner, drop `ConsumeCoreLiveFlowEvent`, omit `V2RAYA_LIVE_FLOW=1` in `StartCoreProcess()`, overwrite `Feed.SubscribeMessage()`, or remove the `/api/live-flow` route in `router/index.go`. Upstream GUI upgrades may overwrite `gui/src/views/LiveFlowView.vue`.
- **Resolution Strategy:**
  1. Verify `service/kernel/v2ray/process.go` retains `env = append(env, "V2RAYA_LIVE_FLOW=1")` and the `ConsumeCoreLiveFlowEvent(line)` hook in `handleLine()`.
  2. Preserve `core/main/liveflow.go` and `core/main/liveflow_test.go`.
  3. Preserve `service/kernel/v2ray/liveflow.go` including `parseAccessLogLine()` fallback, probe exclusion, and domain aggregation logic.
  4. Preserve `ApiFeed.RegisterProduct(LiveFlowProduct)` in `api.go` and auto-registration in `service/kernel/v2ray/feed.go`.
  5. Ensure `service/server/router/index.go` includes `auth.GET("live-flow", controller.GetLiveFlow)`.
  6. Preserve `gui/src/views/LiveFlowView.vue` and `gui/src/plugins/liveflow.ts`.

#### Verification
```bash
cd core && go test ./main -v
cd service && go test -v ./kernel/v2ray -run "TestConsumeCoreLiveFlow|TestParseAccessLogLine|TestParseLiveFlow|TestSelectedDetour|TestRecordCoreAccess|TestFeedAutoRegistersProduct|TestApiFeedRegistersLiveFlowProduct"
cd service && go test -v ./server/service ./server/controller -run "TestLiveFlow.*"
cd gui && yarn typecheck && yarn i18n-check && yarn test && yarn build
```

---

### Feature 4: Server Eligibility & Protocol/Encryption Filter Rules

#### Description & Rationale
Prevents dangerous or broken connection attempts while preserving node visibility.
1. **VMess with TLS and encryption `none` or `false`:** Flagged with a `DisableReason`. VMess's internal cipher is disabled, leaving transport insecure if TLS misbehaves.
2. **VLESS without TLS (`security=none`, `security=false`, or empty TLS):** Universally blocked, as VLESS has no application-layer encryption.
3. **VLESS+TLS:** **Explicitly allowed** because VLESS protocol encryption is `none` by design and TLS provides transport security.
4. **Node Controls:** Protocol and encryption filters are rendered in independent containers for local and subscription tables.

#### Key Implementation Files
- `service/kernel/serverObj/v2ray.go`:
  - `DisableReason()` method containing eligibility logic.
- `service/kernel/serverObj/display_test.go`:
  - Unit tests ensuring VLESS+TLS is enabled and unsafe VMess/VLESS nodes are disabled with appropriate reasons.
- `service/kernel/touch/touch.go` & `touch_test.go`:
  - Rejects connection/touch for nodes with a `DisableReason()`.
- `gui/src/node.vue`:
  - Displays tooltip warning on disabled servers, blocks their selection for connections/groups, and provides independent filter panels.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream parser changes in `serverObj/v2ray.go` or `touch.go` do not validate this combination or may universally block/allow all VLESS configs.
- **Resolution Strategy:**
  1. Ensure `DisableReason()` in `v2ray.go` maintains the exact rule:
     ```go
     if strings.EqualFold(v.Protocol, "vmess") && isTLSEnabled(v.TLS) && isNoneOrFalse(v.Security) {
         return "TLS enabled with encryption 'none' or 'false' is not allowed"
     }
     ```
  2. Confirm VLESS+TLS is NOT disabled for having `encryption: none`.
  3. Ensure `touch.go` blocks servers with `DisableReason != ""`.

#### Verification
```bash
cd service && go test -v ./kernel/serverObj -run "TestServerEligibility.*|TestVless.*"
cd service && go test -v ./kernel/touch -run "TestTouch.*"
```

---

### Feature 5: Bulk Add by Latency (Tab Scoping & Multi-Subscription State Preservation)

#### Description & Rationale
Allows users to add all servers with latency below a specified millisecond threshold to an outbound proxy group in one operation.
- **Tab Scoping:** Candidate node discovery is strictly scoped to the active tab (local SERVER tab or the specific subscription tab currently viewed). Users bulk-adding from Subscription B only see servers from Subscription B as candidates.
- **Node Eligibility Guard:** Servers with a `disableReason` (e.g. insecure VMess/VLESS) are strictly excluded from being added.
- **Cross-Subscription State Preservation:** When the frontend issues `PUT /api/outboundConnections`, it sends the complete list of desired connections. The local implementation retains all existing connected servers across **all** subscriptions and groups, rather than only those on the active tab. In upstream, triggering bulk add on one subscription tab would wipe out connections previously established from other subscriptions.
- **Tab Availability:** The bulk add action is available on the local SERVER tab (`tab == 1`) and individual subscription tabs (`tab >= 2`), but disabled on the subscription overview (`tab == 0`) and Live Flow (`tab == -1`).

#### Key Implementation Files
- `gui/src/components/modalBulkAddByLatency.vue`:
  - `props`: `tabType` (`"server"` | `"subscription"` | `"all"`), `tabIndex` (0-based subscription index).
  - `getAllServers()`: Filters servers by `tabType` and `tabIndex` so preview candidate servers match the active tab.
  - `matchingServers()`: Filters out servers with `s.disableReason` or missing/invalid `pingLatency`.
  - `serversToAdd()`: Computes diff against `currentConnectedServers` for `selectedOutbound`.
  - `confirm()`: Crucial preservation invariant: maps `existingTouches` from `currentConnectedServers` across **all** subscriptions for `selectedOutbound` and merges them with `newTouches` before calling `PUT /api/outboundConnections`.
- `gui/src/node.vue`:
  - `isBulkAddByLatencyAvailable`: Computed property checking `this.tab === 1 || (this.tab >= 2 && this.tab < this.tableData.subscriptions.length + 2)`.
  - `handleClickBulkAddByLatency()`: Passes `tabType` and `tabIndex` to `modalBulkAddByLatency`.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream does not have `modalBulkAddByLatency.vue` or tab scoping in `node.vue`. Upstream merges can remove `isBulkAddByLatencyAvailable`, revert `node.vue` header buttons, or reintroduce code that wipes out other subscriptions' connections.
- **Resolution Strategy:**
  1. Keep `gui/src/components/modalBulkAddByLatency.vue`.
  2. In `gui/src/node.vue`, retain `isBulkAddByLatencyAvailable`, the header button, and `handleClickBulkAddByLatency()`.
  3. Ensure `confirm()` in `modalBulkAddByLatency.vue` retains all existing touches across subscriptions.

---

### Feature 6: Bulk Remove by Latency (Threshold/Timeout Purge & Safety Non-Empty Guard)

#### Description & Rationale
Allows users to purge high-latency or failing servers from a proxy group in one action, while enforcing cluster safety:
- **Threshold & Timeout Detection:** Identifies connected servers in the selected outbound whose ping latency is greater than or equal to `maxLatencyMs`, or servers with no latency data, timeout status (`"TIMEOUT"`, `"-"`), or connection errors.
- **Safety Non-Empty Guard (Backend & Frontend):** Prevents leaving a proxy group completely empty. If removing the matching servers would remove all servers from the group, the frontend warns the user and disables confirmation (`wouldEmptyGroup`), and the backend controller rejects the request with an explicit error (`"cannot remove all servers from the group; at least one server must remain"`).
- **Tab Scoping:** Focuses candidate inspection on the currently viewed tab.

#### Key Implementation Files
- `service/server/controller/outbound.go`:
  - `PostBulkRemoveByLatency(ctx *gin.Context)`: Parses `outbound`, `maxLatencyMs`, and `serversToCheck`. Filters connected servers, identifies timeout/error/high-latency nodes, validates that `len(remaining) > 0`, and calls `service.ReplaceOutboundConnections()`.
- `service/server/router/index.go`:
  - Registers route `auth.POST("bulkRemoveByLatency", controller.PostBulkRemoveByLatency)`.
- `gui/src/components/modalBulkRemoveByLatency.vue`:
  - UI modal displaying latency threshold input, target group selector, preview count of servers to remove, error breakdown, and `wouldEmptyGroup` warning guard.
- `gui/src/node.vue`:
  - `isBulkRemoveByLatencyAvailable`: Computed property mirroring `isBulkAddByLatencyAvailable`.
  - `handleClickBulkRemoveByLatency()`: Launches `modalBulkRemoveByLatency`.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream lacks the `POST /api/bulkRemoveByLatency` endpoint and `PostBulkRemoveByLatency` handler. Merges touching `service/server/router/index.go` or `service/server/controller/outbound.go` will delete the route and handler.
- **Resolution Strategy:**
  1. Ensure `PostBulkRemoveByLatency` is preserved in `service/server/controller/outbound.go`.
  2. Verify `auth.POST("bulkRemoveByLatency", controller.PostBulkRemoveByLatency)` is registered in `service/server/router/index.go`.
  3. Retain `modalBulkRemoveByLatency.vue` and its invocation in `gui/src/node.vue`.

#### Verification
```bash
cd service && go build ./server/controller ./server/router
```

---

### Feature 7: Local Development Persistence Safeguards

#### Description & Rationale
Ensures that local databases, temporary keys, dev configs, and dev scripts survive upstream merges and cannot be accidentally committed or wiped during upstream rebases/merges.

#### Key Implementation Files
- `.gitignore`:
  - Local runtime data: `/data/`
  - SQLite/DB files: `*.db`, `*.db-shm`, `*.db-wal`, `*.sqlite`, `*.sqlite3`, `*.bolt`, `*.boltdb`
  - Local dev environment: `.env.local`, `.env.*.local`
  - Dev scripts: `/dev-*.sh`, `/dev-*.ps1`, `/dev-*.md`, `/dev-*.json`, `/dev-*.yaml`
  - Dev binaries: `/v2raya-dev*`, `/v2raya_core-dev*`
- `.dockerignore`:
  - Ignores `data/` and local development artifacts from container builds.
- `docs/DEV_PERSISTENCE.md`:
  - Complete workflow guide for maintaining non-repo dev state.

#### Merge Vulnerability & Protection Strategy
- **Upstream Hazard:** Upstream `.gitignore` updates often conflict. Taking upstream's file can expose private `.db` files or track `data/`.
- **Resolution Strategy:** Always retain the local persistence block in `.gitignore`.

---

### Feature 8: Docker Build Automation with Version & Timestamp Injection

#### Description & Rationale
Local Docker builds dynamically inject semantic version, commit short hash, and RFC 3339 build timestamps into Go binary ldflags, and utilize multi-stage caching with local Go module tidy.

#### Key Implementation Files
- `docker-build.sh`:
  - Script for building container images with dynamic git version and commit tagging.
- `Dockerfile`:
  - Multi-stage build with explicit build arguments (`V2RAYA_VERSION`, `V2RAYA_COMMIT`, `BUILD_DATE`).

---

## 4. Quick Reference Merge Checklist Matrix

| File Path | Associated Feature | On Conflict: Action Required |
|---|---|---|
| `service/db/configure/outbound.go` | Load Balancing & Active Outbound | Retain `LoadBalance*` fields, `DefaultOutboundSetting()`, `GetActiveProxyOutboundName()` |
| `service/db/schema.go` | All Features | Preserve `outbound_settings` table and `system_config` keys |
| `service/kernel/v2ray/template_outbound.go` | Load Balancing | Retain `『group』_0..N` outbounds and selector outbound |
| `service/kernel/v2ray/template_routing.go` | Load Balancing & Active Outbound | Preserve `setGroupRouting()` and `GetActiveProxyOutboundName()` resolution |
| `service/kernel/v2ray/template_api.go` | Load Balancing | Retain empty selector check and `extractGroupName()` fallback |
| `service/kernel/v2ray/process.go` | Live Flow | Retain line-buffered scanner, `ConsumeCoreLiveFlowEvent` call, and `V2RAYA_LIVE_FLOW=1` env var |
| `service/kernel/v2ray/liveflow.go` | Live Flow | Keep entire file; retain `parseAccessLogLine()` fallback, probe exclusion, domain aggregation, and bracket normalization |
| `service/kernel/v2ray/api.go` | Live Flow | Preserve `ApiFeed.RegisterProduct(LiveFlowProduct)` |
| `service/kernel/v2ray/feed.go` | Live Flow | Preserve auto-registration in `SubscribeMessage()` and `ProductMessage()` |
| `core/main/liveflow.go` | Live Flow | Keep entire file; Xray logger handler creator hook emitting route events to stdout |
| `service/kernel/serverObj/v2ray.go` | Server Eligibility | Retain `DisableReason()` rule for VMess (keep VLESS+TLS allowed) |
| `service/server/controller/outbound.go` | Active Group & Bulk Remove | Preserve `SetActiveOutbound`, `GetOutbounds` enhancements, and `PostBulkRemoveByLatency` |
| `service/server/service/liveflow.go` | Live Flow | Keep entire file; subscribe to `LiveFlowProduct`, send initial `batch_state`, forward live-flow feed |
| `service/server/router/index.go` | Active Group, Live Flow & Bulk Remove | Preserve `PUT /api/outbound/active`, `GET /api/live-flow`, and `POST /api/bulkRemoveByLatency` |
| `gui/src/node.vue` | Active Group, Bulk Add/Remove, Server Filters | Preserve **PROXY GROUP** tab, separate filter boxes, bulk latency action buttons, and tab scoping |
| `gui/src/components/ProxyGroupPanel.vue` | Active Group Selection | Keep entire file |
| `gui/src/views/LiveFlowView.vue` | Live Flow Topology | Keep entire file; preserve interactive SVG graphic canvas, Bezier paths, root domain aggregation, and Topology/Table toggle |
| `gui/src/plugins/liveflow.ts` | Live Flow Client | Keep entire file; endpoint normalization (ip/domain) and reconnect logic |
| `gui/src/components/modalBulkAddByLatency.vue` | Bulk Add by Latency | Keep entire file; verify multi-subscription retention in `confirm()` |
| `gui/src/components/modalBulkRemoveByLatency.vue` | Bulk Remove by Latency | Keep entire file; verify safety guard against emptying groups |
| `.gitignore` | Local Dev Persistence | Preserve dedicated local development persistence block |

---

## 5. Guidelines for Adding Future Local Features

When building a new feature locally that does not exist upstream:
1. **Document immediately:** Add an entry to Section 3 of `MERGE-PRESERVATION.md`.
2. **Specify Touchpoints:** Detail every file, struct, function signature, and UI component touched.
3. **Record Hazards:** Anticipate what upstream changes could break it.
4. **Provide Test Commands:** Document the exact command to verify it.
5. **Update Session:** Cross-reference the addition in `session.md`.

