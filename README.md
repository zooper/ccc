# Community Connectivity Check (CCC)

A lightweight, privacy-focused connectivity monitoring tool for multi-tenant buildings. CCC helps residents answer the question: "Is it my connection, or is the whole building affected?"

## Features

- **Opt-in Monitoring**: Residents voluntarily join connectivity monitoring
- **ISP Detection**: Automatic ISP classification via ASN lookup
- **Real-time Dashboard**: View aggregated connectivity status by ISP
- **Outage Detection**: Configurable thresholds for detecting ISP-wide issues
- **Privacy First**: No personal information collected, anonymous participation
- **Single Binary**: Self-contained deployment with embedded frontend

## Screenshots

The dashboard shows real-time ISP status with the number of endpoints up/down per provider. When connectivity issues are widespread, an outage warning is displayed.

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 18+
- SQLite3

### Build

```bash
# Build frontend and backend
make build

# Set admin password
./bin/ccc-api --set-password <your-password>

# Run the server
./bin/ccc-api
```

The server starts at `http://localhost:8080` by default.

### Development

```bash
# Terminal 1: Run the API server with hot reload
make dev

# Terminal 2: Run the frontend dev server
cd web && npm run dev
```

## Configuration

| Environment Variable | Flag | Default | Description |
|---------------------|------|---------|-------------|
| `CCC_DB_PATH` | `--db` | `./ccc.db` | SQLite database path |
| `CCC_LISTEN_ADDR` | `--listen` | `:8080` | Server listen address |
| `CCC_PING_INTERVAL` | `--ping-interval` | `60s` | Monitoring interval |
| `CCC_EXPIRE_DAYS` | `--expire-days` | `3` | Days before inactive endpoints expire |
| `CCC_TRUSTED_PROXIES` | `--trusted-proxies` | | Comma-separated trusted proxy IPs |
| `CCC_ISP_CONFIG` | `--isp-config` | | Path to ISP configuration JSON |

## How It Works

1. **Visitor arrives**: The system identifies their ISP via IP-to-ASN lookup
2. **Opt-in**: If eligible, they can join the monitoring pool
3. **Monitoring**: Every 60 seconds, all endpoints are pinged in parallel
4. **Dashboard**: Aggregated results show which ISPs are experiencing issues

### Privacy

CCC stores only what's necessary for monitoring:
- IP address (for connectivity checks)
- ISP name (for grouping)
- Connection status and timestamps

No personal information, emails, or identifiers are collected. The public dashboard shows only aggregated statistics.

## Architecture

```
ccc/
├── cmd/ccc-api/          # Application entrypoint
├── internal/
│   ├── api/              # HTTP handlers and routes
│   ├── isp/              # ASN-based ISP classification
│   ├── monitor/          # Ping scheduler and workers
│   ├── storage/          # SQLite database layer
│   └── models/           # Data structures
└── web/                  # React frontend (TypeScript)
```

### Tech Stack

- **Backend**: Go with stdlib HTTP server
- **Frontend**: React + TypeScript + Vite
- **Database**: SQLite
- **Deployment**: Single binary with embedded assets

## API Endpoints

### Public

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/status` | Visitor's ISP and registration status |
| POST | `/api/register` | Join monitoring |
| GET | `/api/dashboard` | Aggregated ISP statistics |
| GET | `/api/events` | Recent status changes |

### Admin (requires authentication)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/admin/endpoints` | List all monitored endpoints |
| POST | `/api/admin/endpoints` | Manually add an endpoint |
| DELETE | `/api/admin/endpoints/{id}` | Remove an endpoint |
| GET | `/api/admin/metrics` | System metrics and statistics |
| GET | `/api/admin/settings` | Get configuration settings |
| PUT | `/api/admin/settings` | Update settings |

## Deployment

### Systemd Service

```ini
[Unit]
Description=Community Connectivity Check API
After=network.target

[Service]
Type=simple
User=ccc
ExecStart=/opt/ccc/ccc-api --db /opt/ccc/ccc.db --listen 127.0.0.1:8090
Restart=always
AmbientCapabilities=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
```

### ICMP Permissions

For ping to work without root:

```bash
# Option 1: Set capability on binary
sudo setcap cap_net_raw+ep ./bin/ccc-api

# Option 2: Enable unprivileged ICMP (system-wide)
echo "net.ipv4.ping_group_range = 0 2147483647" | sudo tee /etc/sysctl.d/99-ping.conf
sudo sysctl --system
```

### Reverse Proxy (Caddy)

```
ccc.example.com {
    reverse_proxy localhost:8090
}
```

## License

MIT
