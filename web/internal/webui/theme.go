package webui

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	themeCookie = "panel_theme"
	themeLight  = "light"
	themeDark   = "dark"
	themeMaxAge = 365 * 24 * 60 * 60
)

type chrome struct {
	Username   string
	Theme      string
	ReturnPath string
}

func (c chrome) NextTheme() string {
	if c.Theme == themeDark {
		return themeLight
	}
	return themeDark
}

func (c chrome) ThemeLabel() string {
	if c.Theme == themeDark {
		return "Светлая тема"
	}
	return "Тёмная тема"
}

func (s *Server) pageChrome(r *http.Request) chrome {
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	return chrome{
		Username:   currentUsername(r),
		Theme:      themeFromRequest(r),
		ReturnPath: path,
	}
}

func themeFromRequest(r *http.Request) string {
	c, err := r.Cookie(themeCookie)
	if err != nil {
		return themeLight
	}
	return normalizeTheme(c.Value)
}

func normalizeTheme(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), themeDark) {
		return themeDark
	}
	return themeLight
}

func setThemeCookie(w http.ResponseWriter, theme string) {
	http.SetCookie(w, &http.Cookie{
		Name:     themeCookie,
		Value:    normalizeTheme(theme),
		Path:     "/",
		MaxAge:   themeMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) handleTheme(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	setThemeCookie(w, r.FormValue("theme"))
	http.Redirect(w, r, safeReturnPath(r), http.StatusSeeOther)
}

func safeReturnPath(r *http.Request) string {
	candidates := []string{r.FormValue("next"), r.Referer()}
	for _, raw := range candidates {
		path, ok := sameOriginPath(raw, r.Host)
		if ok {
			return path
		}
	}
	return "/"
}

func sameOriginPath(raw, host string) (string, bool) {
	if raw == "" {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Host != "" && !strings.EqualFold(u.Host, host) {
		return "", false
	}
	if u.Path == "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "", false
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery, true
	}
	return u.Path, true
}
