package pim

import (
	"net/url"
	"strings"
)

func sanitizeAbsoluteURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return ""
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return ""
	}
	return parsed.String()
}
