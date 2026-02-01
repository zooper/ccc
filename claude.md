# Community Connectivity Check (CCC)

## Purpose
CCC is a lightweight, opt-in monitoring service for a multi-tenant building with multiple ISPs. The goal is to give residents a simple, fast "is it me or is it the building" signal.

The system classifies visitors by ISP using IP → ASN matching, then (optionally) registers a monitoring target and aggregates status into an anonymized dashboard.

## High-level Architecture
- **Backend API (Go)**
  - Registration / deduplication
  - ISP classification (ASN-based via Team Cymru DNS)
  - Parallel ping monitoring with worker pool (50 concurrent workers)
  - Event tracking for status changes (down/up/outage/recovery)
  - Configurable outage detection threshold
  - Admin API with bcrypt password authentication
  - Rate limiting and security middleware
- **Frontend (React + TypeScript)**
  - Dashboard with ISP status cards
  - Events feed showing status changes
  - About page with configurable content
  - Admin panel for endpoint management, metrics, settings, and site config
  - Light/dark theme support
- **Deployment**
  - Single binary with embedded frontend (go:embed)
  - Caddy reverse proxy with automatic TLS
  - Systemd service on Oracle Cloud (uptimekuma)

## Core Flow
1. User visits the site.
2. Backend identifies ISP via client IPv4 ASN lookup.
3. If IPv4 is already registered:
   - Show dashboard view directly.
4. If IPv4 is new and ISP is supported:
   - Show opt-in prompt for monitoring.
   - If user accepts, add to monitoring set.
5. Monitoring runs every 60 seconds (parallel pinging) and updates status.
6. Status changes are recorded as events.

## Monitoring Logic
- **Direct ping only**: Each endpoint is pinged directly via ICMP
- **Parallel execution**: Worker pool with 50 concurrent ping workers
- **Status values**:
  - `up` - Endpoint responds to ping
  - `down` - Endpoint was previously up but no longer responds
  - `unreachable` - Endpoint has never responded (router may block ICMP)
  - `unknown` - Initial state before first check

## Privacy & Data Minimization
Design principle: store the minimum required to run monitoring.

Stored:
- Client IPv4 (for monitoring)
- Derived ISP category (via ASN match)
- Internal anonymized ID (e.g., `CCC-Endpoint-abc123`)
- Timestamps (created_at, last_seen, last_ok)
- Status (up/down/unreachable/unknown)

NOT stored:
- Names, emails, phone numbers, unit numbers
- User agents or personal information

Dashboard:
- Never displays raw IP addresses publicly.
- Only shows anonymized stats and event history.

## Data Retention / Cleanup
Automatic lifecycle rules:
- Endpoints not seen for 3 days are expired.
- Uptime history older than 7 days is cleaned up.
- Events older than 7 days are cleaned up.

## Security Features
- **Rate Limiting**: Token bucket algorithm (100 req/s general, 5 req/s for auth)
- **Trusted Proxies**: Configurable X-Forwarded-For handling
- **Body Size Limits**: 1MB max request body
- **Admin Auth**: bcrypt password hashing
- **IP Validation**: Private/internal IPs blocked from admin endpoint creation
- **SQL Injection Prevention**: All queries use parameterized statements

## Build Outputs
- `bin/ccc-api` - Go binary with embedded frontend
- `web/dist` - Static frontend build directory

## Quick Start

```bash
# Build everything (frontend + backend)
make build

# Set admin password
./bin/ccc-api --set-password <your-password>

# Run the server
./bin/ccc-api

# Or build and run in one step
make run
```

## Development

```bash
# Terminal 1: Run the API server
make dev

# Terminal 2: Run the frontend with hot reload
cd web && npm run dev
```

## Configuration

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| CCC_DB_PATH | --db | ./ccc.db | SQLite database path |
| CCC_LISTEN_ADDR | --listen | :8080 | Server listen address |
| CCC_PING_INTERVAL | --ping-interval | 60s | How often to ping endpoints |
| CCC_EXPIRE_DAYS | --expire-days | 3 | Days before inactive endpoints expire |
| CCC_PRIVILEGED | --privileged | false | Use raw socket ICMP (requires CAP_NET_RAW) |
| CCC_TRUSTED_PROXIES | --trusted-proxies | | Comma-separated trusted proxy IPs/CIDRs |
| CCC_CORS_ORIGIN | --cors-origin | | Allowed CORS origin |
| CCC_ISP_CONFIG | --isp-config | | Path to ISP configuration JSON |

## API Endpoints

### Public
| Method | Path | Description |
|--------|------|-------------|
| GET | /api/health | Health check |
| GET | /api/status | Get visitor's ISP and registration status |
| POST | /api/register | Opt-in for monitoring |
| GET | /api/dashboard | Aggregated ISP statistics |
| GET | /api/events | Recent status change events |
| GET | /api/site-config | Site configuration (name, about content, etc.) |

### Admin (Basic Auth required)
| Method | Path | Description |
|--------|------|-------------|
| GET | /api/admin/endpoints | List all endpoints with IPs |
| POST | /api/admin/endpoints | Add endpoint manually |
| DELETE | /api/admin/endpoints/{id} | Delete endpoint |
| GET | /api/admin/metrics | Comprehensive system metrics |
| GET | /api/admin/settings | Get settings (outage threshold) |
| PUT | /api/admin/settings | Update settings |
| GET | /api/admin/site-config | Get site configuration |
| PUT | /api/admin/site-config | Update site configuration |

## Database Schema

### endpoints
- id, ipv4, ip_hash, isp, status, created_at, last_seen, last_ok
- monitored_hop, hop_number, use_hop (legacy, not currently used)

### events
- id, timestamp, event_type, isp, endpoint_id, message

### uptime_history
- id, timestamp, total_endpoints, endpoints_up, endpoints_down

### settings
- key, value
- Keys: `admin_password_hash`, `outage_threshold`, `site_config`

## Site Configuration
The About page content and site branding are configurable via the admin panel:
- Site name and description
- About page sections (Why, How It Works, Privacy)
- Supported ISPs list
- Contact email
- Footer text
- GitHub URL

Stored as JSON in the `settings` table under the `site_config` key.

## ICMP Permissions

For ping to work, either:
- Run as root (not recommended)
- Set capability: `sudo setcap cap_net_raw+ep ./bin/ccc-api`
- Use unprivileged mode (default, requires `net.ipv4.ping_group_range` sysctl)

To enable unprivileged ICMP permanently:
```bash
echo "net.ipv4.ping_group_range = 0 2147483647" | sudo tee /etc/sysctl.d/99-ping.conf
sudo sysctl --system
```

## Production Deployment

Deployed to Oracle Cloud (uptimekuma) with:
- Systemd service at `/etc/systemd/system/ccc.service`
- Binary at `/opt/ccc/ccc-api`
- Database at `/opt/ccc/ccc.db`
- ISP config at `/opt/ccc/isp_config.json`
- Caddy reverse proxy for TLS at `ccc.as215855.net`

Deploy command:
```bash
scp bin/ccc-api uptimekuma:~/ccc-api.new
ssh uptimekuma "sudo systemctl stop ccc && sudo mv ~/ccc-api.new /opt/ccc/ccc-api && sudo chmod +x /opt/ccc/ccc-api && sudo systemctl start ccc"
```

## Pages
- `/` - Main dashboard with ISP status and events
- `/about` - About page (content from site config)
- `/admin` - Admin panel (password protected)
  - Metrics tab: System stats, ISP breakdown, uptime history
  - Endpoints tab: Manage monitored endpoints
  - Settings tab: Outage detection threshold
  - Site Config tab: Customize About page and branding

## Supported ISPs
Configured via `isp_config.json` or admin panel. Default:
- Comcast
- Starry

Other ISPs can view the dashboard but cannot register for monitoring.
