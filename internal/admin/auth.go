package admin

import (
	"net"
	"net/netip"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func AuthorizedPage(re *core.RequestEvent) bool {
	if re.Auth != nil && re.Auth.IsSuperuser() {
		return true
	}

	host := strings.TrimSpace(re.Request.Host)
	if host != "" {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
	}

	if isLoopbackHost(host) {
		return true
	}

	remoteAddr := strings.TrimSpace(re.Request.RemoteAddr)
	if remoteAddr == "" {
		return false
	}

	remoteHost := remoteAddr
	if parsedHost, _, err := net.SplitHostPort(remoteAddr); err == nil {
		remoteHost = parsedHost
	}

	return isLoopbackHost(remoteHost)
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return false
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	return addr.IsLoopback()
}
