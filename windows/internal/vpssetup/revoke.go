package vpssetup

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const remoteBlockedCNs = "/opt/masque/blocked_cns"

// ClientCN is the mTLS Common Name for issued number N (masque-client-N).
func ClientCN(n int) (string, error) {
	if n < 1 || n > LastAppIndex {
		return "", fmt.Errorf("certificate number must be 1–%d", LastAppIndex)
	}
	return fmt.Sprintf("masque-client-%d", n), nil
}

// RevokeClient appends masque-client-N to /opt/masque/blocked_cns.
// The VPN process only reloads that file on start — call RestartMasque after.
func RevokeClient(c *Client, n int, log Logf) error {
	cn, err := ClientCN(n)
	if err != nil {
		return err
	}
	q := shellSingleQuote(cn)
	if log != nil {
		log("Revoking %s…", cn)
	}
	blocked := shellSingleQuote(remoteBlockedCNs)
	out, err := c.Run(30*time.Second, fmt.Sprintf(`set -e
if [ ! -d /opt/masque ]; then echo NOMASQUE; exit 1; fi
install -d -m 0755 /opt/masque
touch %s
chmod 0644 %s
if grep -qxF %s %s 2>/dev/null; then
  echo ALREADY
else
  printf '%%s\n' %s >> %s
  echo REVOKED
fi
`, blocked, blocked, q, blocked, q, blocked), nil)
	if err != nil {
		if strings.Contains(out, "NOMASQUE") || strings.Contains(err.Error(), "NOMASQUE") {
			return fmt.Errorf("MASQUE is not installed on this VPS")
		}
		return err
	}
	if log != nil {
		if strings.Contains(out, "ALREADY") {
			log("%s was already in %s", cn, remoteBlockedCNs)
		} else {
			log("Appended %s to %s", cn, remoteBlockedCNs)
		}
		log("Restart poc-server (masque.service) for the denylist to take effect")
	}
	return nil
}

// RestartMasque restarts systemd unit masque.service (poc-server).
func RestartMasque(c *Client, log Logf) error {
	if log != nil {
		log("Restarting masque.service…")
	}
	out, err := c.Run(45*time.Second, `set -e
if [ ! -f /etc/systemd/system/masque.service ] && [ ! -f /lib/systemd/system/masque.service ]; then
  echo NOSVC
  exit 1
fi
systemctl restart masque.service
systemctl is-active masque.service
`, nil)
	if err != nil {
		if strings.Contains(out, "NOSVC") || strings.Contains(err.Error(), "NOSVC") {
			return fmt.Errorf("masque.service is not installed")
		}
		return fmt.Errorf("restart masque: %w (%s)", err, strings.TrimSpace(out))
	}
	st := strings.TrimSpace(out)
	if log != nil {
		log("masque.service is %s", lastNonEmptyLine(st))
	}
	if lastNonEmptyLine(st) != "active" {
		return fmt.Errorf("masque.service is %s", lastNonEmptyLine(st))
	}
	return nil
}

func lastNonEmptyLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(lines[i]); t != "" {
			return t
		}
	}
	return s
}

// ParseCertNumber is used by the installer UI.
func ParseCertNumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("enter a certificate number (e.g. 7)")
	}
	if _, err := ClientCN(n); err != nil {
		return 0, err
	}
	return n, nil
}
