# Specification

## Functional Requirements

### 1. Core Proxy Functionality

#### 1.1 Supported Protocols

| Protocol | Version Support | Features |
|----------|----------------|----------|
| VMess | V2Ray/XRay | UUID, AlterID, TLS, WebSocket, HTTP/2, gRPC |
| VLESS | XRay | UUID, Reality, Vision, XTLS |
| Shadowsocks | SS | Multiple ciphers, AEAD support |
| ShadowsocksR | SSR | Protocol obfuscation, custom settings |
| Trojan | Trojan-Go | TLS-based, WebSocket support |
| TUIC | TUIC v5 | QUIC-based, UDP support |
| Juicity | Juicity | QUIC-based, lightweight |

#### 1.2 Proxy Modes

1. **Transparent Proxy (Linux)**
   - TProxy-based traffic interception
   - Full system traffic routing
   - Process-based exclusion rules
   - IP-based white lists

2. **System Proxy (Windows/macOS)**
   - OS-level proxy configuration
   - PAC file support for automatic configuration
   - Manual proxy settings

3. **Per-Application Proxy**
   - Environment variable configuration
   - Application-specific routing
   - SOCKS5/HTTP proxy endpoints

### 2. Server Management

#### 2.1 Server Import Methods

1. **Manual Entry**
   - Server address and port
   - Protocol-specific parameters
   - Remarks and group assignments

2. **Link Import**
   - VMess:// links
   - SS/SSR links
   - Trojan:// links
   - TUIC/Juicity links

3. **Subscription Management**
   - URL-based subscriptions
   - Auto-update intervals
   - Filter rules (by protocol, latency, etc.)
   - Batch operations

#### 2.2 Server Operations

- **Connection:** Connect to selected server
- **Latency Testing:** HTTP and ICMP ping
- **Sharing:** Generate shareable links
- **Grouping:** Organize servers into groups
- **Sorting:** Manual and automatic sorting
- **Import/Export:** JSON and link formats

### 3. Routing System

#### 3.1 Routing Modes

1. **Default**
   - Direct connection for local IPs
   - Proxy for other traffic

2. **GFW List**
   - Direct for domestic sites
   - Proxy for blocked sites

3. **Custom Rules**
   - RoutingA expression language
   - Protocol, domain, IP-based rules
   - Priority-based rule matching

#### 3.2 RoutingA Syntax

```
# Example rules
domain(suffix:google.com) -> proxy
domain(geosite:cn) -> direct
ip(geoip:private) -> direct
# ...
```

#### 3.3 DNS Configuration

- Custom DNS servers
- DNS-over-HTTPS support
- Anti-pollution mechanisms
- Domain-based DNS rules

### 4. Network Features

#### 4.1 Port Configuration

| Port Type | Default | Description |
|-----------|---------|-------------|
| SOCKS5 | 2080 | SOCKS5 proxy |
| HTTP | 2081 | HTTP proxy |
| V2Ray/XRay | Dynamic | Internal ports |

#### 4.2 Transparent Proxy Settings

- IP forwarding configuration
- TProxy module detection
- Port white lists
- Process exclusion rules
- IP group white lists

#### 4.3 Anti-Pollution

- Multiple DNS resolvers
- Domain blocking rules
- GFW list integration

### 5. Subscription System

#### 5.1 Subscription Features

- Auto-update scheduling
- Filter rules
- Auto-selection of servers
- Batch operations
- Error handling and retries

#### 5.2 Supported Formats

- Base64 encoded lists
- JSON configurations
- Standard subscription formats

### 6. Load Balancing

#### 6.1 Load Balancer Configuration

Load balancing is enabled independently for each proxy group. It distributes new proxy connections across the selected members of that group using the configured algorithm; `roundrobin` is the only currently supported algorithm.

**Configuration Structure:**
- `OutboundSetting.loadBalanceEnabled` defaults to `false`; `loadBalanceStrategy` defaults to `roundrobin`.
- An enabled group with two or more selected members creates individual outbounds with tags `『groupName』_0`, `『groupName』_1`, etc.
- Xray `routing.balancers` contains group tag `『groupName』`, an exact selector of the individual tags, and `strategy.type: "roundrobin"`.
- Rules that target the raw group name or wrapped group tag use Xray `balancerTag` to select the routing balancer.
- A disabled or one-member group has no balancer and uses its first selected member as an ordinary outbound.

**Key Implementation Details:**
- When the policy is enabled for a group with multiple members, the system creates individual outbound configurations and one routing balancer that references all of their tags.
- The balancer selector must match the actual outbound tags for proper operation
- The selected Xray round-robin strategy assigns new connections sequentially. Existing connections are not migrated or split across servers.

**Error Handling:**
- "empty selector list" error occurs if balancer selectors don't match any outbound tags
- This is prevented by ensuring selector tags use the same format as outbound tags

#### 6.2 Outbound Group Types

1. **Balancer (Load Balancing)**
    - Opt-in per proxy group; at least two selected members are needed to distribute traffic.
    - Round-robin distribution is performed by Xray for each new proxy connection.
    - Changing the group policy regenerates and restarts the active core configuration through the existing outbound settings API.

2. **Selector (Manual Selection)**
   - Users manually select which server to use
   - GUI provides server switching interface
   - Also uses group outbound structure

#### 6.3 Observatory Integration

The round-robin strategy does not require observatory data and does not provide health-based failover. Observatory settings remain available for latency-related strategies that may be added later.

## User Interface Requirements

### 1. Dashboard

- Connection status display
- Current server information
- Quick connect/disconnect
- Traffic statistics

### 2. Server List

- Sortable server table
- Latency indicators
- Protocol icons
- Connection status markers
- Context menu operations
- The local-server table and individual subscription tabs provide independent protocol and encryption filters in visually separate control containers. Options are generated from the corresponding server list; applying both filters returns servers matching both values. Filter selections survive tab navigation for the active page session.
- VMess servers with TLS enabled and encryption `none` or `false` remain visible with a disable reason but cannot be selected, connected, added to a proxy group, or added by the latency bulk action. VLESS+TLS is always allowed since VLESS protocol encryption is `none` by design and TLS provides transport security.
- VLESS servers with `security=false` or `security=none` (no TLS) are universally blocked since VLESS always uses protocol encryption `none`, making plaintext traffic insecure regardless of whether the address is an IP or domain.
- “Bulk Add by Latency” is available only while the local-server table or an individual subscription-server table is active; it is unavailable from the subscription overview and Live Flow tabs.
- Node-management tables use responsive panel headers, count badges, filter toolbars, and empty-result states in both light and dark themes.

### 3. Settings Interface

- Tabbed configuration panels
- Real-time validation
- Help tooltips
- Reset to defaults option

### 4. Modal Dialogs

- Server editing modal
- Settings modal
- Subscription management modal
- Import/export dialogs

### 5. Live Flow Topology

- The **LIVE FLOW** tab displays an animated SVG route topology: source → selected proxy server/proxy chain → destination.
- It filters observed routes by protocol, status, and selected proxy.
- It must not represent observatory reachability or latency probes as user traffic.
- Clicking a node or path shows the selected route metadata.
- The graph supports dark theme, responsive horizontal scrolling, and reduced-motion preferences.
- Current core route events supply source, destination, protocol, timestamp, and selected detour. Per-session byte/speed metrics and definitive close events are unavailable unless a future core metrics producer supplies them.

## API Specification

### 1. Authentication

#### Login
```http
POST /api/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

Response:
```json
{
  "token": "jwt_token_here",
  "expires": 86400
}
```

### 2. Server Operations

#### List Servers
```http
GET /api/touch
Authorization: Bearer {token}
```

Response server entries include canonical `protocol` and `encryptions` fields in addition to the display-oriented `net` field. `encryptions` is an array because a server may have both a protocol cipher and transport security (for example, VMess cipher plus TLS).

Response:
```json
[
  {
    "id": 1,
    "name": "US Server",
    "address": "us.example.com",
    "port": 443,
    "protocol": "vmess",
    "connected": true,
    "latency": 150
  }
]
```

#### Import Server
```http
POST /api/import
Authorization: Bearer {token}
Content-Type: application/json

{
  "link": "vmess://..."
}
```

#### Connect to Server
```http
POST /api/connection
Authorization: Bearer {token}
Content-Type: application/json

{
  "id": 1,
  "type": "server"
}
```

### 3. Configuration

#### Get Settings
```http
GET /api/setting
Authorization: Bearer {token}
```

#### Update Settings
```http
PUT /api/setting
Authorization: Bearer {token}
Content-Type: application/json

{
  "transparent": "tproxy",
  "socksPort": 2080,
  "httpPort": 2081
}
```

### 4. Routing

#### Get Routing Rules
```http
GET /api/routingA
Authorization: Bearer {token}
```

#### Update Routing Rules
```http
PUT /api/routingA
Authorization: Bearer {token}
Content-Type: application/json

{
  "routingA": "domain(suffix:google.com) -> proxy\n# ..."
}
```

### 5. Live Flow WebSocket

```text
GET /api/live-flow?token={token}
Upgrade: websocket
```

- Authentication uses the existing JWT protection. Browser clients use the `token` query parameter; `Authorization` remains accepted for diagnostic compatibility.
- A browser `Origin` host must match the request host. Origin-less non-browser diagnostic clients are allowed.
- The server emits `batch_state`, `flow_start`, `flow_update`, and `flow_end` messages with a `{ "type", "data" }` envelope.
- `flow_start.data` includes source endpoint, selected `proxy_chain`, destination endpoint, protocol, and timestamp.
- The endpoint describes observed routes. It does not expose observatory health snapshots as flows.

## Non-Functional Requirements

### 1. Performance

- **Startup Time:** < 2 seconds
- **API Response:** < 100ms for local operations
- **Concurrent Connections:** Support 1000+ simultaneous connections
- **Memory Usage:** < 100MB base memory

### 2. Reliability

- **Uptime:** 99.9% availability
- **Crash Recovery:** Automatic state restoration
- **Data Persistence:** No data loss on restart

### 3. Security

- **Authentication:** JWT with secure defaults
- **Encryption:** TLS for API communication
- **Input Validation:** All user inputs sanitized
- **Privilege Escalation:** Proper root/admin checks

### 4. Compatibility

- **OS Support:** Linux, Windows, macOS
- **Browser Support:** Chrome 80+, Firefox 75+, Safari 13+, Edge 80+
- **Architecture:** amd64, arm64, armv7, mips (soft-float)

### 5. Usability

- **Learning Curve:** < 5 minutes for basic usage
- **Error Messages:** Clear and actionable
- **Help System:** Contextual tooltips and documentation

## Data Models

### 1. Server

```json
{
  "id": "integer",
  "type": "string",
  "sub_id": "integer|null",
  "address": "string",
  "port": "integer",
  "protocol": "string",
  "config_json": "string",
  "intel": "string",
  "latency": "string",
  "link": "string",
  "url": "string",
  "sort": "integer",
  "group_id": "string",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### 2. Subscription

```json
{
  "id": "integer",
  "address": "string",
  "remarks": "string",
  "status": "string",
  "info": "string",
  "auto_select": "boolean",
  "filter": "string",
  "group_id": "string",
  "sort": "integer",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

### 3. Settings

```json
{
  "transparent": "enum",
  "socksPort": "integer",
  "httpPort": "integer",
  "pacMode": "enum",
  "gfwListAutoUpdateMode": "enum",
  "subscriptionAutoUpdateMode": "enum",
  "ipForward": "boolean"
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `V2RAYA_CONFIG_DIR` | `~/.config/v2raya` | Configuration directory |
| `V2RAYA_ASSET_DIR` | `<config>/asset` | V2Ray asset directory |
| `V2RAYA_ADDRESS` | `127.0.0.1:2017` | Listening address |
| `V2RAYA_VERBOSE` | `false` | Verbose logging |

## Command-Line Options

| Option | Description |
|--------|-------------|
| `--config <dir>` | Configuration directory |
| `--lite` | Run without root privileges |
| `--reset-password` | Reset admin password |
| `--address <addr>` | Listening address |
| `--web-dir <dir>` | Custom web directory |
| `--passcheckroot` | Skip root check |
| `--print-report` | Print system report |

## Error Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 400 | Bad request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not found |
| 500 | Internal server error |
| 1001 | Invalid server link |
| 1002 | Connection failed |
| 1003 | Subscription update failed |

## Compliance

- **License:** AGPL-3.0-only
- **Privacy:** No cloud data storage
- **Security:** Local data only
- **Accessibility:** WCAG 2.1 AA (target)
