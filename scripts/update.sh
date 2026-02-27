#!/bin/bash

# Service Monitor Update Script
# Run this to pull latest changes and rebuild

set -e

APP_DIR="/opt/service-monitor"

echo "========================================="
echo "Service Monitor Update"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

# Check if app directory exists and is a git repo
if [ ! -d "$APP_DIR/.git" ]; then
  echo "Error: $APP_DIR is not a git repository"
  echo "Please run the install script first, or manually set up git:"
  echo "  rm -rf $APP_DIR"
  echo "  git clone https://github.com/msto2/serverStatus.git $APP_DIR"
  exit 1
fi

cd "$APP_DIR"

# Save config before pull
echo "Backing up config.json..."
cp config.json config.json.backup 2>/dev/null || true

# Pull latest changes
echo "Pulling latest changes..."
git fetch origin
git reset --hard origin/main

# Restore config
if [ -f config.json.backup ]; then
  echo "Restoring config.json..."
  cp config.json.backup config.json
  rm config.json.backup
fi

# Rebuild
echo "Building Service Monitor..."
export PATH=$PATH:/usr/local/go/bin
go build -ldflags="-s -w" -o service-monitor .

# Restart service
echo "Restarting service..."
systemctl restart service-monitor

sleep 2

# Check status
systemctl status service-monitor --no-pager

echo ""
echo "========================================="
echo "Update complete!"
echo "========================================="
