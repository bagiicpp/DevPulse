package main

import (
	"encoding/json"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Repository == "" || request.Question == "" {
		http.Error(w, "Repository and question are required", http.StatusBadRequest)
		return
	}

	log.Printf("Repository: %s | Question: %s",
		request.Repository, request.Question)

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

	log.Printf("DevPulse backend running on http://localhost:%s", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
