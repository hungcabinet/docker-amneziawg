package webui

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeTheme(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"", themeLight},
		{"light", themeLight},
		{"DARK", themeDark},
		{" no ", themeLight},
	}
	for _, tt := range tests {
		if got := normalizeTheme(tt.in); got != tt.want {
			t.Errorf("normalizeTheme(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSameOriginPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		host string
		want string
		ok   bool
	}{
		{name: "relative", raw: "/exits?ok=1", host: "127.0.0.1:8080", want: "/exits?ok=1", ok: true},
		{name: "absolute same host", raw: "http://127.0.0.1:8080/peers", host: "127.0.0.1:8080", want: "/peers", ok: true},
		{name: "other host", raw: "https://evil.example/steal", host: "127.0.0.1:8080", ok: false},
		{name: "protocol-relative", raw: "//evil.example", host: "127.0.0.1:8080", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := sameOriginPath(tt.raw, tt.host)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("sameOriginPath(%q) = %q, %v", tt.raw, got, ok)
			}
		})
	}
}

func TestHandler_ThemeCookie(t *testing.T) {
	app, _ := testApp(t, "mock")
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler())
	t.Cleanup(srv.Close)
	client := &http.Client{Jar: jar}

	res, err := client.Get(srv.URL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	body := readBody(t, res)
	if !strings.Contains(body, `data-theme="light"`) {
		t.Fatalf("default theme missing: %s", body)
	}

	res, err = client.PostForm(srv.URL+"/theme", url.Values{
		"theme": {"dark"},
		"next":  {"/login"},
	})
	if err != nil {
		t.Fatalf("POST /theme: %v", err)
	}
	body = readBody(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("theme status = %d", res.StatusCode)
	}
	if !strings.Contains(body, `data-theme="dark"`) || !strings.Contains(body, `aria-label="Светлая тема"`) {
		t.Fatalf("dark theme not applied: %s", body)
	}

	u, _ := url.Parse(srv.URL)
	var theme string
	for _, c := range jar.Cookies(u) {
		if c.Name == themeCookie {
			theme = c.Value
		}
	}
	if theme != themeDark {
		t.Fatalf("cookie = %q, want dark", theme)
	}
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}
