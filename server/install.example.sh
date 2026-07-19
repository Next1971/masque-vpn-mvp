#!/bin/bash
set -e

UNIT_SRC="$(dirname "$0")/systemd/masque.service.example"
UNIT_DST="/etc/systemd/system/masque.service"

if [ ! -f /opt/masque/vpn-server ]; then
  echo "ERROR: /opt/masque/vpn-server not found. Build or place the binary first." >&2
  exit 1
fi

if [ ! -f /opt/masque/config.server.toml ]; then
  echo "ERROR: /opt/masque/config.server.toml not found." >&2
  exit 1
fi

echo "Installing systemd unit -> $UNIT_DST"
cp "$UNIT_SRC" "$UNIT_DST"
systemctl daemon-reload
systemctl enable masque.service
systemctl restart masque.service
sleep 2
systemctl --no-pager status masque.service | head -20
echo
echo "Done. Logs: journalctl -u masque -f"