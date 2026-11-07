# Database & Configuration Schema

## Overview

v2rayA uses an embedded SQLite database for persistent storage alongside structured JSON models for application configuration. This document describes the database schema, entity relationships, configuration models, migrations, and integrity constraints.

---

## 1. Relational Database Schema (SQLite)

Database initialization and migrations are managed in `service/db/schema.go` and `service/db/migrate.go`.

### 1.1 `system_config`

Key-value store for application configuration, active selection state, and dynamic system settings.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `key` | `TEXT` | `PRIMARY KEY` | Configuration identifier |
| `value` | `TEXT` | `NOT NULL` | Serialized value or string |

**Common Keys:**
- `active_outbound`: Name of the active default proxy group (default: `"proxy"`).
- `setting`: Serialized JSON of global application settings (`Setting` struct).
- `pac`: PAC rules and configuration.
- `routingA`: Custom RoutingA rule definitions.
- `dns_rules`: Custom DNS rules.

---

### 1.2 `servers`

Stores proxy server configurations imported manually, via link, or through subscriptions.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `INTEGER` | `PRIMARY KEY AUTOINCREMENT` | Unique server ID |
| `type` | `TEXT` | `NOT NULL DEFAULT 'server'` | Server type discriminator |
| `sub_id` | `INTEGER` | `DEFAULT NULL` | Foreign key referencing `subscriptions(id)` |
| `address` | `TEXT` | `NOT NULL DEFAULT ''` | Hostname or IP address |
| `port` | `INTEGER` | `NOT NULL DEFAULT 0` | Server port |
| `protocol` | `TEXT` | `NOT NULL DEFAULT ''` | Protocol (`vmess`, `vless`, `ss`, `ssr`, `trojan`, `tuic`, `juicity`, etc.) |
| `config_json` | `TEXT` | `NOT NULL DEFAULT '{}'` | Protocol-specific parameters in JSON |
| `intel` | `TEXT` | `DEFAULT ''` | Intelligence / probe annotations |
| `latency` | `TEXT` | `DEFAULT ''` | Measured latency (HTTP/TCP ping) |
| `link` | `TEXT` | `DEFAULT ''` | Raw import link (`vmess://...`, etc.) |
| `url` | `TEXT` | `DEFAULT ''` | Parsed canonical URL |
| `sort` | `INTEGER` | `NOT NULL DEFAULT 0` | Display sorting order |
| `group_id` | `TEXT` | `DEFAULT ''` | Group identifier |
| `created_at` | `DATETIME` | `DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `DATETIME` | `DEFAULT CURRENT_TIMESTAMP` | Record modification timestamp |

---

### 1.3 `subscriptions`

Stores remote server subscription endpoints.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `INTEGER` | `PRIMARY KEY AUTOINCREMENT` | Unique subscription ID |
| `address` | `TEXT` | `NOT NULL DEFAULT ''` | Subscription URL |
| `remarks` | `TEXT` | `NOT NULL DEFAULT ''` | User-defined label / name |
| `status` | `TEXT` | `NOT NULL DEFAULT ''` | Last update status (`success`, `failed`, etc.) |
| `info` | `TEXT` | `DEFAULT ''` | Subscription metadata or bandwidth info |
| `auto_select` | `INTEGER` | `NOT NULL DEFAULT 0` | Auto-select best node flag (0 or 1) |
| `filter` | `TEXT` | `DEFAULT ''` | Regex / keyword filter for node import |
| `group_id` | `TEXT` | `DEFAULT ''` | Associated group ID |
| `sort` | `INTEGER` | `NOT NULL DEFAULT 0` | Display sorting order |
| `created_at` | `DATETIME` | `DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `DATETIME` | `DEFAULT CURRENT_TIMESTAMP` | Record modification timestamp |

---

### 1.4 `accounts`

Stores administrative authentication credentials.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `INTEGER` | `PRIMARY KEY AUTOINCREMENT` | User account ID |
| `username` | `TEXT` | `UNIQUE NOT NULL` | Login username |
| `password_hash` | `TEXT` | `NOT NULL` | Securely hashed password (bcrypt) |
| `created_at` | `DATETIME` | `DEFAULT CURRENT_TIMESTAMP` | Account creation timestamp |

---

### 1.5 `outbound_names`

Defines configured proxy outbound groups (e.g., `proxy`, `custom_group_1`).

| Column | Type | Constraints | Description |
|---|---|---|---|
| `name` | `TEXT` | `PRIMARY KEY` | Outbound group name |
| `sort` | `INTEGER` | `NOT NULL DEFAULT 0` | Display sorting order |

---

### 1.6 `outbound_connections`

Many-to-many relationship mapping servers to outbound groups.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `id` | `INTEGER` | `PRIMARY KEY AUTOINCREMENT` | Mapping record ID |
| `outbound_name` | `TEXT` | `NOT NULL, REFERENCES outbound_names(name)` | Outbound group name |
| `server_id` | `INTEGER` | `NOT NULL, REFERENCES servers(id)` | Associated server ID |
| `sort` | `INTEGER` | `NOT NULL DEFAULT 0` | Order within the group |

**Constraints:**
- `UNIQUE(outbound_name, server_id)`: Prevents duplicate associations of the same server to a single outbound group.

---

### 1.7 `outbound_settings`


---

## 2. Configuration Data Structures

### 2.1 `OutboundSetting` (`service/db/configure/outbound.go`)

```go
type OutboundSetting struct {
    ProbeURL            string          `json:"probeURL"`
    ProbeInterval       string          `json:"probeInterval"`
    Type                ObservatoryType `json:"type"`
    LoadBalanceEnabled  bool            `json:"loadBalanceEnabled"`
    LoadBalanceStrategy string          `json:"loadBalanceStrategy"`
}
```

- `probeURL`: Endpoint used by the observatory to measure latency (default: `https://www.google.com/generate_204`).
- `probeInterval`: Observatory ping interval (default: `1m`, fallback `10s`).
- `type`: Observatory selection type (`leastping`).
- `loadBalanceEnabled`: Enables per-group round-robin connection balancing across all connected servers.
- `loadBalanceStrategy`: Algorithm for load balancing (`roundrobin`).

### 2.2 `Setting` (`service/db/configure/setting.go`)

Global application configuration persisted in `system_config` under key `"setting"`:
- **Transparent proxy:** `transparent` mode (`close`, `tproxy`, `redirect`), `transparentType`, `ipforward`, `tproxyExcludedInterfaces`.
- **Port settings:** `rulePortMode`, `portSharing`, `tcpFastOpen`, `muxOn`, `mux`.
- **Auto-update:** `pacAutoUpdateMode`, `subscriptionAutoUpdateMode`, update intervals.
- **DNS module & caching:** `dnsListenAddr`, `dnsCacheEnabled`, `dnsCacheSize`, `dnsCacheMinTTL`, `dnsCacheMaxTTL`, `dnsPrefetch`, `dnsNegativeCache`.
- **Traffic sniffing:** `inboundSniffing` (`http,tls,quic`).

---

## 3. Migrations & Schema Evolution

- `InitSchema(db *sql.DB)`: Executes `CREATE TABLE IF NOT EXISTS` for all core tables.
- `MigrateSchema(db *sql.DB)`: Applies incremental migrations (e.g., adding `remarks` or `auto_select` columns to `subscriptions` using `ALTER TABLE` if not present).
- `Migrate(db *sql.DB)`: Executes data migrations between versions (e.g., migrating v2rayA legacy JSON/bolt databases to SQLite).

---

## 4. Preservation Rules During Upstream Sync

1. **Table Schema Invariants:**
   - The tables `outbound_names`, `outbound_connections`, and `outbound_settings` are crucial for multi-server connections and load balancing.
   - Do not alter or drop `outbound_settings.setting_json` as it holds `LoadBalanceEnabled` and `LoadBalanceStrategy`.
2. **`system_config:active_outbound` Invariant:**
   - Must remain stored in `system_config`.
   - Must self-heal to `"proxy"` if missing or pointing to a deleted group.


Stores per-outbound configuration such as health checks and load balancing.

| Column | Type | Constraints | Description |
|---|---|---|---|
| `outbound_name` | `TEXT` | `PRIMARY KEY, REFERENCES outbound_names(name)` | Outbound group name |
| `setting_json` | `TEXT` | `NOT NULL DEFAULT '{}'` | Serialized `OutboundSetting` JSON |
