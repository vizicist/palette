// Package localhttp guards HTTP servers that listen on loopback against
// requests a browser makes on someone else's behalf.
//
// Binding to 127.0.0.1 keeps other machines out, but a browser running on this
// machine can still be pointed at the port by any web page it visits: a hidden
// <form> can POST to http://127.0.0.1:3330/api, an <img> can GET a
// state-changing URL, and with DNS rebinding the attacker's hostname resolves
// to 127.0.0.1 after the page loads, which makes the page same-origin and lets
// it read the replies too. None of those requests carry a token, so the server
// has to tell them apart by what the browser attaches to them.
package localhttp

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// CheckRequest reports why req must not be served, or nil when it may be.
//
// A request passes when:
//
//   - its Host is "localhost" or an IP literal. DNS rebinding needs a hostname
//     the attacker controls, so an IP literal (any IP, which keeps a LAN bind via
//     PALETTE_HTTP_BIND working) or the one loopback name is enough.
//   - its Origin header, when present, names a host that passes the same test.
//     Browsers send Origin on cross-origin requests and on every POST.
//   - its Sec-Fetch-Site header, when present, is not "cross-site".
//   - when requireJSON is set, its Content-Type is application/json or absent.
//     A <form> can only send urlencoded, multipart or text/plain, and a
//     cross-origin fetch cannot drop the header without a CORS preflight, which
//     this server never answers. Absent is allowed because the Python helper
//     sends its JSON body without one.
func CheckRequest(req *http.Request, requireJSON bool) error {
	if err := checkHost("Host", req.Host); err != nil {
		return err
	}
	if origin := req.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return fmt.Errorf("request refused: unparseable Origin %q", origin)
		}
		if err := checkHost("Origin", u.Host); err != nil {
			return err
		}
	}
	if site := req.Header.Get("Sec-Fetch-Site"); strings.EqualFold(site, "cross-site") {
		return fmt.Errorf("request refused: cross-site request")
	}
	if requireJSON {
		if ct := req.Header.Get("Content-Type"); ct != "" {
			mediaType, _, err := mime.ParseMediaType(ct)
			if err != nil || mediaType != "application/json" {
				return fmt.Errorf("request refused: Content-Type must be application/json, got %q", ct)
			}
		}
	}
	return nil
}

// checkHost accepts "localhost" or an IP literal, with or without a port.
func checkHost(header string, hostport string) error {
	if hostport == "" {
		return fmt.Errorf("request refused: no %s", header)
	}
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil {
		return nil
	}
	return fmt.Errorf("request refused: %s %q is not localhost or an IP address", header, hostport)
}

// Middleware wraps next so that a request CheckRequest refuses gets a 403 with
// the reason in the body, and never reaches next.
func Middleware(requireJSON bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := CheckRequest(req, requireJSON); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}
