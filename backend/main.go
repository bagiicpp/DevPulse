package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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

func chatHandler(w http.ResponseWriter, r *http.Request) {
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
	if request.Repository == "" || request.Question == "" {
		http.Error(w, "Repository and question are required", http.StatusBadRequest)
		return
	}

	response, err := askAI(
		r.Context(),
		request.Repository,
		request.Question,
	)

	if err != nil {
		log.Printf("LLM error: %v", err)
		http.Error(
			w,
			"Failed to generate AI response",
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
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", chatHandler)
	auth, err := newAuthServer(os.Getenv("APP_URL"), os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"))
	if err != nil {
		log.Fatal(err)
	}
	auth.register(mux)

	log.Printf("DevPulse backend running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
