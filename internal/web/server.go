// Package web serves the HUD as a local, loopback-only dashboard.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// Server holds the latest snapshot and serves it.
type Server struct {
	client   *ghclient.Client
	opts     ghclient.Options
	interval time.Duration
	state    atomic.Pointer[model.State]
	tmpl     *template.Template
}

// newServer parses the template and builds a Server (no I/O). Shared by Serve
// and tests.
func newServer(client *ghclient.Client, opts ghclient.Options, interval time.Duration) (*Server, error) {
	tmpl, err := template.New("index").Funcs(tmplFuncs).Parse(indexHTML)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	return &Server{client: client, opts: opts, interval: interval, tmpl: tmpl}, nil
}

// Serve starts the dashboard on 127.0.0.1:port and blocks until ctx is done.
func Serve(ctx context.Context, client *ghclient.Client, opts ghclient.Options, port int, interval time.Duration) error {
	s, err := newServer(client, opts, interval)
	if err != nil {
		return err
	}

	// Prime once, then poll in the background.
	s.refresh(ctx)
	go s.poll(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/state.json", s.handleState)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "ok") })
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))

	// Loopback only — never expose this beyond the local machine.
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Printf("gh-repos-hud serving at http://%s  (Ctrl-C to stop)\n", addr)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) poll(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.refresh(ctx)
		}
	}
}

func (s *Server) refresh(ctx context.Context) {
	st, err := s.client.FetchState(ctx, s.opts)
	if err != nil {
		slog.Warn("web refresh failed", "err", err)
		return
	}
	s.state.Store(st)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	st := s.state.Load()
	data := struct {
		State    *model.State
		Interval int
	}{State: st, Interval: int(s.interval / time.Second)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		slog.Error("template execute", "err", err)
	}
}

func (s *Server) handleState(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if st := s.state.Load(); st != nil {
		_ = enc.Encode(st)
	} else {
		_ = enc.Encode(struct{}{})
	}
}
