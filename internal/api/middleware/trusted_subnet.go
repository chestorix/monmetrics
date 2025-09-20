package middleware

import (
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
)

func TrustedSubnetMiddleware(trustedSubnet string, logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trustedSubnet == "" {
				next.ServeHTTP(w, r)
				return
			}
			clientIP := r.Header.Get("X-Real-Ip")
			if clientIP == "" {
				logger.Warn("X-Real-Ip header is missing")
				http.Error(w, "Forbidden: X-Real-Ip header is missing", http.StatusForbidden)
				return
			}
			ip := net.ParseIP(clientIP)
			if ip == nil {
				logger.Warnf("Invalid IP addres in X-Real-Ip header: %s", clientIP)
				http.Error(w, "Forbidden: Invalid IP addres in X-Real-Ip header", http.StatusForbidden)
				return
			}
			_, subnet, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				logger.Warnf("Invalid trusted subnet CIDR: %s", trustedSubnet)
				http.Error(w, "Internal server error: Invalid trusted subnet CIDR", http.StatusInternalServerError)
				return
			}
			if !subnet.Contains(ip) {
				logger.Warnf("IP %s is not in trusted subnet %s", clientIP, trustedSubnet)
				http.Error(w, "Forbidden: IP is not in trusted subnet", http.StatusForbidden)
				return
			}
			logger.Debugf("IP %s is in trusted subnet %s", clientIP, trustedSubnet)
			next.ServeHTTP(w, r)
		})
	}
}
