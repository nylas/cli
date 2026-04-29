// Package webguard provides loopback-only HTTP middleware shared by every
// local CLI web surface (Air, chat, UI). It hardens those servers against
// DNS-rebinding and cross-origin CSRF without coupling to any one server's
// port or routing.
package webguard

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// HostValidationMiddleware rejects requests whose Host header does not target
// a loopback address. This is the primary defence against DNS-rebinding
// attacks reaching a server bound to 127.0.0.1.
func HostValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsLoopbackHost(r.Host) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// OriginProtectionMiddleware rejects state-changing requests (POST/PUT/PATCH/
// DELETE) that do not originate from a same-host loopback origin. Safe-method
// requests pass through unchanged.
func OriginProtectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !RequiresSameOriginProtection(r.Method) || RequestMatchesLoopbackOrigin(r) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

// RequiresSameOriginProtection reports whether a method mutates state and
// therefore needs origin enforcement.
func RequiresSameOriginProtection(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// RequestMatchesLoopbackOrigin returns true when the request's Origin or
// Referer points to the same loopback host as the request itself.
func RequestMatchesLoopbackOrigin(r *http.Request) bool {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return IsLoopbackBrowserOrigin(origin, r.Host)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return IsLoopbackBrowserOrigin(referer, r.Host)
	}
	return false
}

// IsLoopbackBrowserOrigin returns true when rawOrigin is a parseable http://
// URL on a loopback host that matches the request host.
func IsLoopbackBrowserOrigin(rawOrigin, requestHost string) bool {
	if !IsLoopbackHost(requestHost) {
		return false
	}

	parsed, err := url.Parse(rawOrigin)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if parsed.Scheme != "http" {
		return false
	}
	if !IsLoopbackHost(parsed.Host) {
		return false
	}

	return strings.EqualFold(parsed.Host, requestHost)
}

// IsLoopbackHost returns true if requestHost (a Host header value, possibly
// with a port) resolves to "localhost" or any loopback IP.
func IsLoopbackHost(requestHost string) bool {
	host := requestHost
	if parsedHost, _, err := net.SplitHostPort(requestHost); err == nil {
		host = parsedHost
	}

	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
