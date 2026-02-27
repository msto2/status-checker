package api

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// LocalNetworkOnly middleware restricts access to local network addresses
func LocalNetworkOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP address
		clientIP := getClientIP(r)

		// Check if IP is from local network
		if !isLocalNetwork(clientIP) {
			log.Printf("Access denied for IP: %s", clientIP)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Try X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Use RemoteAddr as fallback
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// isLocalNetwork checks if an IP address is from a local network
func isLocalNetwork(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Define local network ranges
	localRanges := []string{
		"127.0.0.0/8",    // Loopback
		"10.0.0.0/8",     // Private Class A
		"172.16.0.0/12",  // Private Class B
		"192.168.0.0/16", // Private Class C
		"169.254.0.0/16", // Link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local
	}

	for _, cidr := range localRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// CORS middleware adds CORS headers
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware logs all requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.RequestURI, getClientIP(r))
		next.ServeHTTP(w, r)
	})
}
