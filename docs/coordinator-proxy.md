# Coordinator Proxy

The **Coordinator Proxy** is a standalone HTTP reverse-proxy application that sits between prover clients and multiple upstream Scroll L2 Coordinators. It exposes the same REST API surface as a real coordinator, allowing provers to connect transparently without knowing that requests are being load-balanced across a pool of backend coordinators.

---

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Key Components](#key-components)
- [Data Structures](#data-structures)
- [API Endpoints](#api-endpoints)
- [Authentication Flow](#authentication-flow)
- [Task Routing](#task-routing)
- [Configuration](#configuration)
- [Usage](#usage)
- [Design Decisions](#design-decisions)

---

## Overview

In a production environment, a single coordinator may not be sufficient to serve a large fleet of provers. The coordinator proxy addresses this by:

- **Multiplexing** a single prover connection across multiple upstream coordinators.
- **Authenticating** provers locally using the same JWT challenge-response mechanism as a real coordinator.
- **Maintaining per-upstream, per-prover sessions** (including bearer tokens) so each coordinator sees the prover as a direct client.
- **Routing `get_task` and `submit_proof` requests** intelligently across upstreams with sticky priority and random fallback.

Because the proxy exposes the standard coordinator API (`/coordinator/v1/...`), existing prover SDKs require no changes to work through the proxy.

---

## Architecture

```
┌─────────────┐      ┌─────────────────────┐      ┌─────────────────┐
│ Prover SDK  │─────>│ Coordinator Proxy   │─────>│ Coordinator A   │
└─────────────┘      │  (port 8590)        │      └─────────────────┘
                     │                     │      ┌─────────────────┐
                     │  • Local Auth       │─────>│ Coordinator B   │
                     │  • Session Manager  │      └─────────────────┘
                     │  • Task Router      │      ┌─────────────────┐
                     │  • Token Cache      │─────>│ Coordinator C   │
                     └─────────────────────┘      └─────────────────┘
```

### Entry Point

| File | Role |
|------|------|
| `coordinator/cmd/proxy/main.go` | Thin wrapper that invokes the CLI application. |
| `coordinator/cmd/proxy/app/app.go` | Bootstraps the proxy: parses `ProxyConfig`, optionally initializes the database, and starts the HTTP server with graceful shutdown. |
| `coordinator/cmd/proxy/app/flags.go` | Defines HTTP server flags (`--http`, `--http.addr`, `--http.port`). Default port is `8590`. |

---

## Key Components

### 1. Auth Controller (`auth.go`)

Handles the `/login` endpoint locally, then fans out a **proxy-login** to every upstream coordinator asynchronously.

- Runs standard challenge-response validation (reuses existing coordinator auth logic).
- On success, spawns goroutines to call `proxy_login` on each upstream.
- Stores returned upstream tokens in the prover's session.

### 2. GetTask Controller (`get_task.go`)

Routes prover task requests to the most appropriate upstream.

- First tries the **priority upstream** (sticky routing from a previous successful task assignment).
- If the priority upstream has no task, shuffles the remaining upstreams randomly and tries each until one returns a task.
- Prefixes the returned `taskID` with the upstream name (`upstream:taskID`) so that `submit_proof` can route correctly.

### 3. SubmitProof Controller (`submit_proof.go`)

Forwards proof submissions to the correct upstream coordinator.

- Parses the `upstream:taskID` prefix to determine the target coordinator.
- Uses the cached upstream token from the prover session.
- On success, clears the priority upstream (task is complete).

### 4. Client Manager (`client_manager.go`)

Manages the proxy's own identity and bearer token for each upstream.

- Keeps a cached `upClient` with a background re-login goroutine.
- If the cached token expires or becomes invalid, `Reset()` clears it so a fresh login is attempted.

### 5. Prover Session Manager (`prover_session.go`)

Maintains in-memory (and optionally DB-backed) sessions for every prover public key.

- Each session holds a map of `upstream -> loginToken`.
- Transparently refreshes expired tokens.
- Implements a session size limit; when exceeded, the old session map is rotated to a deprecated map rather than deleted immediately.

### 6. Priority Upstream Manager

Stores sticky routing hints (`publicKey -> upstreamName`).

- When a prover successfully receives a task from an upstream, that upstream becomes the priority for the next `get_task` call.
- Can be persisted to the database so routing preferences survive restarts.

### 7. HTTP Client (`client.go`)

Low-level HTTP client (`upClient`) that implements the `ProxyCli` and `ProverCli` interfaces.

- `ProxyCli`: used by the proxy itself to log in to an upstream.
- `ProverCli`: used to impersonate a prover when calling `get_task` or `submit_proof` on an upstream.

---

## Data Structures

### Configuration

```go
// ProxyConfig — top-level configuration
type ProxyConfig struct {
    ProxyManager *ProxyManager        `json:"proxy_manager"`
    ProxyName    string               `json:"proxy_name"`
    Coordinators map[string]*UpStream `json:"coordinators"`
}

// ProxyManager — auth, verifier, client identity, optional DB
type ProxyManager struct {
    Verifier *VerifierConfig  `json:"verifier"`   // minimum prover version
    Client   *ProxyClient     `json:"proxy_cli"`  // proxy's own identity
    Auth     *Auth            `json:"auth"`       // JWT secret & expiry
    DB       *database.Config `json:"db,omitempty"`
}

// ProxyClient — identity the proxy uses to authenticate with upstreams
type ProxyClient struct {
    ProxyName    string `json:"proxy_name"`
    ProxyVersion string `json:"proxy_version,omitempty"`
    Secret       string `json:"secret,omitempty"`
}

// UpStream — per-coordinator connection settings
type UpStream struct {
    BaseUrl              string `json:"base_url"`
    RetryCount           uint   `json:"retry_count"`
    RetryWaitTime        uint   `json:"retry_wait_time_sec"`
    ConnectionTimeoutSec uint   `json:"connection_timeout_sec"`
    CompatibileMode      bool   `json:"compatible_mode,omitempty"`
}
```

### Runtime Data Structures

```go
// Client interface — abstracts per-upstream access
type Client interface {
    Client(string) ProverCli                // token-bound prover client
    ClientAsProxy(context.Context) ProxyCli // proxy's own authenticated client
    Name() string
}

// upClient — actual HTTP implementation
type upClient struct {
    httpClient      *http.Client
    baseURL         string
    loginToken      string
    compatibileMode bool
    resetFromMgr    func()
}

// ProverManager — registry of active prover sessions
type ProverManager struct {
    data               map[string]*proverSession
    willDeprecatedData map[string]*proverSession
    sizeLimit          int
    persistent         *proverDataPersist
    // ... prometheus metrics
}

// proverSession — per-prover tokens across all upstreams
type proverSession struct {
    persistent    *proverDataPersist
    proverToken   map[string]loginToken
    completionCtx context.Context
}

// loginToken — upstream token with a monotonic phase
type loginToken struct {
    token string
    phase uint
}
```

### Database Models (Optional Persistence)

When a database is configured, the proxy persists the following tables:

| Table | Columns | Purpose |
|-------|---------|---------|
| `prover_sessions` | `public_key`, `upstream`, `up_token`, `expired` | Stores upstream tokens per prover so restarts do not force re-login. |
| `priority_upstream` | `public_key`, `upstream` | Restores sticky routing preferences after restart. |

---

## API Endpoints

### Exposed API (Prover → Proxy)

All endpoints are mounted under `/coordinator/v1/`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/challenge` | None | Returns a challenge token for the prover to sign. |
| `POST` | `/login` | Challenge | Validates the prover signature, then fans out `proxy_login` to all upstreams. |
| `POST` | `/get_task` | JWT | Routes a task request to the best available upstream. |
| `POST` | `/submit_proof` | JWT | Forwards the proof to the upstream that issued the task. |

### Upstream API Calls (Proxy → Coordinator)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/coordinator/v1/challenge` | Obtain a challenge token for proxy login. |
| `POST` | `/coordinator/v1/login` | Proxy authenticates itself as a client. |
| `POST` | `/coordinator/v1/proxy_login` | Forward prover identity to the upstream. |
| `POST` | `/coordinator/v1/get_task` | Forward prover task request. |
| `POST` | `/coordinator/v1/submit_proof` | Forward proof submission. |

### Key Interfaces

```go
// ProxyCli — proxy's own client to an upstream
type ProxyCli interface {
    Login(ctx context.Context, genLogin func(string) (*types.LoginParameter, error)) (*ctypes.Response, error)
    ProxyLogin(ctx context.Context, param *types.LoginParameter) (*ctypes.Response, error)
    Token() string
    Reset()
}

// ProverCli — prover-impersonating client to an upstream
type ProverCli interface {
    GetTask(ctx context.Context, param *types.GetTaskParameter) (*ctypes.Response, error)
    SubmitProof(ctx context.Context, param *types.SubmitProofParameter) (*ctypes.Response, error)
}
```

---

## Authentication Flow

### 1. Prover → Proxy `/login`

1. Prover requests a challenge from the proxy (`GET /challenge`).
2. Prover signs the challenge and sends it to `POST /login`.
3. The proxy validates the signature locally using the same logic as a real coordinator.
4. On success, the proxy spawns asynchronous goroutines to call `POST /proxy_login` on **every** upstream coordinator.
5. Each upstream returns a bearer token specific to that prover.
6. The proxy stores all upstream tokens in the prover's session and returns its own JWT to the prover.

### 2. Token Refresh & Resilience

Both `GetTask` and `SubmitProof` implement a retry-on-token-expiry pattern:

1. Try the request with the cached upstream token.
2. If the upstream returns `ErrJWTTokenExpired` or `ErrJWTCommonErr`, trigger `maintainLogin` to refresh the token.
3. Retry the request once with the new token.

The `ClientManager` also maintains a background login goroutine for the proxy's own identity; if the cached client fails, `Reset()` clears it so a fresh login is attempted on the next call.

---

## Task Routing

### Sticky Priority Routing

When a prover successfully receives a task from an upstream, that upstream is recorded as the **priority upstream** for that prover. On the next `get_task` call, the proxy tries the priority upstream first.

### Random Fallback

If the priority upstream has no available task (or returns an error), the proxy shuffles the remaining upstreams randomly and tries each one until a task is returned or all upstreams are exhausted.

### Task ID Namespacing

To ensure `submit_proof` routes to the correct upstream, the proxy prefixes the task ID returned to the prover:

```
upstreamName:originalTaskID
```

The `SubmitProof` controller parses this prefix, extracts the upstream name, and forwards the proof to the correct coordinator.

---

## Configuration

Example `config_proxy.json`:

```json
{
  "proxy_manager": {
    "proxy_cli": {
      "proxy_name": "proxy_name",
      "secret": "client private key"
    },
    "auth": {
      "secret": "proxy secret key",
      "challenge_expire_duration_sec": 3600,
      "login_expire_duration_sec": 3600
    },
    "verifier": {
      "min_prover_version": "v4.4.45",
      "verifiers": []
    },
    "db": {
      "driver_name": "postgres",
      "dsn": "postgres://localhost/scroll?sslmode=disable",
      "maxOpenNum": 200,
      "maxIdleNum": 20
    }
  },
  "coordinators": {
    "sepolia": {
      "base_url": "http://localhost:8555",
      "retry_count": 10,
      "retry_wait_time_sec": 10,
      "connection_timeout_sec": 30
    }
  }
}
```

### Field Reference

| Field | Description |
|-------|-------------|
| `proxy_manager.proxy_cli.secret` | ECDSA private key material used to derive the proxy's signing key for upstream login. |
| `proxy_manager.auth.secret` | JWT HMAC key for prover-to-proxy sessions. |
| `proxy_manager.verifier.min_prover_version` | Minimum prover version allowed to connect. |
| `proxy_manager.db` | Optional database configuration. If omitted, the proxy runs in memory-only mode (no persistence across restarts). |
| `coordinators.*.base_url` | HTTP endpoint of the upstream coordinator. |
| `coordinators.*.compatible_mode` | If `true`, skips `proxy_login` and uses standard login with a dummy token (for legacy coordinators). |

---

## Usage

### Build

```bash
cd coordinator
make proxy
```

### Run

```bash
./build/bin/coordinator_proxy --config conf/config_proxy.json
```

### Run with custom HTTP address

```bash
./build/bin/coordinator_proxy --config conf/config_proxy.json --http.addr 0.0.0.0 --http.port 8590
```

### Prover Connection

Provers connect to the proxy exactly as they would connect to a real coordinator:

```
https://proxy.example.com/coordinator/v1
```

No SDK changes are required.

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Sticky routing for tasks** | Reduces cross-coordinator state churn; a prover that received a task from upstream A is likely to get the next task from the same upstream. |
| **Random load balancing as fallback** | Simple and stateless; avoids hot-spotting when one upstream runs out of tasks. |
| **Task ID namespacing** | Allows the proxy to remain stateless for proof submissions; the task ID itself encodes the routing target. |
| **Session size limit with rotation** | Prevents unbounded memory growth in the proxy; old sessions are moved to a deprecated map and eventually garbage-collected. |
| **Phase-based token updates** | A monotonic `phase` counter on `loginToken` prevents stale concurrent login attempts from overwriting a fresher token. |
| **Compatible mode** | Allows the proxy to work with older coordinators that do not support the `proxy_login` endpoint. |
| **Optional DB persistence** | The proxy can run entirely in memory for simplicity, or use a database for token and routing persistence across restarts. |
