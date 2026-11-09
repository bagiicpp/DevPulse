package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.5-flash-lite:generateContent"

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func askAI(ctx context.Context, repository, question string) (ChatResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return ChatResponse{}, fmt.Errorf("GEMINI_API_KEY is not set")
	}

	prompt := fmt.Sprintf(
		`You are DevPulse, a software development assistant.

The user is asking about repository: %s
Question: %s

Answer the user's question concisely.
Provide a summary, findings, and recommendations.
Do not claim to have inspected the repository.
No live GitHub data has been provided.`,
		repository, question,
	)

	payload := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]string{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"responseSchema": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"summary": map[string]string{
						"type": "STRING",
					},
					"findings": map[string]any{
						"type": "ARRAY",
						"items": map[string]string{
							"type": "STRING",
						},
					},
					"recommendations": map[string]any{
						"type": "ARRAY",
						"items": map[string]string{
							"type": "STRING",
						},
					},
				},
				"required": []string{
					"summary",
					"findings",
					"recommendations",
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		geminiURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return ChatResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return ChatResponse{}, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return ChatResponse{}, err
	}
	if len(responseBody) > 1<<20 {
		return ChatResponse{}, fmt.Errorf("Gemini response exceeds 1 MiB")
	}

	if resp.StatusCode != http.StatusOK {
		return ChatResponse{}, fmt.Errorf(
			"Gemini API returned status %d",
			resp.StatusCode,
		)
	}

	var gemini GeminiResponse
	if err := json.Unmarshal(responseBody, &gemini); err != nil {
		return ChatResponse{}, err
	}

	if len(gemini.Candidates) == 0 ||
		len(gemini.Candidates[0].Content.Parts) == 0 {
		return ChatResponse{}, fmt.Errorf("Gemini returned no response")
	}

	var result ChatResponse

	text := gemini.Candidates[0].Content.Parts[0].Text
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return ChatResponse{}, err
	}

	if strings.TrimSpace(result.Summary) == "" {
		return ChatResponse{}, fmt.Errorf("Gemini returned an empty summary")
	}
	if result.Findings == nil || result.Recommendations == nil {
		return ChatResponse{}, fmt.Errorf("Gemini must return findings and recommendations arrays")
	}
	for _, items := range [][]string{result.Findings, result.Recommendations} {
		for _, item := range items {
			if strings.TrimSpace(item) == "" {
				return ChatResponse{}, fmt.Errorf("Gemini returned an empty list item")
			}
		}
	}

	return result, nil
}
