package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	aipb "devpulse/backend/proto"
)

func openRouterEnvelope(content string) string {
	body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}, "finish_reason": "stop"}}})
	return string(body)
}

func mockOpenRouter(t *testing.T, transport roundTripFunc) aipb.AIServiceClient {
	t.Helper()
	client := mockGemini(t, transport)
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OPENROUTER_API_KEY", "openrouter-test-key")
	t.Setenv("OPENROUTER_MODEL", "")
	return client
}

func TestOpenRouterReturnsStructuredRepositoryAnalysis(t *testing.T) {
	for _, model := range []string{"", "openrouter/free"} {
		t.Run("model="+model, func(t *testing.T) {
			client := mockOpenRouter(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.String() != openRouterURL || r.Header.Get("Authorization") != "Bearer openrouter-test-key" || r.Header.Get("x-goog-api-key") != "" {
					t.Fatal("unexpected OpenRouter request")
				}
				var payload struct {
					Model    string
					Messages []struct{ Role, Content string }
					Provider struct {
						RequireParameters bool `json:"require_parameters"`
					}
					ResponseFormat struct {
						Type       string
						JSONSchema struct {
							Strict bool
							Schema struct {
								Type                 string
								Required             []string
								AdditionalProperties *bool
								Properties           map[string]struct {
									Type  string
									Items struct{ Type string }
								}
							}
						} `json:"json_schema"`
					} `json:"response_format"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				wantModel := model
				if wantModel == "" {
					wantModel = defaultOpenRouterModel
				}
				if payload.Model != wantModel || !payload.Provider.RequireParameters || payload.ResponseFormat.Type != "json_schema" || !payload.ResponseFormat.JSONSchema.Strict {
					t.Fatal("must request the configured model with strict structured outputs")
				}
				schema := payload.ResponseFormat.JSONSchema.Schema
				if schema.Type != "object" || schema.AdditionalProperties == nil || *schema.AdditionalProperties || !reflect.DeepEqual(schema.Required, []string{"summary", "findings", "recommendations"}) {
					t.Fatal("incomplete JSON schema")
				}
				if schema.Properties["summary"].Type != "string" || schema.Properties["findings"].Items.Type != "string" || schema.Properties["recommendations"].Items.Type != "string" {
					t.Fatal("wrong response field types")
				}
				if len(payload.Messages) != 1 || payload.Messages[0].Role != "user" {
					t.Fatal("expected a dynamic user prompt")
				}
				for _, text := range []string{"owner/repo", "Question: How is CI?", "src/main.go", "# README", "untrusted data", "not source code", `{"summary":"...","findings":["..."],"recommendations":["..."]}`, "nonblank strings"} {
					if !strings.Contains(payload.Messages[0].Content, text) {
						t.Fatalf("prompt missing %q", text)
					}
				}
				if strings.Contains(payload.Messages[0].Content, "openrouter-test-key") {
					t.Fatal("key must not be included in the prompt")
				}
				return geminiReply(200, openRouterEnvelope(`{"summary":"Repository analysis","findings":["README provided"],"recommendations":[]}`)), nil
			})
			t.Setenv("OPENROUTER_MODEL", model)
			w := httptest.NewRecorder()
			(&authServer{ai: client}).chatHandler(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":" owner/repo ","question":" How is CI? "}`)))
			var got ChatResponse
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || got.Summary != "Repository analysis" || len(got.Findings) != 1 || got.Recommendations == nil {
				t.Fatalf("unexpected analysis: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestOpenRouterRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"malformed envelope", `{`},
		{"no choices", `{"choices":[]}`},
		{"provider error in 200", `{"error":{"message":"private provider details"}}`},
		{"incomplete response", `{"choices":[{"finish_reason":"length","message":{"content":"{}"}}]}`},
		{"empty content", openRouterEnvelope("")},
		{"invalid JSON content", openRouterEnvelope("not JSON")},
		{"missing arrays", openRouterEnvelope(`{"summary":"Help"}`)},
		{"null arrays", openRouterEnvelope(`{"summary":"Help","findings":null,"recommendations":[]}`)},
		{"empty summary", openRouterEnvelope(`{"summary":" ","findings":[],"recommendations":[]}`)},
		{"empty list item", openRouterEnvelope(`{"summary":"Help","findings":[" "],"recommendations":[]}`)},
		{"oversized response", strings.Repeat(" ", (1<<20)+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := mockOpenRouter(t, func(*http.Request) (*http.Response, error) { return geminiReply(200, tc.body), nil })
			w := httptest.NewRecorder()
			(&authServer{ai: client}).chatHandler(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":"owner/repo","question":"Help"}`)))
			if w.Code != 502 || strings.Contains(w.Body.String(), "private provider details") {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body)
			}
		})
	}
}

func TestOpenRouterHandlesProviderFailures(t *testing.T) {
	for _, status := range []int{401, 402, 429, 503, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			mockOpenRouter(t, func(*http.Request) (*http.Response, error) {
				calls++
				response := geminiReply(status, "private provider details")
				response.Header = http.Header{"Location": []string{"https://other-provider.example"}}
				return response, nil
			})
			_, err := askAI(context.Background(), RepositoryContext{Repository: "owner/repo"}, "Help")
			if err == nil || calls != 1 || strings.Contains(err.Error(), "private provider details") {
				t.Fatalf("unexpected provider failure: %v, calls=%d", err, calls)
			}
		})
	}
	for _, failure := range []error{errors.New("connection failed"), context.DeadlineExceeded} {
		mockOpenRouter(t, func(*http.Request) (*http.Response, error) { return nil, failure })
		if _, err := askAI(context.Background(), RepositoryContext{}, "Help"); err == nil {
			t.Fatal("provider network failures must be returned")
		}
	}
}

func TestAIRejectsMissingKeyAndUnknownProvider(t *testing.T) {
	mockOpenRouter(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid configuration must not call a provider")
		return nil, nil
	})
	t.Setenv("OPENROUTER_API_KEY", "")
	if _, err := askAI(context.Background(), RepositoryContext{}, "Help"); err == nil || !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Fatal("missing OpenRouter key must be reported")
	}
	t.Setenv("LLM_PROVIDER", "unknown")
	if _, err := askAI(context.Background(), RepositoryContext{}, "Help"); err == nil || !strings.Contains(err.Error(), "LLM_PROVIDER") {
		t.Fatal("invalid provider must be reported")
	}
}
