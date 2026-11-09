package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testAuth(t *testing.T) (*authServer, *http.ServeMux) {
	t.Helper()
	a, err := newAuthServer("http://localhost:5173", "client-id", "client-secret")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	a.register(mux)
	return a, mux
}

func beginLogin(t *testing.T, mux *http.ServeMux) (*http.Cookie, url.Values) {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil || w.Code != http.StatusFound || u.Host != "github.com" {
		t.Fatalf("unexpected sign-in redirect: %d %s", w.Code, w.Header().Get("Location"))
	}
	return w.Result().Cookies()[0], u.Query()
}

func TestGitHubOAuthSessionLifecycle(t *testing.T) {
	a, mux := testAuth(t)
	cookie, query := beginLogin(t, mux)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Secure || cookie.MaxAge != 600 {
		t.Fatalf("unexpected local state cookie: %#v", cookie)
	}
	if query.Get("client_id") != "client-id" || query.Get("redirect_uri") != "http://localhost:5173/api/auth/github/callback" || query.Get("scope") != "" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected authorization parameters: %v", query)
	}
	var verifier string
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.String() {
		case "https://github.com/login/oauth/access_token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			verifier = r.Form.Get("code_verifier")
			hash := sha256.Sum256([]byte(verifier))
			if r.Method != http.MethodPost || r.Form.Get("client_secret") != "client-secret" || r.Form.Get("code") != "temporary-code" || len(verifier) < 43 || base64.RawURLEncoding.EncodeToString(hash[:]) != query.Get("code_challenge") {
				t.Fatal("token exchange must include the secret and correct PKCE verifier")
			}
			return geminiReply(http.StatusOK, `{"access_token":"private-token","expires_in":28800,"refresh_token":"private-refresh"}`), nil
		case "https://api.github.com/user":
			if r.Header.Get("Authorization") != "Bearer private-token" {
				t.Fatal("GitHub identity must be verified with the access token")
			}
			return geminiReply(http.StatusOK, `{"id":42,"login":"octocat","name":"Octo Cat"}`), nil
		default:
			t.Fatalf("unexpected external URL: %s", r.URL)
			return nil, nil
		}
	})
	a.sessions["old-session"] = userSession{user: GitHubUser{ID: 1, Login: "old"}, expires: time.Now().Add(time.Hour)}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/github/callback?code=temporary-code&state="+query.Get("state"), nil)
	req.AddCookie(cookie)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "old-session"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "http://localhost:5173/dashboard" || verifier == "" {
		t.Fatalf("unexpected callback: %d %s", w.Code, w.Header().Get("Location"))
	}
	if _, found := a.sessions["old-session"]; found {
		t.Fatal("sign-in must rotate any existing session")
	}
	var session *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			session = c
		}
	}
	if session == nil || !session.HttpOnly || session.MaxAge != 28800 || session.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %#v", session)
	}
	check := func(wantUser bool) {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
		r.AddCookie(session)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, r)
		var data struct{ User *GitHubUser }
		if err := json.Unmarshal(out.Body.Bytes(), &data); err != nil || (data.User != nil) != wantUser {
			t.Fatalf("unexpected session: %s", out.Body.String())
		}
		if out.Header().Get("Cache-Control") != "no-store" || strings.Contains(out.Body.String(), "private-") || strings.Contains(out.Body.String(), "client-secret") {
			t.Fatal("session responses must be uncacheable and contain no tokens or secrets")
		}
	}
	check(true)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if !strings.Contains(w.Header().Get("Location"), "invalid_state") {
		t.Fatal("a callback must not be replayable")
	}
	for _, origin := range []string{"https://attacker.example", "", "http://localhost:5173"} {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		r.AddCookie(session)
		r.Header.Set("Origin", origin)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, r)
		if origin != a.origin && out.Code != http.StatusForbidden {
			t.Fatal("cross-site logout must be rejected")
		}
		if origin == a.origin && out.Code != http.StatusNoContent {
			t.Fatalf("logout failed: %d", out.Code)
		}
	}
	check(false)
}

func TestGitHubOAuthRejectsInvalidCallbacks(t *testing.T) {
	for _, name := range []string{"missing cookie", "wrong state", "expired", "denied", "missing code", "provider error", "bad profile", "invalid profile"} {
		t.Run(name, func(t *testing.T) {
			a, mux := testAuth(t)
			cookie, query := beginLogin(t, mux)
			values := url.Values{"state": {query.Get("state")}, "code": {"code"}}
			want := "invalid_state"
			switch name {
			case "wrong state":
				values.Set("state", "attacker")
			case "expired":
				attempt := a.attempts[cookie.Value]
				attempt.expires = time.Now().Add(-time.Minute)
				a.attempts[cookie.Value] = attempt
			case "denied":
				values.Set("error", "access_denied")
				want = "denied"
			case "missing code":
				values.Del("code")
				want = "failed"
			case "provider error", "bad profile", "invalid profile":
				want = "failed"
			}
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if want != "failed" || name == "missing code" {
					t.Error("invalid callback must not contact GitHub")
				}
				if name == "provider error" {
					return nil, errors.New("network failure")
				}
				if r.URL.Host == "github.com" {
					return geminiReply(http.StatusOK, `{"access_token":"private-token"}`), nil
				}
				if name == "bad profile" {
					return geminiReply(http.StatusOK, `{`), nil
				}
				return geminiReply(http.StatusOK, `{"id":42,"login":"<script>"}`), nil
			})
			r := httptest.NewRequest(http.MethodGet, "/api/auth/github/callback?"+values.Encode(), nil)
			if name != "missing cookie" {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != http.StatusFound || w.Header().Get("Location") != a.origin+"/login?error="+want || len(a.sessions) != 0 {
				t.Fatalf("unexpected callback result: %d %s", w.Code, w.Header().Get("Location"))
			}
		})
	}
}

func TestAuthConfigurationAndExpiry(t *testing.T) {
	for _, origin := range []string{"https://example.com/path", "http://example.com", "javascript:alert(1)", "https://user:pass@example.com", "https://example.com?next=evil"} {
		if _, err := newAuthServer(origin, "", ""); err == nil {
			t.Errorf("accepted invalid APP_URL %q", origin)
		}
	}
	a, mux := testAuth(t)
	a.sessions["expired"] = userSession{user: GitHubUser{ID: 1, Login: "octocat"}, expires: time.Now().Add(-time.Hour)}
	r := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "expired"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), `"user":null`) || len(a.sessions) != 0 || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("expired sessions must be removed and their cookie cleared")
	}
	a.clientSecret = ""
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/github", nil))
	if w.Header().Get("Location") != a.origin+"/login?error=not_configured" {
		t.Fatal("missing credentials must produce a recoverable sign-in error")
	}
	a.origin = "https://example.com"
	w = httptest.NewRecorder()
	a.cookie(w, sessionCookie, "opaque", 3600)
	if !w.Result().Cookies()[0].Secure {
		t.Fatal("HTTPS sessions must use Secure cookies")
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("logout must require POST")
	}
}
