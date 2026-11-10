package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const qltyVersion = "0.668.0"

// shortcut: one scan at a time for this local trial, add a job queue if concurrent use is needed.
var qualitySlot = make(chan struct{}, 1)

type FileMetric struct {
	Path       string `json:"path"`
	Language   string `json:"language"`
	Lines      int    `json:"lines"`
	CodeLines  int    `json:"codeLines"`
	Functions  int    `json:"functions"`
	Complexity int    `json:"complexity"`
	Cyclomatic int    `json:"cyclomatic"`
}

type QualityLocation struct {
	Path      string `json:"path"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

type sarifLocation struct {
	Physical struct {
		Artifact struct {
			URI string `json:"uri"`
		} `json:"artifactLocation"`
		Region struct {
			Start     int `json:"startLine"`
			End       int `json:"endLine"`
			EndColumn int `json:"endColumn"`
		} `json:"region"`
	} `json:"physicalLocation"`
}

type QualityFinding struct {
	Rule      string            `json:"rule"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Locations []QualityLocation `json:"locations"`
}

type QualityReport struct {
	Repository      string           `json:"repository"`
	Branch          string           `json:"branch"`
	Commit          string           `json:"commit"`
	CompletedAt     string           `json:"completedAt"`
	EngineVersion   string           `json:"engineVersion"`
	DurationSeconds float64          `json:"durationSeconds"`
	InputFiles      int              `json:"inputFiles"`
	SkippedFiles    int              `json:"skippedFiles"`
	Limited         bool             `json:"limited"`
	CodeLines       int              `json:"codeLines"`
	Functions       int              `json:"functions"`
	Complexity      int              `json:"complexity"`
	Cyclomatic      int              `json:"cyclomatic"`
	Files           []FileMetric     `json:"files"`
	Findings        []QualityFinding `json:"findings"`
	TotalFindings   int              `json:"totalFindings"`
	Limitations     []string         `json:"limitations"`
}

func (a *authServer) qualityHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		http.Error(w, "Request must be at most 4 KiB", http.StatusRequestEntityTooLarge)
		return
	}
	var input struct {
		Repository string `json:"repository"`
	}
	if json.Unmarshal(body, &input) != nil || !repositoryName.MatchString(strings.TrimSpace(input.Repository)) {
		http.Error(w, "Use owner/repository", http.StatusBadRequest)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != a.origin {
		http.Error(w, "Invalid request origin", http.StatusForbidden)
		return
	}
	select {
	case qualitySlot <- struct{}{}:
		defer func() { <-qualitySlot }()
	default:
		http.Error(w, "Another quality scan is running. Retry when it finishes.", http.StatusTooManyRequests)
		return
	}
	bin := strings.TrimSpace(os.Getenv("QLTY_BIN"))
	if bin == "" {
		bin = "qlty"
	}
	if _, err := exec.LookPath(bin); err != nil {
		http.Error(w, "Qlty CLI is unavailable. Rebuild the backend image or set QLTY_BIN for local development.", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	session, _ := a.currentSession(r)
	name := strings.TrimSpace(input.Repository)
	var repository Repository
	if err := githubData(ctx, "/repos/"+name, session.token, &repository); err != nil {
		writeGitHubError(w, err)
		return
	}
	if repository.FullName != "" && repositoryName.MatchString(repository.FullName) {
		name = repository.FullName
	}
	if repository.DefaultBranch == "" {
		http.Error(w, "The repository has no default branch to scan.", http.StatusUnprocessableEntity)
		return
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := githubData(ctx, "/repos/"+name+"/commits/"+url.PathEscape(repository.DefaultBranch), session.token, &commit); err != nil {
		writeGitHubError(w, err)
		return
	}
	if len(commit.SHA) != 40 || strings.Trim(commit.SHA, "0123456789abcdef") != "" {
		http.Error(w, "GitHub returned an invalid commit reference.", http.StatusBadGateway)
		return
	}
	dir, err := os.MkdirTemp("", "devpulse-quality-")
	if err != nil {
		http.Error(w, "Could not prepare a scan workspace.", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(dir)
	started := time.Now()
	snapshot, err := downloadQualitySource(ctx, name, commit.SHA, session.token, dir)
	if err != nil {
		var status githubStatusError
		if errors.As(err, &status) {
			writeGitHubError(w, err)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	if len(snapshot.Files) == 0 {
		http.Error(w, "No supported source files were found within the scan limits. Choose another repository.", http.StatusUnprocessableEntity)
		return
	}
	report, err := scanQuality(ctx, bin, dir, snapshot.Files)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			http.Error(w, "The quality scan timed out. Try a smaller repository.", http.StatusGatewayTimeout)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	report.Repository, report.Branch, report.Commit = name, repository.DefaultBranch, commit.SHA
	report.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	report.DurationSeconds = time.Since(started).Seconds()
	report.InputFiles, report.SkippedFiles, report.Limited = len(snapshot.Files), snapshot.Skipped, snapshot.Limited
	report.Limitations = []string{
		"Experimental Qlty trial: built-in metrics and structural/duplication rules only. No linter plugins, AI, build scripts, dependency installation, or tests were run.",
		"Source files are scanned from a single GitHub commit. Repository Qlty configuration and ignore files are not used; DevPulse supplies a controlled configuration.",
		"Generated/dependency directories, secret-like paths, links, unsupported formats, and files over 512 KiB are excluded. Limits: 2,000 source files and 24 MiB of source text.",
		"Code-smell checks vary by language. No findings means no reported findings under this policy, not verified correctness or a passing quality gate. Test coverage is not measured.",
	}
	if snapshot.Limited {
		report.Limitations = append(report.Limitations, "The source-size or file-count limit was reached. This is a partial scan of the repository.")
	}
	if len(report.Files) < report.InputFiles {
		report.Limitations = append(report.Limitations, fmt.Sprintf("Qlty produced file metrics for %d of %d supplied files; unsupported, ignored, or unparseable files are not included in metric totals.", len(report.Files), report.InputFiles))
	}
	if len(report.Findings) < report.TotalFindings {
		report.Limitations = append(report.Limitations, "Only the first 200 findings are displayed. Total findings includes all validated findings returned by the scanner.")
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(report)
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		return 0, fmt.Errorf("scanner output exceeded its limit")
	}
	return b.Buffer.Write(p)
}

func qualityCommand(ctx context.Context, dir, executable string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	// Preserve the existing home location; never pass backend credentials to children.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "QLTY_TELEMETRY=off", "QLTY_LOG=off", "NO_COLOR=1", "RAYON_NUM_THREADS=2", "TMPDIR=" + dir, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0"}
	stdout, stderr := &limitedBuffer{limit: 4 << 20}, &limitedBuffer{limit: 64 << 10}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("Qlty could not complete the scan. Check the pinned CLI installation or try a smaller repository.")
	}
	return stdout.Bytes(), nil
}

func scanQuality(ctx context.Context, bin, dir string, files map[string]int) (QualityReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	version, err := qualityCommand(ctx, dir, bin, "--version")
	if err != nil {
		return QualityReport{}, err
	}
	fields := strings.Fields(string(version))
	if len(fields) < 2 || fields[0] != "qlty" || fields[1] != qltyVersion {
		return QualityReport{}, fmt.Errorf("This trial requires Qlty CLI %s. Rebuild the backend image.", qltyVersion)
	}
	if _, err := qualityCommand(ctx, dir, "git", "-c", "init.defaultBranch=main", "init", "--template=", "--quiet", "."); err != nil {
		return QualityReport{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, ".qlty"), 0700); err != nil {
		return QualityReport{}, fmt.Errorf("Could not prepare the scanner configuration.")
	}
	if err := os.WriteFile(filepath.Join(dir, ".qlty", "qlty.toml"), []byte("config_version = \"0\"\n"), 0600); err != nil {
		return QualityReport{}, fmt.Errorf("Could not prepare the scanner configuration.")
	}
	metrics, err := qualityCommand(ctx, dir, bin, "metrics", "--all", "--json", "--quiet", "--no-upgrade-check")
	if err != nil {
		return QualityReport{}, err
	}
	smells, err := qualityCommand(ctx, dir, bin, "smells", "--all", "--include-tests", "--sarif", "--no-snippets", "--quiet", "--no-upgrade-check")
	if err != nil {
		return QualityReport{}, err
	}
	return decodeQuality(metrics, smells, files)
}

func decodeQuality(metrics, smells []byte, files map[string]int) (QualityReport, error) {
	report := QualityReport{EngineVersion: qltyVersion, Files: []FileMetric{}, Findings: []QualityFinding{}}
	var raw struct {
		Stats []struct {
			FileMetric
			Kind string `json:"kind"`
		} `json:"stats"`
	}
	if json.Unmarshal(metrics, &raw) != nil || raw.Stats == nil {
		return report, fmt.Errorf("Qlty returned invalid file metrics. Retry the scan.")
	}
	seen := map[string]bool{}
	for _, stat := range raw.Stats {
		// Qlty also emits directory rollups; summing those would count files twice.
		if stat.Kind == "COMPONENT_TYPE_DIRECTORY" {
			continue
		}
		if stat.Kind != "COMPONENT_TYPE_FILE" || files[stat.Path] == 0 || seen[stat.Path] || stat.CodeLines < 0 || stat.Lines < 0 || stat.Functions < 0 || stat.Complexity < 0 || stat.Cyclomatic < 0 || stat.CodeLines > files[stat.Path] {
			return report, fmt.Errorf("Qlty returned unsupported file metrics. Retry the scan.")
		}
		seen[stat.Path] = true
		report.Files = append(report.Files, stat.FileMetric)
		report.CodeLines += stat.CodeLines
		report.Functions += stat.Functions
		report.Complexity += stat.Complexity
		report.Cyclomatic += stat.Cyclomatic
	}
	if len(report.Files) == 0 {
		return report, fmt.Errorf("Qlty could not analyze any of the supplied files. The repository may use unsupported languages.")
	}
	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Invocations []struct {
				Success bool `json:"executionSuccessful"`
			} `json:"invocations"`
			Results []struct {
				Rule    string `json:"ruleId"`
				Level   string `json:"level"`
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations        []sarifLocation `json:"locations"`
				RelatedLocations []sarifLocation `json:"relatedLocations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if json.Unmarshal(smells, &sarif) != nil || sarif.Version != "2.1.0" || len(sarif.Runs) != 1 {
		return report, fmt.Errorf("Qlty returned invalid findings. Retry the scan.")
	}
	run := sarif.Runs[0]
	if len(run.Invocations) == 0 || !run.Invocations[0].Success {
		return report, fmt.Errorf("Qlty did not complete its code-smell analysis. Retry the scan.")
	}
	for _, item := range run.Results {
		if strings.TrimSpace(item.Rule) == "" || strings.TrimSpace(item.Message.Text) == "" || len(item.Locations) == 0 {
			return report, fmt.Errorf("Qlty returned an incomplete finding. Retry the scan.")
		}
		finding := QualityFinding{Rule: item.Rule, Level: item.Level, Message: item.Message.Text, Locations: []QualityLocation{}}
		for _, location := range append(item.Locations, item.RelatedLocations...) {
			path := location.Physical.Artifact.URI
			var err error
			if files[path] == 0 {
				path, err = url.PathUnescape(path)
			}
			start, end := location.Physical.Region.Start, location.Physical.Region.End
			if end == 0 {
				end = start
			} else if location.Physical.Region.EndColumn == 1 && end > start {
				// SARIF end positions are exclusive; column 1 is on the following line.
				end--
			}
			if (item.Rule == "qlty:identical-code" || item.Rule == "qlty:similar-code") && end == files[path]+1 && location.Physical.Region.EndColumn == 0 {
				// Qlty duplication ranges can include the terminal empty line at EOF.
				end--
			}
			if err != nil || files[path] == 0 || start < 1 || end < start || end > files[path] {
				return report, fmt.Errorf("Qlty returned an unsupported source reference at %q:%d-%d (file has %d lines).", path, start, end, files[path])
			}
			finding.Locations = append(finding.Locations, QualityLocation{path, start, end})
		}
		report.TotalFindings++
		if len(report.Findings) < 200 {
			report.Findings = append(report.Findings, finding)
		}
	}
	sort.Slice(report.Files, func(i, j int) bool {
		if report.Files[i].Complexity == report.Files[j].Complexity {
			return report.Files[i].Path < report.Files[j].Path
		}
		return report.Files[i].Complexity > report.Files[j].Complexity
	})
	return report, nil
}
