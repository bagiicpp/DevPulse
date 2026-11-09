package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const sessionCookie = "devpulse_session"
const stateCookie = "devpulse_oauth_state"

var githubLogin = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}$`)

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

type loginAttempt struct {
	verifier string
	expires  time.Time
}

type userSession struct {
	user    GitHubUser
	token   string
	expires time.Time
}

type authServer struct {
	origin, clientID, clientSecret string
	mu                             sync.Mutex
	// shortcut: sessions live in one server's memory, use shared storage before running multiple instances.
	attempts map[string]loginAttempt
	sessions map[string]userSession
}

func newAuthServer(origin, clientID, clientSecret string) (*authServer, error) {
	if origin == "" {
		origin = "http://localhost:5173"
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("APP_URL must be an origin such as http://localhost:5173")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return nil, fmt.Errorf("APP_URL requires HTTPS except on localhost")
	}
	return &authServer{
		origin: strings.TrimRight(origin, "/"), clientID: clientID, clientSecret: clientSecret,
		attempts: make(map[string]loginAttempt), sessions: make(map[string]userSession),
	}, nil
}

func (a *authServer) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth/session", a.session)
	mux.HandleFunc("GET /api/auth/github", a.start)
	mux.HandleFunc("GET /api/auth/github/callback", a.callback)
	mux.HandleFunc("POST /api/auth/logout", a.logout)
	mux.HandleFunc("GET /api/repositories", a.repositories)
	mux.HandleFunc("/api/chat", a.chatHandler)
}

func (a *authServer) currentSession(r *http.Request) (userSession, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return userSession{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	session, found := a.sessions[cookie.Value]
	if !found || !time.Now().Before(session.expires) {
		delete(a.sessions, cookie.Value)
		return userSession{}, false
	}
	return session, true
}

func (a *authServer) configured() bool {
	return a.clientID != "" && a.clientSecret != ""
}

func (a *authServer) cookie(w http.ResponseWriter, name, value string, age int) {
	path := "/"
	if name == stateCookie {
		path = "/api/auth/github"
	}
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: path, MaxAge: age,
		HttpOnly: true, Secure: strings.HasPrefix(a.origin, "https://"), SameSite: http.SameSiteLaxMode,
	})
}

func (a *authServer) session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	var user *GitHubUser
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		session, found := a.sessions[cookie.Value]
		if found && time.Now().Before(session.expires) {
			user = &session.user
		} else {
			delete(a.sessions, cookie.Value)
			a.cookie(w, sessionCookie, "", -1)
		}
		a.mu.Unlock()
	}
	if err := json.NewEncoder(w).Encode(struct {
		User             *GitHubUser `json:"user"`
		GitHubConfigured bool        `json:"githubConfigured"`
	}{user, a.configured()}); err != nil {
		log.Printf("Failed to encode session: %v", err)
	}
}

func (a *authServer) start(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !a.configured() {
		http.Redirect(w, r, a.origin+"/login?error=not_configured", http.StatusFound)
		return
	}
	state, verifier := rand.Text(), rand.Text()+rand.Text()
	now := time.Now()
	a.mu.Lock()
	for key, attempt := range a.attempts {
		if !now.Before(attempt.expires) {
			delete(a.attempts, key)
		}
	}
	for key, session := range a.sessions {
		if !now.Before(session.expires) {
			delete(a.sessions, key)
		}
	}
	a.attempts[state] = loginAttempt{verifier: verifier, expires: now.Add(10 * time.Minute)}
	a.mu.Unlock()
	a.cookie(w, stateCookie, state, 600)
	hash := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id": {a.clientID}, "redirect_uri": {a.origin + "/api/auth/github/callback"},
		"state": {state}, "scope": {"repo"}, "code_challenge_method": {"S256"},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(hash[:])},
	}
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+query.Encode(), http.StatusFound)
}

func githubJSON(req *http.Request, target any) error {
	client := &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(body) > 1<<20 {
		return fmt.Errorf("GitHub response exceeds 1 MiB")
	}
	return json.Unmarshal(body, target)
}

func (a *authServer) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	fail := func(reason string) {
		http.Redirect(w, r, a.origin+"/login?error="+reason, http.StatusFound)
	}
	cookie, err := r.Cookie(stateCookie)
	a.cookie(w, stateCookie, "", -1)
	state := r.URL.Query().Get("state")
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(state)) != 1 {
		fail("invalid_state")
		return
	}
	a.mu.Lock()
	attempt, found := a.attempts[state]
	delete(a.attempts, state)
	a.mu.Unlock()
	if !found || !time.Now().Before(attempt.expires) {
		fail("invalid_state")
		return
	}
	if r.URL.Query().Get("error") != "" {
		fail("denied")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" || !a.configured() {
		fail("failed")
		return
	}
	values := url.Values{
		"client_id": {a.clientID}, "client_secret": {a.clientSecret}, "code": {code},
		"redirect_uri": {a.origin + "/api/auth/github/callback"}, "code_verifier": {attempt.verifier},
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(values.Encode()))
	if err != nil {
		fail("failed")
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := githubJSON(req, &token); err != nil || token.AccessToken == "" {
		log.Print("GitHub token exchange failed")
		fail("failed")
		return
	}
	req, err = http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		fail("failed")
		return
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	var user GitHubUser
	if err := githubJSON(req, &user); err != nil || user.ID <= 0 || !githubLogin.MatchString(user.Login) {
		log.Print("GitHub identity verification failed")
		fail("failed")
		return
	}
	id := rand.Text() + rand.Text()
	lifetime := 8 * time.Hour
	if token.ExpiresIn > 0 && token.ExpiresIn < int64(lifetime/time.Second) {
		lifetime = time.Duration(token.ExpiresIn) * time.Second
	}
	a.mu.Lock()
	if old, err := r.Cookie(sessionCookie); err == nil {
		delete(a.sessions, old.Value)
	}
	a.sessions[id] = userSession{user: user, token: token.AccessToken, expires: time.Now().Add(lifetime)}
	a.mu.Unlock()
	a.cookie(w, sessionCookie, id, int(lifetime/time.Second))
	http.Redirect(w, r, a.origin+"/dashboard", http.StatusFound)
}

func (a *authServer) logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("Origin") != a.origin {
		http.Error(w, "Invalid request origin", http.StatusForbidden)
		return
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
	}
	a.cookie(w, sessionCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
