# Architecture

## Overview

v2rayA is a modern, web-based proxy management platform built with a Go backend and Vue.js frontend. The system orchestrates V2Ray/XRay core functionality while providing an intuitive web interface for configuration and management.

## System Components

### 1. Backend Service (`service/`)

The core of v2rayA is a Go service built with the Gin web framework.

**Entry Point:** `service/main.go`

**Key Components:**
- **Web Server:** Gin-based HTTP server with JWT authentication
- **API Controllers:** RESTful API endpoints for GUI interaction
- **Kernel Management:** V2Ray/XRay process lifecycle management
- **Database Layer:** SQLite-based persistent storage
- **Configuration Management:** Settings, subscriptions, and routing rules
- **Network Tools:** Port management, IP forwarding, iptables manipulation

**Architecture Pattern:**
- Controller → Service → Repository pattern
- Clean separation between API layer and business logic
- Dependency injection for testability

### 2. Core Binary (`core/`)

A custom-built V2Ray/XRay binary with additional features:

- Extended V4-compatible command-line interface
- MultiObservatory support for latency monitoring
- Custom protocol implementations
- Built-in protocol sniffing capabilities

**Entry Point:** `core/main/main.go`

### 3. Web GUI (`gui/`)

Vue.js-based single-page application:

- **Framework:** Vue 2 with Buefy UI components
- **Build Tool:** Vite (with Vue CLI fallback)
- **State Management:** Vuex store
- **HTTP Client:** Axios with request caching
- **Internationalization:** Multi-language support via `locales/`
- **Server filtering and eligibility:** `/api/touch` supplies canonical server `protocol`, `encryptions`, and optional `disableReason` metadata. `node.vue` derives dynamic options for local and per-subscription server tables in independent filter containers, retaining filter state locally per tab. VMess servers with TLS enabled and `none` or `false` encryption remain visible but cannot be selected or connected. VLESS+TLS is always allowed (VLESS protocol encryption is `none` by design; TLS provides transport security). VLESS servers without TLS (`security=false`, `security=none`, or empty TLS) are universally blocked.

**Entry Point:** `gui/src/main.js`

### 4. Live Flow Topology (`core/`, `service/`, `gui/`)

The experimental Live Flow feature is a Vue-native SVG topology that renders observed routes as source → selected proxy/chain → destination. It does not use React Flow because the GUI is Vue 2.

1. The bundled `v2raya_core` emits an opt-in structured event after Xray selects an outbound for an accepted access request.
2. The service consumes only the dedicated private stdout prefix, validates source/destination/detour fields, excludes the internal `gstatic.com:443` HTTP latency probe, resolves selected server metadata from connected-server configuration, and publishes `flow_*` events to `ApiFeed` product `live_flow`.
3. `/api/live-flow` bridges that product to JWT-authenticated WebSocket clients.
4. The Vuex-backed dashboard renders shared SVG source, proxy, and destination nodes with animated directed paths.

The event source provides route selection, not exact per-session byte counters or close callbacks. Sessions expire after a bounded period without another observation. Observatory health events remain separate and are not transformed into traffic routes.

## Data Flow

```
┌─────────────────────────────────────────────────────────┐
│                    User Browser                        │
│               (Vue.js Web Application)                 │
└─────────────────────────────────────────────────────────┘
                          │
                          │ HTTP/WebSocket
                          ▼
┌─────────────────────────────────────────────────────────┐
│                  v2rayA Backend Service                 │
│                    (Gin + Go)                           │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │ Controllers │  │  Services   │  │  Database   │    │
│  │  (API)      │──│  (Logic)    │──│  (SQLite)   │    │
│  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Process Management
                          ▼
┌─────────────────────────────────────────────────────────┐
│               V2Ray/XRay Core Binary                   │
│            (Protocol Implementation)                   │
└─────────────────────────────────────────────────────────┘
```

## Database Schema

### Tables

1. **`servers`** - Proxy server configurations
   - Server addresses, ports, protocols
   - Configuration JSON, latency data
   - Subscription associations

2. **`subscriptions`** - Subscription sources
   - URLs, auto-update settings
   - Filter rules and status

3. **`accounts`** - User authentication
   - Username, password hashes

4. **`system_config`** - Key-value settings store
   - Application settings
   - Feature flags

5. **`outbound_names`** / **`outbound_connections`** / **`outbound_settings`** - Custom outbound routing

### Migration Strategy

- Schema versioning via migration functions
- Backward-compatible column additions
- Safe rollback capabilities

## API Architecture

### Authentication
- JWT-based authentication
- Token refresh mechanism
- Session management

### RESTful Endpoints

**Core Operations:**
- `POST /import` - Import server configurations
- `POST /connection` - Connect to proxy server
- `DELETE /connection` - Disconnect
- `GET /setting` / `PUT /setting` - Configuration management

**Server Management:**
- `GET /touch` - List servers
- `DELETE /touch` - Remove servers
- `POST /v2ray` / `DELETE /v2ray` - Server operations

**Subscription Management:**
- `PUT /subscription` - Update subscriptions
- `PATCH /subscription` - Partial updates

**Routing & DNS:**
- `GET /routingA` / `PUT /routingA` - Routing rules
- `GET /dnsRules` / `PUT /dnsRules` - DNS configuration

**Monitoring:**
- `GET /message` - WebSocket for real-time updates
- `GET /logger` - Log access
- `GET /pingLatency` / `GET /httpLatency` - Latency testing

## Security Architecture

### Authentication Flow
1. Initial login with username/password
2. JWT token generation
3. Token validation on subsequent requests
4. Secure cookie-based sessions

### Privilege Management
- Root/Admin required for transparent proxy
- `--lite` mode for non-root operation
- Platform-specific privilege checks

### Data Protection
- Local storage only - no cloud sync
- Encrypted password storage
- Secure defaults for all configurations

## Network Architecture

### Transparent Proxy Modes

1. **TProxy** (Linux)
   - Uses `IP_TRANSPARENT` socket option
   - Requires `CAP_NET_ADMIN` capability
   - iptables integration for traffic redirection

2. **System Proxy** (Windows/macOS)
   - OS-level proxy settings
   - PAC file support
   - Direct system configuration

### Port Management
- SOCKS5 proxy port (default: 2080)
- HTTP proxy port (default: 2081)
- V2Ray/XRay internal ports
- Custom port configurations

### IP Forwarding
- Configurable IP forwarding for routing
- Platform-specific implementations
- Automatic setup on connection

## Configuration Management

### Settings Categories
1. **General:** Version, update modes
2. **Proxy:** Protocol, ports, timeout
3. **Routing:** Rule modes, GFW lists
4. **DNS:** Custom rules, anti-pollution
5. **Logging:** Levels, output destinations

### Outbound Group Configuration

v2rayA supports grouping selected servers into outbound groups. Each group persists an `OutboundSetting` record, including its latency-probe options and an independent load-balancing policy (`loadBalanceEnabled`, `loadBalanceStrategy`).

#### Balancer Groups (Load Balancing)

When a group enables load balancing and has at least two selected members:
1. **Individual server outbounds:** Each member gets its own Xray outbound configuration with a tag following `『groupName』_N` (for example, `『PROXY』_0` through `『PROXY』_4`).
2. **Routing balancer:** The template adds `routing.balancers[]` with group tag `『groupName』`, a selector containing those exact member tags, and the group’s selected strategy. The current supported strategy is `roundrobin`.
3. **Routing integration:** Rules targeting either the raw group name (`PROXY`) or group tag are translated to Xray’s `balancerTag` (`『PROXY』`).
4. **Disabled or one-member groups:** No Xray balancer is generated; the first selected member remains the ordinary active outbound. Group membership is retained so enabling the policy later does not require re-adding members.

**Configuration Example:**
```json
{
  "outbounds": [
    {
      "tag": "『proxy』_0",
      "protocol": "vmess",
      "settings": { ... }
    },
    {
      "tag": "『proxy』_1",
      "protocol": "vmess",
      "settings": { ... }
    }
  ],
  "routing": {
    "balancers": [
      {
        "tag": "『proxy』",
        "selector": ["『proxy』_0", "『proxy』_1"],
        "strategy": { "type": "roundrobin" }
      }
    ],
    "rules": [{ "balancerTag": "『proxy』" }]
  }
}
```

Round robin is connection-level selection: each new proxied connection is assigned to the next selector member. It does not split an established connection across servers and does not itself provide health-based failover.

#### Groups without Load Balancing

When a group policy is disabled, v2rayA preserves the conventional connect behavior and routes through its first selected member. This keeps existing proxy groups backward-compatible while allowing the user to opt in per group.

#### Tag Format

All group-related outbound tags follow the format:
- Group tag: `『groupName』`
- Individual server tags: `『groupName』_N` where N is the server index

This consistent tagging ensures proper routing and load balancing functionality.

### Auto-Update Mechanisms
1. **Subscription Updates:**
   - Configurable intervals
   - Batch processing with concurrency limits
   - Auto-selection of servers

2. **GFW List Updates:**
   - Daily/interval-based
   - Background download and processing

3. **Version Checks:**
   - Weekly update checks
   - Non-intrusive notifications

## Process Management

### Startup Sequence
1. Environment check (root, ports, config)
2. Database initialization
3. Configuration loading
4. Update checks (GFW, subscriptions)
5. Kernel status restoration
6. Web server start

### Shutdown Sequence
1. Graceful signal handling (SIGINT, SIGTERM)
2. Transparent proxy cleanup
3. V2Ray/XRay process stop
4. Database close
5. Server shutdown

### Crash Recovery
- Last exit status tracking
- Automatic state cleanup
- Manual restart requirement after crashes

## Platform Abstractions

### OS-Specific Implementations
- **Windows:** Service mode, Windows Firewall integration
- **Linux:** Systemd integration, iptables, tproxy
- **macOS:** pf firewall, launchd integration

### Build Constraints
- Platform-specific driver selection
- Feature flags via build tags
- Conditional compilation

## Extension Points

1. **Protocol Support:** Add new protocols via core extensions
2. **UI Customization:** Theme and component overrides
3. **Routing Rules:** Custom RoutingA expressions
4. **Plugin System:** (Planned) External module integration

## Performance Considerations

### Caching Strategies
- Static file ETags and caching
- Request deduplication
- Response compression

### Concurrency
- Goroutine-based subscription updates
- Rate limiting for API requests
- Connection pooling for database

## Development Environment

### Backend
```bash
cd service
go mod download
go build -o v2raya
```

### Frontend
```bash
cd gui
yarn install
yarn serve  # Development
yarn build  # Production
```

### Testing
```bash
cd service
go test ./...

# Frontend
yarn lint
yarn test:unit  # If available
```
