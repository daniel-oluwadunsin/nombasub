package utils

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ValidateWebhookURL enforces that a tenant-supplied webhook endpoint is safe to
// call from the server: it must be HTTPS and must not resolve to a loopback,
// private, link-local (incl. cloud metadata 169.254.169.254), or unspecified
// address. This blocks SSRF into internal services. It is evaluated both when a
// tenant sets the URL and again immediately before each delivery, since DNS can
// change between the two (rebinding).
func ValidateWebhookURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("invalid webhook URL")
	}
	if u.Scheme != "https" {
		return errors.New("webhook URL must use https")
	}

	host := u.Hostname()
	if host == "" {
		return errors.New("webhook URL must include a host")
	}

	// If the host is a literal IP, check it directly; otherwise resolve it.
	if ip := net.ParseIP(host); ip != nil {
		if isDisallowedIP(ip) {
			return errors.New("webhook URL points to a disallowed address")
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return errors.New("could not resolve webhook host")
	}
	for _, ip := range ips {
		if isDisallowedIP(ip) {
			return errors.New("webhook URL resolves to a disallowed address")
		}
	}

	return nil
}

func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}
