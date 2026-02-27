# Service Monitor

A lightweight, efficient service monitoring system that pings services on your local network and displays their status on an internet-accessible dashboard.

## Features

- **ICMP Ping Monitoring**: Monitors services on local network via ping every 5 minutes
- **Internet-Accessible Dashboard**: Public status page hosted on Cloudflare Pages
- **Backend Offline Detection**: Frontend automatically detects when backend hasn't reported in 10+ minutes
- **Admin Interface**: Modern web UI (local network only) to manage services and alerts
- **Email Alerts**: Sends email notifications when services go down
- **Historical Data**: Stores status history in SQLite database
- **Lightweight**: Uses ~10-20MB RAM, minimal CPU usage
- **Secure**: LAN-only admin access, encrypted HTTPS communication

## Architecture

```
Internet: status.triplepoint.me (Cloudflare Pages)
    ├── Public status dashboard
    └── API endpoints (Cloudflare Functions + KV)
          ↑
          | HTTPS POST (every 5 min)
          |
Local Network: Proxmox LXC Container
    ├── Go Backend
    │   ├── Ping services every 5 minutes
    │   ├── Push status to Cloudflare
    │   └── Send email alerts
    └── Admin Interface (port 8080, LAN only)
```

## Technology Stack

- **Backend**: Go 1.23+ (single binary, minimal resources)
- **Frontend**: Vanilla HTML/CSS/JavaScript (no build tools)
- **Database**: SQLite (embedded, single file)
- **Hosting**: Cloudflare Pages + Functions
- **Deployment**: Proxmox LXC container (Debian/Ubuntu)

## Quick Start

### Prerequisites

- Proxmox LXC container (Debian 12 or Ubuntu 22.04)
- Cloudflare account
- Domain configured in Cloudflare (e.g., status.triplepoint.me)
- SMTP server for email alerts (optional)

### 1. Deploy Backend (Proxmox Container)

```bash
# SSH into your Proxmox container
ssh root@your-container-ip

# Clone or copy this repository
cd /tmp
git clone <your-repo-url>
cd service-monitor

# Run installation script
chmod +x scripts/install.sh
./scripts/install.sh

# The script will:
# - Install Go and dependencies
# - Build the application
# - Generate API key
# - Configure firewall
# - Setup systemd service
```

**Save the API key shown during installation!** You'll need it for Cloudflare.

### 2. Deploy Frontend (Cloudflare Pages)

```bash
# Install Wrangler CLI
npm install -g wrangler

# Login to Cloudflare
wrangler login

# Create KV namespace
wrangler kv:namespace create STATUS_STORE --preview=false
# Save the namespace ID

# Deploy frontend
cd frontend
wrangler pages deploy . --project-name=service-monitor
```

**Configure in Cloudflare Dashboard:**

1. Go to Workers & Pages → service-monitor → Settings
2. Add KV namespace binding:
   - Variable name: `STATUS_STORE`
   - KV namespace: (select the one you created)
3. Add environment variable:
   - Name: `API_KEY`
   - Value: (the key generated during backend installation)
4. Add custom domain:
   - Go to Custom domains
   - Add: `status.triplepoint.me`

### 3. Configure Backend

Update backend config with your domain:

```bash
cd /opt/service-monitor
nano config.json
```

Update these values:
```json
{
  "frontend_url": "https://status.triplepoint.me/api/update",
  "api_key": "YOUR_API_KEY_FROM_INSTALLATION"
}
```

Restart the service:
```bash
systemctl restart service-monitor
```

### 4. Access Admin Interface

Open browser and navigate to:
```
http://YOUR_CONTAINER_IP:8080/admin.html
```

## Usage

### Adding Services to Monitor

1. Open admin interface (http://container-ip:8080/admin.html)
2. Click "Add Service"
3. Enter:
   - Service Name (e.g., "Router")
   - IP Address (e.g., "192.168.1.1")
   - Description (optional)
4. Click "Save"

The service will be checked within 5 minutes.

### Configuring Email Alerts

1. Open admin interface → "Email Alerts" tab
2. Click "Add Recipient" and enter email address
3. Go to "Configuration" tab
4. Enter SMTP settings:
   - SMTP Host: `smtp.gmail.com` (or your provider)
   - SMTP Port: `587` (TLS) or `465` (SSL)
   - Username: Your email address
   - Password: App-specific password
   - From: `Service Monitor <alerts@yourdomain.com>`
5. Click "Save Configuration"
6. Test by clicking "Test Email"

**Gmail Users**: Use [App Passwords](https://support.google.com/accounts/answer/185833)

### Viewing Status

Public dashboard: **https://status.triplepoint.me**

The page shows:
- Total services
- Services up/down count
- Real-time status for each service
- Response times
- Last update timestamp
- Backend offline warning (if no updates for 10+ minutes)

Auto-refreshes every 10 seconds.

## Project Structure

```
service-monitor/
├── main.go                          # Entry point
├── config.json                      # Configuration
├── go.mod                           # Go dependencies
├── internal/
│   ├── models/models.go            # Data structures
│   ├── database/db.go              # SQLite operations
│   ├── monitor/
│   │   ├── pinger.go               # ICMP ping logic
│   │   └── scheduler.go            # 5-minute scheduling
│   ├── pusher/pusher.go            # Push to Cloudflare
│   ├── alerts/alerter.go           # Email notifications
│   └── api/
│       ├── admin.go                # API endpoints
│       └── middleware.go           # Security middleware
├── web/
│   └── admin.html                  # Admin interface
├── frontend/
│   ├── index.html                  # Public status page
│   ├── style.css                   # Styling
│   ├── app.js                      # Frontend logic
│   ├── functions/api/
│   │   ├── update.js               # Receive updates
│   │   └── status.js               # Serve status
│   └── README.md                   # Frontend deployment guide
└── scripts/
    ├── install.sh                  # Installation script
    ├── uninstall.sh                # Uninstallation script
    └── service-monitor.service     # Systemd service file
```

## Configuration

### Backend Configuration (`config.json`)

```json
{
  "admin_port": 8080,                  // Admin interface port
  "admin_bind": "0.0.0.0",             // Bind address
  "database_path": "./monitor.db",     // SQLite database file
  "log_file": "./app.log",             // Log file location
  "log_level": "info",                 // Log level
  "frontend_url": "https://status.triplepoint.me/api/update",
  "api_key": "YOUR_64_CHAR_KEY",       // Must match Cloudflare
  "check_interval_seconds": 300,       // Check every 5 minutes
  "ping_timeout_seconds": 5,           // Ping timeout
  "ping_count": 4,                     // Number of pings per check
  "max_concurrent_pings": 10,          // Parallel ping limit
  "push_timeout_seconds": 10,          // HTTP timeout
  "push_retry_attempts": 3,            // Retry on failure
  "alert_cooldown_minutes": 60         // Min time between alerts
}
```

### Database Schema

The system uses SQLite with these tables:

- **services**: Monitored services (name, IP, description, enabled)
- **status_history**: Historical ping results (status, response time, timestamp)
- **alert_recipients**: Email addresses for alerts
- **alert_log**: Log of sent/failed alert emails
- **config**: SMTP and other configuration

## Management

### Systemd Service

```bash
# Start service
systemctl start service-monitor

# Stop service
systemctl stop service-monitor

# Restart service
systemctl restart service-monitor

# Check status
systemctl status service-monitor

# Enable auto-start on boot
systemctl enable service-monitor

# View logs
journalctl -u service-monitor -f
```

### Log Files

```bash
# Application logs
tail -f /var/log/service-monitor/app.log

# Systemd logs
journalctl -u service-monitor -f

# Log location: /var/log/service-monitor/
```

### Backup

Automatic daily backups are configured at 2 AM:

```bash
# Backup location
/opt/backups/service-monitor/

# Manual backup
/opt/backups/backup-service-monitor.sh

# Backups older than 30 days are automatically deleted
```

### Database Operations

```bash
# Connect to database
sqlite3 /opt/service-monitor/monitor.db

# View all services
SELECT * FROM services;

# View recent status checks
SELECT * FROM status_history ORDER BY checked_at DESC LIMIT 20;

# View alert recipients
SELECT * FROM alert_recipients;

# View recent alerts
SELECT * FROM alert_log ORDER BY sent_at DESC LIMIT 10;
```

## API Endpoints

### Admin API (Local Network Only)

- `GET /health` - Health check
- `GET /api/services` - List all services
- `POST /api/services` - Create service
- `PUT /api/services/:id` - Update service
- `DELETE /api/services/:id` - Delete service
- `POST /api/services/:id/check` - Check service now
- `GET /api/services/:id/history` - Get service history
- `GET /api/alerts` - List alert recipients
- `POST /api/alerts` - Add alert recipient
- `DELETE /api/alerts/:id` - Remove alert recipient
- `POST /api/alerts/test` - Send test email
- `GET /api/config` - Get configuration
- `PUT /api/config` - Update configuration

### Public API (Cloudflare)

- `GET /api/status` - Get current service status (public)
- `POST /api/update` - Update status (backend only, requires API key)

## Security

### LAN-Only Admin Access

The admin interface is protected by:

1. **Middleware**: Checks client IP against private network ranges
2. **Firewall (UFW)**: Blocks external access to port 8080
3. **No Internet Exposure**: Backend never accepts inbound connections

Allowed IP ranges:
- `127.0.0.0/8` - Loopback
- `10.0.0.0/8` - Private Class A
- `172.16.0.0/12` - Private Class B
- `192.168.0.0/16` - Private Class C

### API Key Authentication

- 64-character random key
- Required for backend→frontend communication
- Stored in backend config and Cloudflare environment variables
- Generate new key: `openssl rand -hex 32`

### HTTPS Communication

- All backend→frontend communication over HTTPS
- Cloudflare provides automatic SSL/TLS
- No sensitive data exposed

## Troubleshooting

### Backend Not Starting

```bash
# Check logs
journalctl -u service-monitor -n 50

# Check if port is in use
netstat -tulpn | grep 8080

# Verify binary exists
ls -l /opt/service-monitor/service-monitor

# Check config file
cat /opt/service-monitor/config.json
```

### Frontend Not Updating

```bash
# Check backend logs for push errors
tail -f /var/log/service-monitor/app.log | grep -i push

# Verify API key matches
# Backend: /opt/service-monitor/config.json
# Cloudflare: Dashboard → Environment Variables

# Test update endpoint manually
curl -X POST https://status.triplepoint.me/api/update \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"services":[],"last_update_time":"2024-01-01T00:00:00Z"}'
```

### Email Alerts Not Working

1. Check SMTP configuration in admin UI
2. Test email functionality (Configuration tab → Test Email)
3. Check alert logs in admin UI (Alert Logs tab)
4. Verify SMTP credentials
5. For Gmail: Ensure App Password is used, not regular password

### Admin Interface Not Accessible

```bash
# Check if service is running
systemctl status service-monitor

# Check firewall rules
ufw status

# Verify listening on port
netstat -tulpn | grep 8080

# Check from another machine on LAN
curl http://CONTAINER_IP:8080/health
```

### High CPU Usage

```bash
# Check active checks
ps aux | grep service-monitor

# Reduce check frequency in config.json
# "check_interval_seconds": 300 → 600 (10 minutes)

# Reduce concurrent pings
# "max_concurrent_pings": 10 → 5
```

## Performance

- **RAM Usage**: 10-20MB typical, 50MB maximum
- **CPU Usage**: <1% idle, 5-10% during checks
- **Disk Usage**: ~10MB application + database (grows ~1KB/service/day)
- **Network**: Minimal (ping packets + one HTTPS POST per 5 min)
- **Scalability**: Handles 100+ services easily

## Uninstallation

```bash
cd /opt/service-monitor/scripts
./uninstall.sh

# This will:
# - Stop and disable the service
# - Remove application files
# - Remove logs
# - Optionally remove backups
# - Optionally remove firewall rules
```

## Development

### Local Development

```bash
# Clone repository
git clone <repo-url>
cd service-monitor

# Install dependencies
go mod download

# Build
go build -o service-monitor .

# Run locally
./service-monitor

# Access admin interface
open http://localhost:8080/admin.html
```

### Testing

```bash
# Run tests (if implemented)
go test ./...

# Test ping functionality
go run main.go # and check logs

# Test API endpoints
curl http://localhost:8080/api/services
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

[Your License Here]

## Support

For issues, questions, or feature requests:
- Open an issue on GitHub
- Check the troubleshooting section
- Review logs for error messages

## Roadmap

Future enhancements:
- [ ] Uptime percentage calculations
- [ ] Historical graphs/charts
- [ ] Multiple check types (HTTP, TCP, custom)
- [ ] Discord/Slack webhook notifications
- [ ] Mobile app notifications
- [ ] Multi-region monitoring
- [ ] Custom status page themes
- [ ] REST API for external integrations

## Acknowledgments

Built with:
- Go - https://go.dev
- SQLite - https://www.sqlite.org
- Cloudflare Pages - https://pages.cloudflare.com
- Gorilla Mux - https://github.com/gorilla/mux
