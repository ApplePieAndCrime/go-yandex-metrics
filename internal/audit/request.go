package audit

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// NewEvent создаёт событие аудита для изменённых метрик и HTTP-запроса.
func NewEvent(metricNames []string, req *http.Request) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Metrics:   append([]string(nil), metricNames...),
		IPAddress: extractIPAddress(req),
	}
}

func extractIPAddress(req *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(req.Header.Get(header))
		if value == "" {
			continue
		}

		if header == "X-Forwarded-For" {
			parts := strings.Split(value, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}

		return value
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	if err == nil {
		return host
	}

	return strings.TrimSpace(req.RemoteAddr)
}
