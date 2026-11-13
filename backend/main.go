package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	aipb "devpulse/backend/proto"

	"github.com/joho/godotenv"
)

type ChatRequest struct {
	Repository string `json:"repository"`
	Question   string `json:"question"`
}

type ChatResponse struct {
	Summary         string   `json:"summary"`
	Findings        []string `json:"findings"`
	Recommendations []string `json:"recommendations"`
}

func (a *authServer) chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<10))
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			http.Error(w, "Request must be at most 16 KiB", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "Could not read request", http.StatusBadRequest)
		}
		return
	}

	var request ChatRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	request.Repository = strings.TrimSpace(request.Repository)
	request.Question = strings.TrimSpace(request.Question)
	if !repositoryName.MatchString(request.Repository) || request.Question == "" {
		http.Error(w, "Use owner/repository and provide a question", http.StatusBadRequest)
		return
	}

	session, _ := a.currentSession(r)
	if session.token != "" && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != a.origin {
		http.Error(w, "Invalid request origin", http.StatusForbidden)
		return
	}
	repository, err := fetchRepository(r.Context(), request.Repository, session.token)
	if err != nil {
		writeGitHubError(w, err)
		return
	}
	response, err := a.analyze(
		r.Context(),
		repository,
		request.Question,
	)

	if err != nil {
		log.Printf("LLM error: %v", err)
		http.Error(
			w,
			"AI service could not generate a response. Please try again.",
			http.StatusBadGateway,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func main() {
	aiService := flag.Bool("ai-service", false, "Run the AI gRPC service instead of the HTTP API")
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	if *aiService {
		if err := serveAI(); err != nil {
			log.Fatal(err)
		}
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	auth, err := newAuthServer(os.Getenv("APP_URL"), os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"))
	if err != nil {
		log.Fatal(err)
	}
	connection, err := newAIClient()
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()
	auth.ai = aipb.NewAIServiceClient(connection)
	auth.register(mux)

	log.Printf("DevPulse backend running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
