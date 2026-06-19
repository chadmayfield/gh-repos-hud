package web

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
)

// demoServer builds a Server backed by the synthetic demo snapshot. Demo mode
// short-circuits FetchState before it touches the (nil) client receiver, so no
// live gh client is required.
func demoServer(t *testing.T, interval time.Duration) *Server {
	t.Helper()
	s, err := newServer(nil, ghclient.Options{Demo: true}, interval)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	return s
}

func TestRefreshPopulatesState(t *testing.T) {
	s := demoServer(t, time.Second)
	if s.state.Load() != nil {
		t.Fatalf("state should be nil before refresh")
	}
	s.refresh(context.Background())
	st := s.state.Load()
	if st == nil {
		t.Fatal("state is nil after refresh")
	}
	if len(st.Owners) == 0 {
		t.Fatal("refresh produced no owners")
	}
}

func TestRefreshFreshPopulatesState(t *testing.T) {
	s := demoServer(t, time.Second)
	s.refreshFresh(context.Background())
	st := s.state.Load()
	if st == nil {
		t.Fatal("state is nil after refreshFresh")
	}
	if len(st.Owners) == 0 {
		t.Fatal("refreshFresh produced no owners")
	}
}

func TestPollPopulatesAndStops(t *testing.T) {
	s := demoServer(t, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.poll(ctx)
		close(done)
	}()

	// Wait for at least one tick to populate state.
	deadline := time.After(2 * time.Second)
	for s.state.Load() == nil {
		select {
		case <-deadline:
			cancel()
			t.Fatal("poll never populated state")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("poll did not return after cancel")
	}

	if st := s.state.Load(); st == nil || len(st.Owners) == 0 {
		t.Fatal("poll left state empty")
	}
}

func TestHandleIndexRefreshQuery(t *testing.T) {
	s := demoServer(t, time.Second)
	rr := httptest.NewRecorder()
	s.handleIndex(rr, httptest.NewRequest("GET", "/?refresh=1", nil))
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	// ?refresh=1 triggers refreshFresh, which loads the demo snapshot.
	for _, want := range []string{"gh-repos-hud", "acme-corp", "payments-api"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestHandleIndexNotFound(t *testing.T) {
	s := demoServer(t, time.Second)
	rr := httptest.NewRecorder()
	s.handleIndex(rr, httptest.NewRequest("GET", "/nope", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestHandleStateNil(t *testing.T) {
	s := demoServer(t, time.Second) // no state stored
	rr := httptest.NewRecorder()
	s.handleState(rr, httptest.NewRequest("GET", "/api/state.json", nil))
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	if body := strings.TrimSpace(rr.Body.String()); body != "{}" {
		t.Errorf("nil state body = %q, want {}", body)
	}
}

// freePort binds an ephemeral port, then releases it so Serve can claim it.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func TestServeEndToEnd(t *testing.T) {
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- Serve(ctx, nil, ghclient.Options{Demo: true}, port, time.Second, false)
	}()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: time.Second}

	// Wait for the server to come up via /healthz.
	healthy := false
	deadline := time.After(3 * time.Second)
	for !healthy {
		select {
		case <-deadline:
			cancel()
			t.Fatal("server never became healthy")
		default:
		}
		resp, err := client.Get(base + "/healthz")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				healthy = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("index", func(t *testing.T) {
		resp, err := client.Get(base + "/")
		if err != nil {
			t.Fatalf("GET /: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("status = %d", resp.StatusCode)
		}
		b, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(b), "gh-repos-hud") {
			t.Errorf("index missing title")
		}
	})

	t.Run("state.json", func(t *testing.T) {
		resp, err := client.Get(base + "/api/state.json")
		if err != nil {
			t.Fatalf("GET state: %v", err)
		}
		defer resp.Body.Close()
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
		b, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(b), "payments-api") {
			t.Errorf("state json missing demo repo")
		}
	})

	t.Run("asset", func(t *testing.T) {
		resp, err := client.Get(base + "/assets/app.css")
		if err != nil {
			t.Fatalf("GET asset: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("asset status = %d", resp.StatusCode)
		}
	})

	cancel()
	select {
	case err := <-srvErr:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
}
