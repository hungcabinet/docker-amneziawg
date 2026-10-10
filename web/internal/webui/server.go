package webui

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/hungcabinet/docker-amneziawg/web/internal/auth"
	"github.com/hungcabinet/docker-amneziawg/web/internal/config"
	"github.com/hungcabinet/docker-amneziawg/web/internal/status"
	"github.com/hungcabinet/docker-amneziawg/web/internal/store"
)

const cookieName = "panel_session"

//go:embed templates/*.html
var templateFS embed.FS

// Server is the panel HTTP application.
type Server struct {
	cfg      config.Config
	store    *store.Store
	provider status.Provider
	tmpl     *template.Template
	secret   []byte
}

type ctxKey int

const usernameKey ctxKey = 1

// New wires handlers and parses templates.
func New(cfg config.Config, st *store.Store, provider status.Provider) (*Server, error) {
	secret, err := st.SessionSecret(context.Background())
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"bytes":     formatBytes,
		"handshake": formatHandshake,
		"latency":   formatLatency,
		"kind":      formatKind,
	}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Server{
		cfg:      cfg,
		store:    st,
		provider: provider,
		tmpl:     tmpl,
		secret:   secret,
	}, nil
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", s.handleLoginGet)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("POST /theme", s.handleTheme)
	mux.Handle("POST /logout", s.requireAuth(s.handleLogout))
	mux.Handle("GET /{$}", s.requireAuth(s.handleHome))
	mux.Handle("GET /peers", s.requireAuth(s.handlePeers))
	mux.Handle("GET /peers/{name}/qr", s.requireAuth(s.handlePeerQR))
	mux.Handle("GET /peers/{name}/config", s.requireAuth(s.handlePeerConfig))
	mux.Handle("GET /exits", s.requireAuth(s.handleExits))
	mux.Handle("POST /exits/select", s.requireAuth(s.handleSelectExit))
	mux.Handle("POST /exits/ip", s.requireAuth(s.handleExitIP))
	mux.Handle("GET /account", s.requireAuth(s.handleAccountGet))
	mux.Handle("POST /account", s.requireAuth(s.handleAccountPost))
	return withLogging(mux)
}

func (s *Server) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.sessionFromRequest(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), usernameKey, sess.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) sessionFromRequest(r *http.Request) (auth.Session, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return auth.Session{}, err
	}
	return auth.Decode(s.secret, c.Value)
}

func (s *Server) setSession(w http.ResponseWriter, username string) error {
	value, err := auth.Encode(s.secret, auth.Session{Username: username})
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.TTL().Seconds()),
	})
	return nil
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func currentUsername(r *http.Request) string {
	if v, ok := r.Context().Value(usernameKey).(string); ok {
		return v
	}
	return ""
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render template", "name", name, "err", err)
		http.Error(w, "ошибка отрисовки страницы", http.StatusInternalServerError)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.code,
			"duration", time.Since(start),
		)
	})
}
