package admin

import (
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func AuthorizedPage(re *core.RequestEvent) bool {
	if re.Auth != nil {
		return true
	}

	if !allowLoopbackBypass() {
		return false
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

func allowLoopbackBypass() bool {
	// Production defaults to strict mode: require authenticated superuser only.
	// Non-production keeps loopback convenience by default.
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	isProd := appEnv == "prod" || appEnv == "production"

	raw := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_ALLOW_LOOPBACK_BYPASS")))
	if raw != "" {
		return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
	}

	return !isProd
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
