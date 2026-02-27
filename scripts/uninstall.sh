#!/bin/bash

# Service Monitor Uninstallation Script
# This script removes the service monitoring system

set -e

echo "========================================="
echo "Service Monitor Uninstallation"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  exit 1
fi

read -p "Are you sure you want to uninstall Service Monitor? (y/n): " CONFIRM

if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
  echo "Uninstallation cancelled."
  exit 0
fi

echo "Stopping service..."
systemctl stop service-monitor || true

echo "Disabling service..."
systemctl disable service-monitor || true

echo "Removing systemd service..."
rm -f /etc/systemd/system/service-monitor.service
systemctl daemon-reload

echo "Removing application files..."
rm -rf /opt/service-monitor

echo "Removing logs..."
rm -rf /var/log/service-monitor

echo "Removing log rotation config..."
rm -f /etc/logrotate.d/service-monitor

echo "Removing backup script from crontab..."
crontab -l | grep -v '/opt/backups/backup-service-monitor.sh' | crontab - || true

echo "Removing backup script..."
rm -f /opt/backups/backup-service-monitor.sh

echo ""
read -p "Do you want to remove backups as well? (y/n): " REMOVE_BACKUPS

if [ "$REMOVE_BACKUPS" = "y" ] || [ "$REMOVE_BACKUPS" = "Y" ]; then
  echo "Removing backups..."
  rm -rf /opt/backups/service-monitor
fi

echo ""
read -p "Do you want to remove firewall rules? (y/n): " REMOVE_FW

if [ "$REMOVE_FW" = "y" ] || [ "$REMOVE_FW" = "Y" ]; then
  echo "Removing firewall rules..."
  ufw delete allow 8080/tcp || true
fi

echo ""
echo "========================================="
echo "Uninstallation Complete!"
echo "========================================="
echo ""
echo "Service Monitor has been removed from the system."
echo ""
echo "Note: Go language was not removed. To remove Go:"
echo "  rm -rf /usr/local/go"
echo "  # Remove 'export PATH=\$PATH:/usr/local/go/bin' from /etc/profile"
echo ""
