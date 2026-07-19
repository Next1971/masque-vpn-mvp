# MASQUE VPN Server — provisioning a new VPS

This document describes how to recreate the MASQUE server from scratch on a
new VPS. It does not contain any secrets.

## Requirements on the new VPS

- Ubuntu (tested on 20.04, works on newer versions), root or sudo access.
- Incoming **4433/udp** open in the firewall.
- `/dev/net/tun` present (usually available by default).
- `gcc` (for CGO builds) and Go 1.25+ (installed below).

## 1. System packages and Go

```bash
apt-get update && apt-get install -y build-essential

cd /usr/local
curl -LO https://go.dev/dl/go1.25.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.25.5.linux-amd64.tar.gz

export PATH=$PATH:/usr/local/go/bin
go version   # should print go1.25.5
```

## 2. Sources and build

```bash
mkdir -p /opt/masque/src

# deliver the source archive (e.g. masque-src.tgz from your project)
# and unpack it into /opt/masque/src

cd /opt/masque/src/vpn_server
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o /opt/masque/vpn-server .
```

> NOTE: the binary is built with CGO and dynamically linked against glibc.
> Always rebuild it on the target VPS; do not copy the binary between
> different distributions.

## 3. Certificates (mTLS)

Generate certificates directly on the VPS under `/opt/masque/cert/`:

- internal CA (RSA 4096),
- server certificate with SAN including your MASQUE domain, external IP
  and `localhost` (extendedKeyUsage = serverAuth),
- client certificates (extendedKeyUsage = clientAuth).

Recommended file permissions:

```bash
chmod 600 *.key
chmod 644 *.crt
```

**Do not export private keys from the VPS.**

## 4. Server config

Create `/opt/masque/config.server.toml` based on the example
`config.server.example.toml` in the repository. Typical fields:

```toml
listen_addr = "0.0.0.0:4433"

# VPN subnet that does NOT conflict with the provider gateway.
# Check the default route:
#   ip route get 8.8.8.8
assign_cidr = "10.8.0.0/24"

tun_name = "masque0"
server_name = "vpn.example.com"

[api_server]
listen_addr = "127.0.0.1:8080"  # keep API bound to localhost only

[metrics]
enabled = false
```

- `assign_cidr` must not overlap with the upstream gateway network.
- `server_name` should match the MASQUE domain used by clients (SNI / URI template host).
- The API server should remain on `127.0.0.1`; do not expose it publicly.

## 5. Networking and NAT

Enable IPv4 forwarding and set up NAT for the VPN subnet:

```bash
# find the external interface:
IFACE=$(ip route get 8.8.8.8 | grep -oP 'dev \K\S+')

sysctl -w net.ipv4.ip_forward=1
echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-masque.conf

iptables -t nat -C POSTROUTING -s 10.8.0.0/24 -o "$IFACE" -j MASQUERADE 2>/dev/null \
  || iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o "$IFACE" -j MASQUERADE
```

> The NAT rule can also be restored idempotently from `ExecStartPre` in the
> systemd unit, so it survives reboots without `iptables-persistent`.
> If your external interface name differs from `ens3`, adjust the unit file
> accordingly.

## 6. systemd unit (auto‑start)

Install and enable the systemd unit:

```bash
cp 03_server/systemd/masque.service /etc/systemd/system/masque.service

# If the external interface in the unit is not correct, edit the service file
# to match your IFACE value from the previous step.

systemctl daemon-reload
systemctl enable --now masque.service
systemctl status masque.service
journalctl -u masque -f
```

## 7. Basic checks

```bash
ss -ulpn | grep 4433              # server is listening on 4433/udp
ip -br addr show masque0          # e.g. 10.8.0.1/24 on the MASQUE TUN
ping -c1 8.8.8.8                  # VPS still has internet connectivity
iptables -t nat -S POSTROUTING | grep 10.8.0.0/24   # exactly one MASQUERADE rule
```

## Firewall (ufw example)

If you use `ufw`, open incoming 4433/udp:

```bash
ufw allow 4433/udp comment 'MASQUE VPN server'
```
