# Code Graph & System Architecture

## Overview

This document provides a comprehensive structural and behavioral code graph of v2rayA. It maps relationships between subsystems, packages, data flows, and critical execution paths, with explicit touchpoints for locally developed features.

---

## 1. Top-Level Directory Layout

```text
v2rayA/
├── core/                # Custom V2Ray/Xray core binary entry point & extensions
│   ├── dns/             # Internal DNS module, caching, upstreams, ECS handling
│   ├── hint/            # Protocol & observatory hint descriptors
│   ├── main/            # Core binary main package, distro loader, versioning
│   └── xray/            # Submodule / vendored Xray codebase with logging hooks
├── service/             # Backend service (Go + Gin + SQLite)
│   ├── common/          # Common types, cryptographic helpers, utilities
│   ├── conf/            # Environment and runtime configurations
│   ├── db/              # Database schema, operations, migrations, and SQLite drivers
│   │   └── configure/   # Configuration persistence (settings, outbounds, DNS)
│   ├── kernel/          # Process management, server models, template generation
│   │   ├── coreObj/     # Abstract core process objects
│   │   ├── ipforward/   # Linux kernel IP forwarding controls
│   │   ├── serverObj/   # Server protocols (VMess, VLESS, SS, Trojan, etc.) & eligibility
│   │   ├── touch/       # Server connection state, latency testing, touch endpoints
│   │   └── v2ray/       # Xray/V2Ray config templates, process lifecycle, live flow
│   ├── pkg/             # Supporting utilities (JWT, logging, process execution)
│   └── server/          # Web server & API layer
│       ├── controller/  # Gin HTTP and WebSocket request handlers
│       ├── router/      # Route registrations, middleware, static web asset serving
│       └── service/     # Business logic layer (connection, outbound, live flow)
├── gui/                 # Frontend Web application (Vue.js + Buefy + Vite)
│   └── src/
│       ├── components/  # Modals, panels, Live Flow SVG canvas, Proxy Group panel
│       ├── plugins/     # Axios client, WebSocket bridges (liveflow.js)
│       └── store/       # Vuex state management
├── docs/                # Architecture, specifications, roadmaps, contributor guides
├── tools/               # Benchmarking, performance testing scripts
└── web/                 # Built frontend static distribution files
```

---

## 2. Subsystem Dependency & Layer Graph

```text
[ Browser / GUI (Vue.js) ]
       │
       │ HTTP / REST & WebSockets
       ▼
[ Web Server Layer (service/server/router) ]
       │
       ▼
[ Controller Layer (service/server/controller) ]
       │
       ▼
[ Service Layer (service/server/service) ]
       │
       ├──► [ Database & Configuration (service/db & service/db/configure) ]
       │         │
       │         └──► SQLite Database File (/etc/v2raya/ or local data/)
       │
       └──► [ Kernel Orchestration (service/kernel/v2ray) ]
                 │
                 ├──► Configuration Generation (template_*.go)
                 │         ├── Outbounds & Balancers (template_outbound.go)
                 │         ├── Routing & Active Group (template_routing.go)
                 │         └── Balancer API & Metrics (template_api.go)
                 │
                 └──► Process Execution (process.go)
                           │
                           ▼ Spawns & Monitors via Pipe / IPC
                 [ Core Binary (v2raya_core) ]
                           │
                           ├── Routing & Load Balancing (Xray roundrobin)
                           ├── MultiObservatory Probes
                           └── Structured Route Log Events (V2RAYA_LIVE_FLOW=1)
```


---

## 3. Key Data & Control Flow Pipelines

### 3.1 Kernel Configuration & Proxy Group Load Balancing Pipeline

1. **User Action / API Request:**
   - GUI triggers `PUT /api/outbound/active` or `PUT /api/outbound/setting`.
   - `service/server/controller/outbound.go` validates group/strategy.
2. **Persistence:**
   - Stored in SQLite via `service/db/configure/outbound.go`.
3. **Template Assembly:**
   - `service/kernel/v2ray/template_core.go` calls:
     - `template_outbound.go`: When `LoadBalanceEnabled` is true, generates separate outbounds `『group』_0` ... `『group』_N` and selector outbound `『group』`.
     - `template_routing.go`: Converts proxy routing rules to point to `ActiveProxyOutboundName` or balancer selector tags.
     - `template_api.go`: Configures Xray routing balancers with strategy `roundrobin`.
4. **Process Reload:**
   - `service/kernel/v2ray/process.go` restarts or reloads `v2raya_core` with the new configuration.

---

### 3.2 Live Flow Real-time Event Streaming Pipeline

```text
[ Core Process: Xray Outbound Selection ]
                   │
                   ▼ Opt-in Route Event (V2RAYA_LIVE_FLOW=1)
          stdout (Line-buffered)
                   │
                   ▼
[ service/kernel/v2ray/process.go: Scanner ]
                   │
                   ▼ Calls ConsumeCoreLiveFlowEvent()
[ service/kernel/v2ray/liveflow.go ]
   - Drops gstatic.com:443 probe
   - Groups subdomains by root domain
   - Resolves proxy server metadata
   - Publishes to ApiFeed("live_flow")
                   │
                   ▼
[ service/server/service/liveflow.go ]
   - Bridges feed to authenticated WebSockets
                   │
                   ▼ /api/live-flow (WebSocket)
[ gui/src/plugins/liveflow.js ]
                   │
                   ▼ Vuex Store Update
[ gui/src/components/liveFlowDashboard.vue ]
   - Renders animated SVG route topology
```

---

## 4. Local Feature Integration Map

| Feature | Key Backend Packages | Key Frontend Components |
|---|---|---|
| **Per-Group Round-Robin Balancing** | `db/configure`, `kernel/v2ray` (`template_outbound`, `template_routing`, `template_api`), `server/service` | `modalOutboundSetting.vue`, `outboundGroupPanel.vue` |
| **Active Proxy Group Selection** | `db/configure`, `server/controller/outbound`, `kernel/v2ray/template_routing` | `ProxyGroupPanel.vue`, `node.vue` |
| **Live Flow Topology** | `core/xray/app/log`, `kernel/v2ray/liveflow`, `server/controller/liveflow`, `server/service/liveflow` | `liveFlowDashboard.vue`, `liveflow.js` |
| **Server Eligibility (VMess/VLESS)** | `kernel/serverObj/v2ray.go`, `kernel/touch` | `node.vue` (tooltips, filter containers) |
| **Bulk Add & Remove by Latency** | `server/controller/outbound` (`PostBulkRemoveByLatency`), `server/router`, `kernel/touch` | `modalBulkAddByLatency.vue`, `modalBulkRemoveByLatency.vue`, `node.vue` |
| **Dev Persistence & Safe Merging** | `.gitignore`, `.dockerignore`, `db/sqlite.go` | `docs/DEV_PERSISTENCE.md`, `MERGE-PRESERVATION.md` |
