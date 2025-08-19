#!/bin/bash

set -e

INSTALL_DIR="/opt/elevation-api"
SERVICE_NAME="elevation-api"
SERVICE_USER="elevation"

echo "Deploying GSI Elevation API..."

if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (use sudo)"
    exit 1
fi

echo "Creating service user..."
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd -r -s /bin/false "$SERVICE_USER"
fi

echo "Creating installation directory..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/data"
mkdir -p "$INSTALL_DIR/config"

echo "Building application..."
make build

echo "Copying files..."
cp elevation-api "$INSTALL_DIR/"
cp -r config/* "$INSTALL_DIR/config/"
cp -r data/* "$INSTALL_DIR/data/" 2>/dev/null || true

echo "Setting permissions..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR/elevation-api"

echo "Installing systemd service..."
cp systemd/elevation-api.service /etc/systemd/system/
systemctl daemon-reload

echo "Starting service..."
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

echo "Checking service status..."
sleep 2
systemctl status "$SERVICE_NAME" --no-pager

echo "Testing API endpoint..."
sleep 2
curl -s "http://localhost:8080/health" | python3 -m json.tool

echo "Deployment complete!"
echo ""
echo "Service status: systemctl status $SERVICE_NAME"
echo "Service logs: journalctl -u $SERVICE_NAME -f"
echo "Test endpoint: curl 'http://localhost:8080/elevation?lat=35.6812&lon=139.7671'"