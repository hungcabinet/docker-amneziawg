package webui

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"

	"github.com/hungcabinet/docker-amneziawg/web/internal/status"
	"github.com/hungcabinet/docker-amneziawg/web/internal/store"
	"github.com/skip2/go-qrcode"
)

var peerNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type loginPage struct {
	chrome
	Error string
}

type accountPage struct {
	chrome
	Error string
	Flash string
}

type peersPage struct {
	chrome
	Server status.ServerStatus
	Peers  []status.Peer
	Error  string
}

type exitsPage struct {
	chrome
	Exits        []status.Exit
	SelectedExit string
	IPIface      string
	IP           string
	Error        string
	Flash        string
}

func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessionFromRequest(r); err == nil {
		http.Redirect(w, r, "/peers", http.StatusSeeOther)
		return
	}
	s.render(w, "login", loginPage{chrome: s.pageChrome(r)})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, "login", loginPage{chrome: s.pageChrome(r), Error: "не удалось прочитать форму"})
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	if err := s.store.Authenticate(r.Context(), username, password); err != nil {
		s.render(w, "login", loginPage{chrome: s.pageChrome(r), Error: "неверный логин или пароль"})
		return
	}
	if err := s.setSession(w, username); err != nil {
		s.render(w, "login", loginPage{chrome: s.pageChrome(r), Error: "не удалось создать сессию"})
		return
	}
	http.Redirect(w, r, "/peers", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSession(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/peers", http.StatusSeeOther)
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	page := peersPage{chrome: s.pageChrome(r)}
	server, err := s.provider.Server(r.Context())
	if err != nil {
		page.Error = userFacing(err)
		s.render(w, "peers", page)
		return
	}
	page.Server = server
	if !server.Up {
		page.Error = "сервер не поднят"
		s.render(w, "peers", page)
		return
	}
	peers, err := s.provider.Peers(r.Context())
	if err != nil {
		page.Error = userFacing(err)
		s.render(w, "peers", page)
		return
	}
	page.Peers = peers
	s.render(w, "peers", page)
}

func (s *Server) handlePeerQR(w http.ResponseWriter, r *http.Request) {
	peer, ok := s.lookupPeer(w, r)
	if !ok {
		return
	}
	png, err := qrcode.Encode(peer.Config, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "не удалось построить QR-код", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

func (s *Server) handlePeerConfig(w http.ResponseWriter, r *http.Request) {
	peer, ok := s.lookupPeer(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename="+peer.Name+".conf")
	_, _ = w.Write([]byte(peer.Config))
}

func (s *Server) lookupPeer(w http.ResponseWriter, r *http.Request) (status.Peer, bool) {
	name := r.PathValue("name")
	if !peerNameRe.MatchString(name) {
		http.NotFound(w, r)
		return status.Peer{}, false
	}
	peers, err := s.provider.Peers(r.Context())
	if err != nil {
		http.Error(w, userFacing(err), http.StatusBadGateway)
		return status.Peer{}, false
	}
	for _, peer := range peers {
		if peer.Name == name {
			return peer, true
		}
	}
	http.NotFound(w, r)
	return status.Peer{}, false
}

func (s *Server) handleExits(w http.ResponseWriter, r *http.Request) {
	page := exitsPage{
		chrome:  s.pageChrome(r),
		IPIface: r.URL.Query().Get("ip_iface"),
		IP:      r.URL.Query().Get("ip"),
		Error:   r.URL.Query().Get("error"),
		Flash:   r.URL.Query().Get("ok"),
	}
	selected, err := s.store.SelectedExit(r.Context())
	if err != nil {
		page.Error = "не удалось прочитать выбранный выход"
		s.render(w, "exits", page)
		return
	}
	page.SelectedExit = selected
	exits, err := s.provider.Exits(r.Context())
	if err != nil {
		page.Error = userFacing(err)
		s.render(w, "exits", page)
		return
	}
	page.Exits = exits
	s.render(w, "exits", page)
}

func (s *Server) handleSelectExit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape("не удалось прочитать форму"), http.StatusSeeOther)
		return
	}
	iface := r.FormValue("iface")
	if !peerNameRe.MatchString(iface) {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape("некорректное имя интерфейса"), http.StatusSeeOther)
		return
	}
	if err := s.store.SetSelectedExit(r.Context(), iface); err != nil {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape("не удалось сохранить выход"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/exits?ok="+url.QueryEscape("выход сохранён: "+iface), http.StatusSeeOther)
}

func (s *Server) handleExitIP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape("не удалось прочитать форму"), http.StatusSeeOther)
		return
	}
	iface := r.FormValue("iface")
	if !peerNameRe.MatchString(iface) {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape("некорректное имя интерфейса"), http.StatusSeeOther)
		return
	}
	ip, err := s.provider.ExitIP(r.Context(), iface)
	if err != nil {
		http.Redirect(w, r, "/exits?error="+url.QueryEscape(userFacing(err)), http.StatusSeeOther)
		return
	}
	q := url.Values{}
	q.Set("ip_iface", iface)
	q.Set("ip", ip)
	http.Redirect(w, r, "/exits?"+q.Encode(), http.StatusSeeOther)
}

func (s *Server) handleAccountGet(w http.ResponseWriter, r *http.Request) {
	s.render(w, "account", accountPage{
		chrome: s.pageChrome(r),
		Flash:  r.URL.Query().Get("ok"),
		Error:  r.URL.Query().Get("error"),
	})
}

func (s *Server) handleAccountPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render(w, "account", accountPage{
			chrome: s.pageChrome(r),
			Error:  "не удалось прочитать форму",
		})
		return
	}
	current := r.FormValue("current_password")
	newUser := r.FormValue("new_username")
	newPass := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")
	if newPass != "" && newPass != confirm {
		s.render(w, "account", accountPage{
			chrome: s.pageChrome(r),
			Error:  "новый пароль и подтверждение не совпадают",
		})
		return
	}
	err := s.store.ChangeCredentials(r.Context(), current, newUser, newPass)
	if err != nil {
		s.render(w, "account", accountPage{
			chrome: s.pageChrome(r),
			Error:  credentialsError(err),
		})
		return
	}
	username := currentUsername(r)
	if newUser != "" {
		username = newUser
	}
	if err := s.setSession(w, username); err != nil {
		page := accountPage{
			chrome: s.pageChrome(r),
			Error:  "учётные данные обновлены, но сессию обновить не удалось — войдите снова",
		}
		page.Username = username
		s.render(w, "account", page)
		return
	}
	http.Redirect(w, r, "/account?ok="+url.QueryEscape("учётные данные обновлены"), http.StatusSeeOther)
}

func userFacing(err error) string {
	if errors.Is(err, status.ErrNotConnected) {
		return "живой источник данных ещё не подключён"
	}
	return "не удалось получить данные"
}

func credentialsError(err error) string {
	switch {
	case errors.Is(err, store.ErrInvalidPassword):
		return "текущий пароль неверный"
	case errors.Is(err, store.ErrNothingToChange):
		return "укажите новый логин или пароль"
	default:
		return "не удалось сохранить изменения"
	}
}
