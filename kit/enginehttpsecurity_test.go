package kit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineAPIRequestSecurity(t *testing.T) {
	cases := []struct {
		name, host, method, origin, site, contentType string
		want                                          int
	}{
		{"web UI", "127.0.0.1:3330", "POST", "http://127.0.0.1:3330", "same-origin", "application/json", 200},
		{"localhost UI", "localhost:3330", "POST", "http://localhost:3330", "same-origin", "application/json; charset=utf-8", 200},
		{"CLI", "127.0.0.1:3330", "POST", "", "", "application/json", 200},
		{"Python helper", "127.0.0.1:3330", "POST", "", "", "", 200},
		{"IPv6", "[::1]:3330", "POST", "http://[::1]:3330", "same-origin", "application/json", 200},
		{"LAN UI", "192.168.1.50:3330", "POST", "http://192.168.1.50:3330", "same-origin", "application/json", 200},
		{"original exploit", "127.0.0.1:3330", "POST", "http://attacker.example", "cross-site", "text/plain", 403},
		{"foreign origin JSON", "127.0.0.1:3330", "POST", "http://attacker.example", "", "application/json", 403},
		{"foreign IP origin", "127.0.0.1:3330", "POST", "http://192.168.1.2:3330", "", "application/json", 403},
		{"other local port", "127.0.0.1:3330", "POST", "http://127.0.0.1:8000", "", "application/json", 403},
		{"other scheme", "127.0.0.1:3330", "POST", "https://127.0.0.1:3330", "", "application/json", 403},
		{"null origin", "127.0.0.1:3330", "POST", "null", "", "application/json", 403},
		{"malformed origin", "127.0.0.1:3330", "POST", "%", "", "application/json", 403},
		{"origin with path", "127.0.0.1:3330", "POST", "http://127.0.0.1:3330/evil", "", "application/json", 403},
		{"rebinding", "attacker.example:3330", "POST", "http://attacker.example:3330", "same-origin", "application/json", 403},
		{"rebinding no origin", "attacker.example:3330", "POST", "", "", "application/json", 403},
		{"cross-site metadata", "127.0.0.1:3330", "POST", "", "cross-site", "application/json", 403},
		{"same-site metadata", "127.0.0.1:3330", "POST", "", "same-site", "application/json", 403},
		{"text form", "127.0.0.1:3330", "POST", "", "", "text/plain", 415},
		{"URL form", "127.0.0.1:3330", "POST", "", "", "application/x-www-form-urlencoded", 415},
		{"multipart form", "127.0.0.1:3330", "POST", "", "", "multipart/form-data; boundary=x", 415},
		{"missing browser content type", "127.0.0.1:3330", "POST", "http://127.0.0.1:3330", "", "", 415},
		{"GET", "127.0.0.1:3330", "GET", "", "", "", 405},
		{"preflight", "127.0.0.1:3330", "OPTIONS", "http://attacker.example", "cross-site", "", 403},
	}
	for _, path := range []string{"/api", "/nats/api"} {
		for _, tc := range cases {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				called := false
				mux := http.NewServeMux()
				mux.HandleFunc(path, engineAPIHandler(func(r *http.Request) (string, error) { called = true; return `{}`, nil }))
				req := httptest.NewRequest(tc.method, "http://"+tc.host+path, strings.NewReader(`{"api":"global.done"}`))
				for name, value := range map[string]string{"Origin": tc.origin, "Sec-Fetch-Site": tc.site, "Content-Type": tc.contentType} {
					if value != "" {
						req.Header.Set(name, value)
					}
				}
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, req)
				if rec.Code != tc.want || called != (tc.want == 200) {
					t.Fatalf("status=%d dispatched=%v; want status=%d dispatched=%v", rec.Code, called, tc.want, tc.want == 200)
				}
			})
		}
	}
}

func TestEngineAPIRejectsMultipleOrigins(t *testing.T) {
	req := httptest.NewRequest("POST", "http://127.0.0.1:3330/api", nil)
	req.Header.Add("Origin", "http://127.0.0.1:3330")
	req.Header.Add("Origin", "http://attacker.example")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engineAPIHandler(func(*http.Request) (string, error) {
		t.Fatal("dispatched request with multiple origins")
		return "", nil
	})(rec, req)
	if rec.Code != 403 {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}
