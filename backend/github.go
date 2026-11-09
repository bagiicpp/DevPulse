package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var repositoryName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}/[a-zA-Z0-9_.-]{1,100}$`)

type Repository struct {
	FullName      string `json:"full_name"`
	Description   string `json:"description"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}

type RepositoryContext struct {
	Repository      string   `json:"repository"`
	Ref             string   `json:"ref"`
	Files           []string `json:"files"`
	FilesTruncated  bool     `json:"filesTruncated"`
	Readme          string   `json:"readme"`
	ReadmeTruncated bool     `json:"readmeTruncated"`
}

type githubStatusError int

func (s githubStatusError) Error() string { return fmt.Sprintf("GitHub returned status %d", s) }

// Only fixed GitHub API paths are accepted; never follow repository-provided URLs.
func githubGet(ctx context.Context, path, token, accept string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, githubStatusError(response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, limit+1))
}

func githubData(ctx context.Context, path, token string, target any) error {
	const limit = 8 << 20
	body, err := githubGet(ctx, path, token, "application/vnd.github+json", limit)
	if err != nil {
		return err
	}
	if len(body) > limit {
		return fmt.Errorf("GitHub response exceeds 8 MiB")
	}
	return json.Unmarshal(body, target)
}

func writeGitHubError(w http.ResponseWriter, err error) {
	log.Printf("Repository retrieval failed: %v", err)
	var status githubStatusError
	if errors.As(err, &status) {
		switch int(status) {
		case http.StatusUnauthorized:
			http.Error(w, "GitHub access expired. Sign out and sign in again.", http.StatusUnauthorized)
			return
		case http.StatusNotFound:
			http.Error(w, "Repository not found or not accessible. For private repositories, sign in again and grant repository access.", http.StatusNotFound)
			return
		case http.StatusForbidden, http.StatusTooManyRequests:
			http.Error(w, "GitHub denied access or its rate limit was reached. Check repository permissions or try again later.", http.StatusServiceUnavailable)
			return
		}
	}
	http.Error(w, "Could not retrieve repository data from GitHub. Please try again.", http.StatusBadGateway)
}

func (a *authServer) repositories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	session, ok := a.currentSession(r)
	if !ok || session.token == "" {
		http.Error(w, "Sign in with GitHub to select your repositories.", http.StatusUnauthorized)
		return
	}
	page := 1
	if value := r.URL.Query().Get("page"); value != "" {
		var err error
		page, err = strconv.Atoi(value)
		if err != nil || page < 1 || page > 10000 {
			http.Error(w, "Invalid page", http.StatusBadRequest)
			return
		}
	}
	repositories := []Repository{}
	path := fmt.Sprintf("/user/repos?sort=updated&per_page=100&page=%d", page)
	if err := githubData(r.Context(), path, session.token, &repositories); err != nil {
		writeGitHubError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(repositories); err != nil {
		log.Printf("Failed to encode repositories: %v", err)
	}
}

func fetchRepository(ctx context.Context, name, token string) (RepositoryContext, error) {
	result := RepositoryContext{Repository: name, Files: []string{}}
	base := "/repos/" + name
	var repository Repository
	if err := githubData(ctx, base, token, &repository); err != nil {
		return result, err
	}
	if repository.DefaultBranch == "" {
		return result, fmt.Errorf("GitHub returned no default branch")
	}
	var tree struct {
		SHA       string `json:"sha"`
		Truncated bool   `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	err := githubData(ctx, base+"/git/trees/"+url.PathEscape(repository.DefaultBranch)+"?recursive=1", token, &tree)
	if errors.Is(err, githubStatusError(http.StatusConflict)) {
		return result, nil // GitHub reports an empty repository as 409.
	}
	if err != nil {
		return result, err
	}
	if tree.SHA == "" {
		return result, fmt.Errorf("GitHub returned no tree reference")
	}
	result.Ref, result.FilesTruncated = repository.DefaultBranch, tree.Truncated
	pathBytes := 0
	// shortcut: cap prompt context for coursework, add chunked analysis if larger repositories need complete coverage.
	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		if len(result.Files) >= 1000 || pathBytes+len(entry.Path) > 64<<10 {
			result.FilesTruncated = true
			break
		}
		result.Files = append(result.Files, entry.Path)
		pathBytes += len(entry.Path)
	}
	const readmeLimit = 64 << 10
	body, err := githubGet(ctx, base+"/readme?ref="+url.QueryEscape(result.Ref), token, "application/vnd.github.raw+json", readmeLimit)
	if errors.Is(err, githubStatusError(http.StatusNotFound)) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.ReadmeTruncated = len(body) > readmeLimit
	if result.ReadmeTruncated {
		body = body[:readmeLimit]
	}
	result.Readme = strings.ToValidUTF8(string(body), "")
	return result, nil
}
