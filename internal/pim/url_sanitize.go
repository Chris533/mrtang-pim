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

func sanitizeURLWithBase(value string, base string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if absolute := sanitizeAbsoluteURL(trimmed); absolute != "" {
		return absolute
	}
	base = sanitizeAbsoluteURL(base)
	if base == "" {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(ref)
	return sanitizeAbsoluteURL(resolved.String())
}
