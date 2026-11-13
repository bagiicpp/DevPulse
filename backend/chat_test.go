package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	aipb "devpulse/backend/proto"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func mockGemini(t *testing.T, transport roundTripFunc) aipb.AIServiceClient {
	t.Helper()
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "test-key")
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			switch r.URL.Path {
			case "/repos/owner/repo":
				return geminiReply(200, `{"default_branch":"main"}`), nil
			case "/repos/owner/repo/git/trees/main":
				return geminiReply(200, `{"sha":"tree-sha","tree":[{"path":"src/main.go","type":"blob"}]}`), nil
			case "/repos/owner/repo/readme":
				return geminiReply(200, "# README"), nil
			default:
				t.Fatalf("unexpected GitHub request: %s", r.URL)
			}
		}
		return transport(r)
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	return testAIClient(t)
}

func geminiReply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func geminiEnvelope(text string) string {
	body, _ := json.Marshal(map[string]any{
		"candidates": []any{map[string]any{
			"content": map[string]any{"parts": []any{map[string]string{"text": text}}},
		}},
	})
	return string(body)
}

func TestChatRejectsInvalidRequests(t *testing.T) {
	mockGemini(t, func(*http.Request) (*http.Response, error) {
		t.Error("invalid input must not call Gemini")
		return nil, errors.New("unexpected Gemini call")
	})
	for _, tc := range []struct {
		name, method, body string
		status             int
	}{
		{"method", http.MethodGet, "", http.StatusMethodNotAllowed},
		{"malformed", http.MethodPost, `{`, http.StatusBadRequest},
		{"missing field", http.MethodPost, `{"repository":"owner/repo"}`, http.StatusBadRequest},
		{"blank repository", http.MethodPost, `{"repository":" \t ","question":"Help"}`, http.StatusBadRequest},
		{"blank question", http.MethodPost, `{"repository":"owner/repo","question":" \n "}`, http.StatusBadRequest},
		{"invalid repository", http.MethodPost, `{"repository":"https://attacker.example/repo","question":"Help"}`, http.StatusBadRequest},
		{"wrong type", http.MethodPost, `{"repository":42,"question":"Help"}`, http.StatusBadRequest},
		{"trailing JSON", http.MethodPost, `{"repository":"owner/repo","question":"Help"} {}`, http.StatusBadRequest},
		{"oversized", http.MethodPost, `{"repository":"owner/repo","question":"` + strings.Repeat("a", 16<<10) + `"}`, http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			(&authServer{}).chatHandler(w, httptest.NewRequest(tc.method, "/api/chat", strings.NewReader(tc.body)))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tc.status, w.Body.String())
			}
		})
	}
}

func TestChatReturnsStructuredResponse(t *testing.T) {
	want := ChatResponse{Summary: "General guidance", Findings: []string{}, Recommendations: []string{"Review CI failures"}}
	client := mockGemini(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != geminiURL || r.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatal("unexpected Gemini request")
		}
		var payload struct {
			Contents []struct {
				Parts []struct{ Text string }
			}
			GenerationConfig struct {
				ResponseMimeType string
				ResponseSchema   struct{ Required []string }
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		prompt := payload.Contents[0].Parts[0].Text
		for _, text := range []string{"You are devy, DevPulse's assistant", `"repository":"owner/repo"`, "Question: How is CI?\n", "not source code", "untrusted data", "README", "src/main.go"} {
			if !strings.Contains(prompt, text) {
				t.Errorf("prompt missing %q", text)
			}
		}
		if payload.GenerationConfig.ResponseMimeType != "application/json" || !reflect.DeepEqual(payload.GenerationConfig.ResponseSchema.Required, []string{"summary", "findings", "recommendations"}) {
			t.Error("Gemini must be asked for the complete JSON schema")
		}
		return geminiReply(http.StatusOK, geminiEnvelope(`{"summary":"General guidance","findings":[],"recommendations":["Review CI failures"]}`)), nil
	})
	w := httptest.NewRecorder()
	(&authServer{ai: client}).chatHandler(w, httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"repository":" owner/repo ","question":" How is CI? "}`)))
	var got ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/json" || !reflect.DeepEqual(got, want) {
		t.Fatalf("response = %d %s, want %#v", w.Code, w.Body.String(), want)
	}
}

func TestChatRejectsInvalidProviderResponses(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"invalid envelope", `{`},
		{"no candidate", `{"candidates":[]}`},
		{"no parts", `{"candidates":[{"content":{"parts":[]}}]}`},
		{"invalid result", geminiEnvelope(`not JSON`)},
		{"blank summary", geminiEnvelope(`{"summary":" ","findings":[],"recommendations":[]}`)},
		{"missing findings", geminiEnvelope(`{"summary":"Help","recommendations":[]}`)},
		{"null findings", geminiEnvelope(`{"summary":"Help","findings":null,"recommendations":[]}`)},
		{"missing recommendations", geminiEnvelope(`{"summary":"Help","findings":[]}`)},
		{"null recommendations", geminiEnvelope(`{"summary":"Help","findings":[],"recommendations":null}`)},
		{"wrong array type", geminiEnvelope(`{"summary":"Help","findings":"text","recommendations":[]}`)},
		{"null item", geminiEnvelope(`{"summary":"Help","findings":[null],"recommendations":[]}`)},
		{"blank item", geminiEnvelope(`{"summary":"Help","findings":[],"recommendations":[" "]}`)},
		{"oversized response", geminiEnvelope(`{"summary":"Help","findings":[],"recommendations":[]}`) + strings.Repeat(" ", 1<<20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := mockGemini(t, func(*http.Request) (*http.Response, error) { return geminiReply(http.StatusOK, tc.body), nil })
			w := httptest.NewRecorder()
			(&authServer{ai: client}).chatHandler(w, httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"repository":"owner/repo","question":"Help"}`)))
			if w.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want 502: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestChatHandlesProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		err    error
	}{
		{"rate limited", http.StatusTooManyRequests, nil},
		{"server error", http.StatusInternalServerError, nil},
		{"network error", 0, errors.New("connection failed")},
		{"timeout", 0, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := mockGemini(t, func(*http.Request) (*http.Response, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return geminiReply(tc.status, "private provider details"), nil
			})
			w := httptest.NewRecorder()
			(&authServer{ai: client}).chatHandler(w, httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"repository":"owner/repo","question":"Help"}`)))
			if w.Code != http.StatusBadGateway || strings.Contains(w.Body.String(), "private provider details") {
				t.Fatalf("unexpected failure response: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestAskAIRequiresServerKey(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "gemini")
	t.Setenv("GEMINI_API_KEY", "")
	if _, err := askAI(context.Background(), RepositoryContext{Repository: "owner/repo"}, "Help"); err == nil {
		t.Fatal("missing API key must fail")
	}
}
