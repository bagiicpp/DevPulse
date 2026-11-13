package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aipb "devpulse/backend/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func testAIClient(t *testing.T) aipb.AIServiceClient {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	server := newAIServer()
	go func() {
		if err := server.Serve(listener); err != nil {
			t.Errorf("gRPC server: %v", err)
		}
	}()
	connection, err := grpc.NewClient("passthrough:///ai-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1<<20)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		connection.Close()
		server.Stop()
		listener.Close()
	})
	return aipb.NewAIServiceClient(connection)
}

func TestAIRejectsInvalidRPCRequests(t *testing.T) {
	client := mockGemini(t, func(*http.Request) (*http.Response, error) {
		t.Error("invalid gRPC input must not call the provider")
		return nil, errors.New("unexpected provider call")
	})
	for _, tc := range []struct {
		name     string
		context  *aipb.RepositoryContext
		question string
		code     codes.Code
	}{
		{"missing context", nil, "Help", codes.InvalidArgument},
		{"invalid name", &aipb.RepositoryContext{Repository: "https://example.com/repo"}, "Help", codes.InvalidArgument},
		{"blank question", &aipb.RepositoryContext{Repository: "owner/repo"}, " \n ", codes.InvalidArgument},
		{"long question", &aipb.RepositoryContext{Repository: "owner/repo"}, strings.Repeat("a", (16<<10)+1), codes.InvalidArgument},
		{"long readme", &aipb.RepositoryContext{Repository: "owner/repo", Readme: strings.Repeat("a", (64<<10)+1)}, "Help", codes.InvalidArgument},
		{"too many files", &aipb.RepositoryContext{Repository: "owner/repo", Files: make([]string, 1001)}, "Help", codes.InvalidArgument},
		{"long paths", &aipb.RepositoryContext{Repository: "owner/repo", Files: []string{strings.Repeat("a", (64<<10)+1)}}, "Help", codes.InvalidArgument},
		{"long ref", &aipb.RepositoryContext{Repository: "owner/repo", Ref: strings.Repeat("a", 1025)}, "Help", codes.InvalidArgument},
		{"message limit", &aipb.RepositoryContext{Repository: "owner/repo", Readme: strings.Repeat("a", 256<<10)}, "Help", codes.ResourceExhausted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_, err := client.Analyze(ctx, &aipb.AnalyzeRequest{Context: tc.context, Question: tc.question})
			if status.Code(err) != tc.code {
				t.Fatalf("code = %v, want %v: %v", status.Code(err), tc.code, err)
			}
		})
	}
}

func TestAICancellationReachesProvider(t *testing.T) {
	for _, code := range []codes.Code{codes.Canceled, codes.DeadlineExceeded} {
		t.Run(code.String(), func(t *testing.T) {
			started, stopped := make(chan struct{}), make(chan struct{})
			client := mockGemini(t, func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > aiTimeout {
					t.Error("provider request must have a bounded deadline")
				}
				close(started)
				<-r.Context().Done()
				close(stopped)
				return nil, r.Context().Err()
			})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := (&authServer{ai: client}).analyze(ctx, RepositoryContext{Repository: "owner/repo"}, "Help")
				result <- err
			}()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("provider request did not start")
			}
			if code == codes.Canceled {
				cancel()
			}
			if err := <-result; status.Code(err) != code {
				t.Fatalf("code = %v, want %v: %v", status.Code(err), code, err)
			}
			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("gRPC cancellation did not reach the provider")
			}
		})
	}
}

func TestChatHandlesUnavailableAIService(t *testing.T) {
	mockGemini(t, func(*http.Request) (*http.Response, error) {
		t.Error("unavailable gRPC service must not fall back to a direct provider call")
		return nil, errors.New("unexpected provider call")
	})
	connection, err := grpc.NewClient("passthrough:///unavailable-ai",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return nil, errors.New("offline") }))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	a := &authServer{ai: aipb.NewAIServiceClient(connection)}
	w := httptest.NewRecorder()
	a.chatHandler(w, httptest.NewRequest("POST", "/api/chat", strings.NewReader(`{"repository":"owner/repo","question":"Help"}`)))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "Please try again") || strings.Contains(w.Body.String(), "offline") {
		t.Fatalf("unexpected unavailable-service response: %d %s", w.Code, w.Body)
	}
}

func TestAIContextPreservedAcrossRPC(t *testing.T) {
	client := mockGemini(t, func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		for _, expected := range []string{`\"ref\":\"feature/test\"`, `\"filesTruncated\":true`, `\"readmeTruncated\":true`, `\"files\":[]`} {
			if !strings.Contains(string(body), expected) {
				t.Errorf("prompt lost repository context %s", expected)
			}
		}
		return geminiReply(200, geminiEnvelope(`{"summary":"Limited context","findings":[],"recommendations":[]}`)), nil
	})
	response, err := (&authServer{ai: client}).analyze(context.Background(), RepositoryContext{
		Repository: "owner/repo", Ref: "feature/test", FilesTruncated: true, ReadmeTruncated: true,
	}, "Help")
	if err != nil || response.Findings == nil || response.Recommendations == nil {
		t.Fatalf("empty arrays must survive protobuf: %#v, %v", response, err)
	}
}
