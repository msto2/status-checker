#!/bin/bash

# Service Monitor Installation Script for Proxmox LXC Container
# This script installs and configures the service monitoring system

set -e

echo "========================================="
echo "Service Monitor Installation"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

# Update system
echo "Updating system packages..."
apt update && apt upgrade -y

# Install required packages
echo "Installing required packages..."
apt install -y \
  wget \
  curl \
  git \
  iputils-ping \
  sqlite3 \
  ufw \
  build-essential

# Install Go
echo "Installing Go..."
GO_VERSION="1.23.6"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"

cd /tmp
if [ ! -f "$GO_TARBALL" ]; then
  wget "https://go.dev/dl/${GO_TARBALL}"
fi

# Remove old Go installation if exists
rm -rf /usr/local/go

# Extract Go
tar -C /usr/local -xzf "$GO_TARBALL"

# Add Go to PATH
if ! grep -q "/usr/local/go/bin" /etc/profile; then
  echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
fi

export PATH=$PATH:/usr/local/go/bin

# Verify Go installation
echo "Verifying Go installation..."
go version

# Create application directory
echo "Creating application directory..."
APP_DIR="/opt/service-monitor"
mkdir -p "$APP_DIR"
mkdir -p /var/log/service-monitor

# Determine source directory for application files
SCRIPT_PATH="$(readlink -f "$0")"
SCRIPT_DIR="$(dirname "$SCRIPT_PATH")"
SOURCE_DIR=""

# Check if go.mod exists in parent directory (running from scripts/)
if [ -f "$SCRIPT_DIR/../go.mod" ]; then
  SOURCE_DIR="$(readlink -f "$SCRIPT_DIR/..")"
  echo "Found source files in: $SOURCE_DIR"
# Check if go.mod exists in current directory
elif [ -f "./go.mod" ]; then
  SOURCE_DIR="$(pwd)"
  echo "Found source files in current directory: $SOURCE_DIR"
# Check if go.mod exists two levels up (running from nested location)
elif [ -f "../../go.mod" ]; then
  SOURCE_DIR="$(readlink -f ../..)"
  echo "Found source files in: $SOURCE_DIR"
else
  echo "Error: Cannot find go.mod file!"
  echo "Please run this script from:"
  echo "  - The repository root directory: ./scripts/install.sh"
  echo "  - The scripts directory: ./install.sh"
  echo "Current directory: $(pwd)"
  echo "Script location: $SCRIPT_DIR"
  exit 1
fi

# Copy application files
echo "Copying application files from $SOURCE_DIR to $APP_DIR..."
cp -r "$SOURCE_DIR"/* "$APP_DIR/"
cd "$APP_DIR"

# Build the application
echo "Building Service Monitor..."
cd "$APP_DIR"
go mod download
go build -ldflags="-s -w" -o service-monitor .

# Verify binary
if [ ! -f "$APP_DIR/service-monitor" ]; then
  echo "Error: Failed to build service-monitor binary"
  exit 1
fi

echo "Binary built successfully"

# Generate API key if not already set in config
echo "Checking API key in config.json..."
if grep -q "CHANGE_THIS_TO_64_CHAR_KEY" "$APP_DIR/config.json"; then
  echo "Generating secure API key..."
  API_KEY=$(openssl rand -hex 32)
  sed -i "s/CHANGE_THIS_TO_64_CHAR_KEY/$API_KEY/g" "$APP_DIR/config.json"
  echo ""
  echo "========================================="
  echo "IMPORTANT: Your API Key"
  echo "========================================="
  echo "$API_KEY"
  echo ""
  echo "Save this key! You'll need it for:"
  echo "1. Cloudflare Worker configuration"
  echo "2. Backend cannot push updates without it"
  echo "========================================="
  echo ""
  read -p "Press Enter to continue..."
fi

# Create systemd service
echo "Creating systemd service..."
cat > /etc/systemd/system/service-monitor.service <<EOF
[Unit]
Description=Service Monitor
Documentation=https://github.com/msto2/serverStatus
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/service-monitor
Restart=always
RestartSec=10
StandardOutput=append:/var/log/service-monitor/app.log
StandardError=append:/var/log/service-monitor/app.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
systemctl daemon-reload

# Configure firewall
echo "Configuring firewall..."

# Enable UFW if not already enabled
if ! ufw status | grep -q "Status: active"; then
  echo "y" | ufw enable
fi

# Allow SSH
ufw allow 22/tcp comment 'SSH'

# Allow admin interface from local network only
echo "Enter your local network CIDR (e.g., 192.168.1.0/24):"
read -p "Local network CIDR: " LOCAL_NETWORK

if [ -n "$LOCAL_NETWORK" ]; then
  ufw allow from "$LOCAL_NETWORK" to any port 8080 comment 'Service Monitor Admin'
  echo "Admin interface will be accessible only from $LOCAL_NETWORK"
else
  echo "Warning: No network specified. Admin interface might not be accessible."
fi

# Ensure outbound HTTPS is allowed (for pushing to Cloudflare)
ufw allow out 443/tcp comment 'HTTPS out'

# Reload firewall
ufw reload

# Setup log rotation
echo "Setting up log rotation..."
cat > /etc/logrotate.d/service-monitor <<EOF
/var/log/service-monitor/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 root root
    sharedscripts
    postrotate
        systemctl reload service-monitor > /dev/null 2>&1 || true
    endscript
}
EOF

# Create backup script
echo "Creating backup script..."
mkdir -p /opt/backups
cat > /opt/backups/backup-service-monitor.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/opt/backups/service-monitor"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup database
cp /opt/service-monitor/monitor.db "$BACKUP_DIR/monitor_${DATE}.db"

# Backup config
cp /opt/service-monitor/config.json "$BACKUP_DIR/config_${DATE}.json"

# Keep only last 30 days
find "$BACKUP_DIR" -name "monitor_*.db" -mtime +30 -delete
find "$BACKUP_DIR" -name "config_*.json" -mtime +30 -delete

echo "Backup completed: $DATE"
EOF

chmod +x /opt/backups/backup-service-monitor.sh

# Add backup to crontab (daily at 2 AM)
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/backups/backup-service-monitor.sh") | crontab -

# Get local IP
LOCAL_IP=$(hostname -I | awk '{print $1}')

echo ""
echo "========================================="
echo "Installation Complete!"
echo "========================================="
echo ""
echo "Service Monitor has been installed successfully."
echo ""
echo "Next steps:"
echo ""
echo "1. Start the service:"
echo "   systemctl start service-monitor"
echo ""
echo "2. Enable auto-start on boot:"
echo "   systemctl enable service-monitor"
echo ""
echo "3. Check service status:"
echo "   systemctl status service-monitor"
echo ""
echo "4. Access admin interface:"
echo "   http://$LOCAL_IP:8080/admin.html"
echo ""
echo "5. View logs:"
echo "   tail -f /var/log/service-monitor/app.log"
echo ""
echo "6. Configure Cloudflare Pages:"
echo "   - Deploy frontend/ directory to Cloudflare Pages"
echo "   - Set domain: status.triplepoint.me"
echo "   - Add KV binding: STATUS_STORE"
echo "   - Set environment variable API_KEY (shown above)"
echo ""
echo "Configuration file: $APP_DIR/config.json"
echo "Database location: $APP_DIR/monitor.db"
echo ""
echo "========================================="
echo ""

# Prompt to start service
read -p "Would you like to start the service now? (y/n): " START_NOW

if [ "$START_NOW" = "y" ] || [ "$START_NOW" = "Y" ]; then
  systemctl enable service-monitor
  systemctl start service-monitor
  sleep 2
  systemctl status service-monitor
  echo ""
  echo "Service is now running!"
  echo "Access admin interface at: http://$LOCAL_IP:8080/admin.html"
fi

echo ""
echo "Installation script completed successfully."
