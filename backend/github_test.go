package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mockGitHub(t *testing.T, transport roundTripFunc) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
}

func TestRepositorySelectionRequiresSessionAndPaginates(t *testing.T) {
	a, mux := testAuth(t)
	a.sessions["valid"] = userSession{user: GitHubUser{Login: "octocat"}, token: "server-only-token", expires: time.Now().Add(time.Hour)}
	a.sessions["expired"] = userSession{token: "expired-token", expires: time.Now().Add(-time.Second)}
	calls := 0
	mockGitHub(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.github.com/user/repos?sort=updated&per_page=100&page=2" || r.Header.Get("Authorization") != "Bearer server-only-token" {
			t.Fatalf("unexpected request: %s", r.URL)
		}
		return geminiReply(200, `[{"full_name":"octocat/private","private":true,"default_branch":"main"}]`), nil
	})
	for _, tc := range []struct {
		cookie, query string
		status        int
	}{
		{"", "", 401}, {"expired", "", 401}, {"valid", "?page=0", 400}, {"valid", "?page=abc", 400}, {"valid", "?page=2", 200},
	} {
		r := httptest.NewRequest("GET", "/api/repositories"+tc.query, nil)
		if tc.cookie != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: tc.cookie})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "server-only-token") {
			t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
		}
		if tc.status == 200 && !strings.Contains(w.Body.String(), `"private":true`) {
			t.Fatal("private repositories must be selectable")
		}
	}
	if calls != 1 {
		t.Fatalf("unexpected GitHub calls: %d", calls)
	}
}

func TestFetchRepositoryContext(t *testing.T) {
	for _, scenario := range []string{"normal", "no README", "empty", "truncated", "rate limited", "unavailable", "oversized", "invalid JSON"} {
		t.Run(scenario, func(t *testing.T) {
			mockGitHub(t, func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Authorization") != "Bearer private-token" {
					t.Fatal("repository reads must use the user's token")
				}
				switch r.URL.Path {
				case "/repos/owner/repo":
					if scenario == "unavailable" {
						return nil, errors.New("network error")
					}
					if scenario == "rate limited" {
						return geminiReply(403, "secret provider details"), nil
					}
					if scenario == "oversized" {
						return geminiReply(200, strings.Repeat(" ", (8<<20)+1)), nil
					}
					if scenario == "invalid JSON" {
						return geminiReply(200, `{`), nil
					}
					return geminiReply(200, `{"default_branch":"feature/docs"}`), nil
				case "/repos/owner/repo/git/trees/feature/docs":
					if r.URL.Query().Get("recursive") != "1" {
						t.Fatal("must retrieve nested files")
					}
					if scenario == "empty" {
						return geminiReply(409, `{}`), nil
					}
					entries := []map[string]string{{"path": "src", "type": "tree"}, {"path": "src/main.go", "type": "blob"}, {"path": "README.md", "type": "blob"}}
					if scenario == "truncated" {
						for range 1100 {
							entries = append(entries, map[string]string{"path": "file.go", "type": "blob"})
						}
					}
					body, _ := json.Marshal(map[string]any{"sha": "tree-sha", "tree": entries, "truncated": scenario == "truncated"})
					return geminiReply(200, string(body)), nil
				case "/repos/owner/repo/readme":
					if r.URL.Query().Get("ref") != "feature/docs" || r.Header.Get("Accept") != "application/vnd.github.raw+json" {
						t.Fatal("README must use the default branch and raw content")
					}
					if scenario == "no README" {
						return geminiReply(404, `{}`), nil
					}
					if scenario == "truncated" {
						return geminiReply(200, strings.Repeat("x", (64<<10)+10)), nil
					}
					return geminiReply(200, "# Project\nUses Go tests."), nil
				default:
					t.Fatalf("unexpected URL: %s", r.URL)
					return nil, nil
				}
			})
			got, err := fetchRepository(context.Background(), "owner/repo", "private-token")
			if scenario == "rate limited" || scenario == "unavailable" || scenario == "oversized" || scenario == "invalid JSON" {
				if err == nil {
					t.Fatal("failure must not produce context")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "empty":
				if len(got.Files) != 0 || got.Readme != "" {
					t.Fatal("empty repository must have empty context")
				}
			case "truncated":
				if len(got.Files) != 1000 || !got.FilesTruncated || !got.ReadmeTruncated || len(got.Readme) != 64<<10 {
					t.Fatalf("context limits not respected: %d", len(got.Files))
				}
			default:
				if len(got.Files) != 2 || got.Files[0] != "src/main.go" {
					t.Fatalf("unexpected files: %v", got.Files)
				}
				if scenario == "no README" && got.Readme != "" {
					t.Fatal("missing README must stay empty")
				}
				if scenario == "normal" && !strings.Contains(got.Readme, "Uses Go tests") {
					t.Fatal("README content missing")
				}
			}
		})
	}
}

func TestChatUsesPrivateRepositoryContext(t *testing.T) {
	a, mux := testAuth(t)
	a.ai = testAIClient(t)
	a.sessions["valid"] = userSession{token: "private-token", expires: time.Now().Add(time.Hour)}
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "test-key")
	mockGitHub(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			if r.Header.Get("Authorization") != "Bearer private-token" {
				t.Fatal("private repository must use session token")
			}
			switch r.URL.Path {
			case "/repos/owner/private":
				return geminiReply(200, `{"default_branch":"main"}`), nil
			case "/repos/owner/private/git/trees/main":
				return geminiReply(200, `{"sha":"tree","tree":[{"path":"internal/worker.go","type":"blob"}]}`), nil
			case "/repos/owner/private/readme":
				return geminiReply(200, "# Private project"), nil
			}
		}
		if r.URL.String() != geminiURL {
			t.Fatalf("unexpected URL: %s", r.URL)
		}
		var payload struct {
			Contents []struct{ Parts []struct{ Text string } }
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		prompt := payload.Contents[0].Parts[0].Text
		for _, text := range []string{"internal/worker.go", "# Private project", "owner/private", "untrusted data"} {
			if !strings.Contains(prompt, text) {
				t.Fatalf("prompt missing %q", text)
			}
		}
		if strings.Contains(prompt, "private-token") {
			t.Fatal("token must never reach Gemini")
		}
		return geminiReply(200, geminiEnvelope(`{"summary":"Based on the README","findings":[],"recommendations":[]}`)), nil
	})
	r := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":"owner/private","question":"Analyze"}`))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("private analysis failed: %d %s", w.Code, w.Body)
	}
}

func TestChatDoesNotAnalyzeFailedRetrieval(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			mockGitHub(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Host != "api.github.com" {
					t.Fatal("GitHub failure must not call Gemini")
				}
				return geminiReply(status, "private provider details"), nil
			})
			r := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":"owner/repo","question":"Analyze"}`))
			w := httptest.NewRecorder()
			(&authServer{}).chatHandler(w, r)
			want := 502
			switch status {
			case 401:
				want = 401
			case 404:
				want = 404
			case 403, 429:
				want = 503
			}
			if w.Code != want || strings.Contains(w.Body.String(), "private provider details") {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestChatRejectsCrossOriginPrivateRequests(t *testing.T) {
	a, mux := testAuth(t)
	a.sessions["valid"] = userSession{token: "private-token", expires: time.Now().Add(time.Hour)}
	mockGitHub(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("cross-origin requests must not retrieve repository data")
		return nil, nil
	})
	r := httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":"owner/private","question":"Analyze"}`))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	r.Header.Set("Origin", "https://attacker.example")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}
