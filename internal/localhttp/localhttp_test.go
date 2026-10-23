package localhttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func request(t *testing.T, host string, headers map[string]string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://"+host+"/api", strings.NewReader(`{"api":"global.status"}`))
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestAcceptsLegitimateCallers(t *testing.T) {
	cases := map[string]*http.Request{
		"web UI":          request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json", "Origin": "http://127.0.0.1:3330", "Sec-Fetch-Site": "same-origin"}),
		"localhost":       request(t, "localhost:3330", map[string]string{"Content-Type": "application/json"}),
		"ipv6 loopback":   request(t, "[::1]:3330", map[string]string{"Content-Type": "application/json"}),
		"no port":         request(t, "127.0.0.1", map[string]string{"Content-Type": "application/json"}),
		"LAN bind":        request(t, "192.168.1.50:3330", map[string]string{"Content-Type": "application/json"}),
		"charset param":   request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json; charset=utf-8"}),
		"python helper":   request(t, "127.0.0.1:3330", nil), // requests.post(data=str) sends no Content-Type
		"same-site fetch": request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json", "Sec-Fetch-Site": "none"}),
	}
	for name, req := range cases {
		if err := CheckRequest(req, true); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}

func TestRefusesBrowserMediatedRequests(t *testing.T) {
	cases := map[string]*http.Request{
		"DNS rebinding":         request(t, "attacker.example:3330", map[string]string{"Content-Type": "application/json"}),
		"bare hostname":         request(t, "palette.local", map[string]string{"Content-Type": "application/json"}),
		"empty host":            request(t, "", map[string]string{"Content-Type": "application/json"}),
		"form text/plain":       request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "text/plain"}),
		"form urlencoded":       request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}),
		"form multipart":        request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "multipart/form-data; boundary=x"}),
		"cross-origin fetch":    request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json", "Origin": "https://attacker.example"}),
		"null origin":           request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json", "Origin": "null"}),
		"sec-fetch-site":        request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json", "Sec-Fetch-Site": "cross-site"}),
		"rebinding with origin": request(t, "attacker.example:3330", map[string]string{"Content-Type": "application/json", "Origin": "http://attacker.example:3330"}),
	}
	for name, req := range cases {
		if err := CheckRequest(req, true); err == nil {
			t.Errorf("%s: accepted, want refusal", name)
		}
	}
}

// A server whose endpoints are GETs with query parameters has no Content-Type
// to check, so the other three tests still have to carry it.
func TestWithoutJSONRequirement(t *testing.T) {
	ok := request(t, "127.0.0.1:8080", map[string]string{"Content-Type": "text/plain"})
	if err := CheckRequest(ok, false); err != nil {
		t.Errorf("text/plain refused when JSON is not required: %v", err)
	}
	img := request(t, "127.0.0.1:8080", map[string]string{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Dest": "image"})
	if err := CheckRequest(img, false); err == nil {
		t.Error("cross-site <img> request accepted")
	}
	rebind := request(t, "attacker.example:8080", nil)
	if err := CheckRequest(rebind, false); err == nil {
		t.Error("rebinding host accepted")
	}
}

func TestMiddlewareReturns403(t *testing.T) {
	called := false
	h := Middleware(true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, request(t, "attacker.example:3330", map[string]string{"Content-Type": "application/json"}))
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("refused request: status %d, handler called %v", rec.Code, called)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, request(t, "127.0.0.1:3330", map[string]string{"Content-Type": "application/json"}))
	if rec.Code != http.StatusOK || !called {
		t.Fatalf("allowed request: status %d, handler called %v", rec.Code, called)
	}
}
