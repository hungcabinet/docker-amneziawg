package webui

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
	"github.com/hungcabinet/docker-amneziawg/web/internal/status"
	"github.com/hungcabinet/docker-amneziawg/web/internal/store"
)

func testApp(t *testing.T, source string) (*Server, config.Config) {
	t.Helper()
	cfg := config.Config{
		Source:          source,
		DirectInterface: "eth0",
		PingTargets:     []string{"1.1.1.1", "8.8.8.8"},
		PingTargetsByIF: map[string][]string{},
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Bootstrap(context.Background(), "admin", "secret"); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	provider, err := status.New(cfg)
	if err != nil {
		t.Fatalf("status.New() error = %v", err)
	}
	app, err := New(cfg, st, provider)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return app, cfg
}

func TestHandler_LoginAndPeers(t *testing.T) {
	app, _ := testApp(t, status.SourceMock)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler())
	t.Cleanup(srv.Close)
	client := &http.Client{Jar: jar}

	res, err := client.Get(srv.URL + "/peers")
	if err != nil {
		t.Fatalf("GET /peers error = %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "Sign in") {
		t.Fatalf("unauthenticated /peers = %d, body %s", res.StatusCode, body)
	}

	res, err = client.PostForm(srv.URL+"/login", url.Values{
		"username": {"admin"},
		"password": {"secret"},
	})
	if err != nil {
		t.Fatalf("POST /login error = %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", res.StatusCode)
	}
	page := string(body)
	if !strings.Contains(page, "bunny") || !strings.Contains(page, "online") {
		t.Fatalf("peers page missing mock data: %s", page)
	}
}

func TestHandler_LiveSourceMessage(t *testing.T) {
	app, _ := testApp(t, status.SourceLive)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler())
	t.Cleanup(srv.Close)
	client := &http.Client{Jar: jar}

	res, err := client.PostForm(srv.URL+"/login", url.Values{
		"username": {"admin"},
		"password": {"secret"},
	})
	if err != nil {
		t.Fatalf("POST /login error = %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(body), "could not load status") &&
		!strings.Contains(string(body), "server is down") {
		t.Fatalf("live page = %s", body)
	}
}
