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

// SecurityHeadersMiddleware adds defensive HTTP response headers, including a
// strict Content Security Policy with script-src 'self' (no inline scripts or
// inline event handlers). Pages served behind this middleware must load all
// JavaScript from external files and attach handlers via addEventListener.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy.
		//
		// script-src 'self' deliberately omits 'unsafe-inline': every server
		// using this middleware serves only external JS files with event
		// handlers attached via addEventListener / data-action delegation.
		// State handed from Go templates to JS must use non-executable
		// <script type="application/json"> data blocks.
		//
		// frame-ancestors / base-uri / form-action / object-src prevent
		// clickjacking, base-tag injection, form-action redirection and
		// legacy plugin embeds even on browsers that ignore the older
		// X-Frame-Options header.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' data: https://fonts.gstatic.com; "+
				"connect-src 'self' https://api.us.nylas.com https://api.eu.nylas.com; "+
				"frame-ancestors 'self'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"object-src 'none';")

		next.ServeHTTP(w, r)
	})
}

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
