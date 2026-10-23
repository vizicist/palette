package kit

import (
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// engineAPIHandler rejects browser requests from other origins before reading
// the body or dispatching commands. CLI clients may omit browser headers.
func engineAPIHandler(handler func(*http.Request) (string, error)) http.HandlerFunc {
	next := jsonAPIHandler(handler)
	return func(w http.ResponseWriter, r *http.Request) {
		reject := func(status int, message string) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(ErrorResponse(errors.New(message))))
		}
		// Accept literal addresses for deliberate LAN binds, but never arbitrary
		// DNS names: those let a rebinding page become same-origin with the API.
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
		if !strings.EqualFold(host, "localhost") && net.ParseIP(host) == nil {
			reject(http.StatusForbidden, "API Host must be localhost or an IP address")
			return
		}
		origins := r.Header.Values("Origin")
		if len(origins) > 0 {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			origin, err := url.Parse(origins[0])
			if len(origins) != 1 || err != nil || origin.Scheme != scheme ||
				!strings.EqualFold(origin.Host, r.Host) || origin.User != nil ||
				origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
				reject(http.StatusForbidden, "API requests must have the same origin")
				return
			}
		}
		site := r.Header.Get("Sec-Fetch-Site")
		if site != "" && site != "same-origin" && site != "none" {
			reject(http.StatusForbidden, "Cross-origin API requests are forbidden")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			reject(http.StatusMethodNotAllowed, "API requests must use POST")
			return
		}
		contentType := r.Header.Get("Content-Type")
		// The Python helper historically sends raw JSON without a Content-Type.
		// Only non-browser callers without Origin or Fetch Metadata may omit it.
		if contentType != "" || len(origins) != 0 || site != "" {
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err != nil || mediaType != "application/json" {
				reject(http.StatusUnsupportedMediaType, "API requests must use application/json")
				return
			}
		}
		next(w, r)
	}
}
