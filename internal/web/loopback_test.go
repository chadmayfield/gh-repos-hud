package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoopbackOnly verifies the DNS-rebinding guard: loopback Host headers pass,
// everything else is rejected with 403.
func TestLoopbackOnly(t *testing.T) {
	h := loopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		host string
		want int
	}{
		{"127.0.0.1:8787", http.StatusOK},
		{"localhost:8787", http.StatusOK},
		{"[::1]:8787", http.StatusOK},
		{"127.0.0.1", http.StatusOK},
		{"evil.com", http.StatusForbidden},
		{"evil.com:8787", http.StatusForbidden},
		{"169.254.169.254", http.StatusForbidden}, // cloud metadata endpoint
		{"attacker.example:80", http.StatusForbidden},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = c.host
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != c.want {
			t.Errorf("Host %q: got %d, want %d", c.host, rr.Code, c.want)
		}
	}
}
