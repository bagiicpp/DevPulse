package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type archiveFile struct {
	name, body string
	kind       byte
}

func qualityArchive(t *testing.T, entries []archiveFile) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, entry := range entries {
		kind := entry.kind
		if kind == 0 {
			kind = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0600, Typeflag: kind, Linkname: "../../outside.go"}
		if kind == tar.TypeXGlobalHeader {
			header = &tar.Header{Name: entry.name, Typeflag: kind, PAXRecords: map[string]string{"comment": "GitHub commit metadata"}}
		}
		if kind == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if kind == tar.TypeReg {
			if _, err := writer.Write([]byte(entry.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestQualitySourceExcludesConfigsLinksSecretsAndGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	entries := []archiveFile{{"snapshot/src/main.go", "package main\nfunc main() {}\n", 0}, {"snapshot/app.ts", "export const a = 1", 0}, {"snapshot/.qlty/qlty.toml", "malicious config", 0}, {"snapshot/.env.go", "secret", 0}, {"snapshot/secrets.ts", "secret", 0}, {"snapshot/node_modules/pkg/a.js", "dependency", 0}, {"snapshot/README.md", "docs", 0}, {"snapshot/link.go", "", tar.TypeSymlink}, {"snapshot/hard.go", "", tar.TypeLink}, {"snapshot/binary.go", "\x00bad", 0}, {"snapshot/empty.go", "", 0}, {"snapshot/large.go", strings.Repeat("x", maxQualityFile+1), 0}}
	entries = append([]archiveFile{{"pax_global_header", "", tar.TypeXGlobalHeader}}, entries...)
	result, err := extractQualitySource(bytes.NewReader(qualityArchive(t, entries)), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 2 || result.Files["src/main.go"] != 2 || result.Files["app.ts"] != 1 || result.Skipped != 10 {
		t.Fatalf("wrong extraction: %#v", result)
	}
	for _, name := range []string{".qlty", ".env.go", "secrets.ts", "link.go", "hard.go", "node_modules"} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("excluded file retained: %s", name)
		}
	}
}

func TestQualitySourceRejectsTraversalMalformedAndDuplicateArchives(t *testing.T) {
	for _, name := range []string{"snapshot/../outside.go", "/absolute.go", "snapshot/a/../../outside.go", "snapshot/a\\outside.go"} {
		if _, err := extractQualitySource(bytes.NewReader(qualityArchive(t, []archiveFile{{name, "package main", 0}})), t.TempDir()); err == nil {
			t.Fatalf("accepted path %q", name)
		}
	}
	for _, entries := range [][]archiveFile{
		{{"snapshot/a.go", "package main", 0}, {"snapshot/a.go", "overwrite", 0}},
		{{"first/a.go", "package main", 0}, {"other/b.go", "package main", 0}},
	} {
		if _, err := extractQualitySource(bytes.NewReader(qualityArchive(t, entries)), t.TempDir()); err == nil {
			t.Fatal("accepted duplicate or mixed archive roots")
		}
	}
	if _, err := extractQualitySource(strings.NewReader("not gzip"), t.TempDir()); err == nil {
		t.Fatal("malformed archive accepted")
	}
}

func TestQualitySourceDisclosesFileCountLimit(t *testing.T) {
	entries := make([]archiveFile, maxQualityFiles+1)
	for i := range entries {
		entries[i] = archiveFile{fmt.Sprintf("snapshot/file%d.go", i), "package main\n", 0}
	}
	result, err := extractQualitySource(bytes.NewReader(qualityArchive(t, entries)), t.TempDir())
	if err != nil || !result.Limited || len(result.Files) != maxQualityFiles || result.Skipped != 1 {
		t.Fatalf("limit not disclosed: %v %#v", err, result)
	}
}

const qualityMetrics = `{"stats":[{"path":"src/main.go","language":"LANGUAGE_GO","kind":"COMPONENT_TYPE_FILE","lines":2,"codeLines":2,"functions":1,"complexity":3,"cyclomatic":4}]}`
const qualitySmells = `{"version":"2.1.0","runs":[{"invocations":[{"executionSuccessful":true}],"results":[{"ruleId":"qlty:complexity","level":"warning","message":{"text":"Complex function"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/main.go"},"region":{"startLine":1,"endLine":2}}}]}]}]}`

func TestQualityDecodesMeasuredMetricsAndSourceLinkedFindings(t *testing.T) {
	report, err := decodeQuality([]byte(qualityMetrics), []byte(qualitySmells), map[string]int{"src/main.go": 2})
	if err != nil {
		t.Fatal(err)
	}
	if report.CodeLines != 2 || report.Functions != 1 || report.Complexity != 3 || report.Cyclomatic != 4 || report.TotalFindings != 1 || report.Findings[0].Locations[0].EndLine != 2 {
		t.Fatalf("wrong report: %#v", report)
	}
	for _, tc := range []struct{ path, uri string }{{"src/100%.go", "src/100%.go"}, {"src/a%20b.go", "src/a%20b.go"}, {"src/a b.go", "src/a%20b.go"}} {
		metrics := strings.ReplaceAll(qualityMetrics, "src/main.go", tc.path)
		smells := strings.ReplaceAll(qualitySmells, "src/main.go", tc.uri)
		decoded, err := decodeQuality([]byte(metrics), []byte(smells), map[string]int{tc.path: 2})
		if err != nil || decoded.Findings[0].Locations[0].Path != tc.path {
			t.Fatalf("source path %q was not preserved: %v", tc.path, err)
		}
	}
	withDirectory := strings.Replace(qualityMetrics, `}]}`, `},{"kind":"COMPONENT_TYPE_DIRECTORY","path":"src","codeLines":2,"complexity":3}]}`, 1)
	rolledUp, err := decodeQuality([]byte(withDirectory), []byte(qualitySmells), map[string]int{"src/main.go": 2})
	if err != nil || rolledUp.CodeLines != 2 || rolledUp.Complexity != 3 || len(rolledUp.Files) != 1 {
		t.Fatal("directory aggregates were counted twice")
	}
	exclusive := strings.Replace(qualitySmells, `"endLine":2`, `"endLine":3,"endColumn":1`, 1)
	endingAtEOF, err := decodeQuality([]byte(qualityMetrics), []byte(exclusive), map[string]int{"src/main.go": 2})
	if err != nil || endingAtEOF.Findings[0].Locations[0].EndLine != 2 {
		t.Fatal("exclusive SARIF end at EOF was not converted to the last included line")
	}
	duplicate := `{"version":"2.1.0","runs":[{"invocations":[{"executionSuccessful":true}],"results":[{"ruleId":"qlty:identical-code","level":"warning","message":{"text":"Duplicated code"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/main.go"},"region":{"startLine":1,"endLine":3}}}],"relatedLocations":[{"physicalLocation":{"artifactLocation":{"uri":"src/copy.go"},"region":{"startLine":1,"endLine":3}}}]}]}]}`
	duplicates, err := decodeQuality([]byte(qualityMetrics), []byte(duplicate), map[string]int{"src/main.go": 2, "src/copy.go": 2})
	if err != nil || len(duplicates.Findings[0].Locations) != 2 || duplicates.Findings[0].Locations[1].Path != "src/copy.go" || duplicates.Findings[0].Locations[1].EndLine != 2 {
		t.Fatal("duplication must link both copies and handle the terminal empty line")
	}
	for _, invalid := range []string{strings.Replace(qualityMetrics, "src/main.go", "unread.go", 1), strings.Replace(qualityMetrics, `"complexity":3`, `"complexity":-1`, 1), `{}`, `null`} {
		if _, err := decodeQuality([]byte(invalid), []byte(qualitySmells), map[string]int{"src/main.go": 2}); err == nil {
			t.Fatal("invalid metrics accepted")
		}
	}
	for _, invalid := range []string{strings.Replace(qualitySmells, `"endLine":2`, `"endLine":3`, 1), strings.Replace(qualitySmells, "src/main.go", "../outside.go", 1), strings.Replace(qualitySmells, `"executionSuccessful":true`, `"executionSuccessful":false`, 1), `{}`} {
		if _, err := decodeQuality([]byte(qualityMetrics), []byte(invalid), map[string]int{"src/main.go": 2}); err == nil {
			t.Fatal("invalid findings accepted")
		}
	}
	empty := `{"version":"2.1.0","runs":[{"invocations":[{"executionSuccessful":true}],"results":[]}]}`
	report, err = decodeQuality([]byte(qualityMetrics), []byte(empty), map[string]int{"src/main.go": 2})
	if err != nil || report.Findings == nil || report.TotalFindings != 0 {
		t.Fatal("zero findings must remain valid")
	}
}

func TestQualityArchiveRedirectDoesNotForwardToken(t *testing.T) {
	archive := qualityArchive(t, []archiveFile{{"snapshot/main.go", "package main\n", 0}})
	calls := 0
	mockGitHub(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host == "api.github.com" {
			if r.Header.Get("Authorization") != "Bearer github-secret" {
				t.Fatal("initial private download lacks authorization")
			}
			response := geminiReply(302, "")
			response.Header = make(http.Header)
			response.Header.Set("Location", "https://codeload.github.com/owner/repo/tar.gz/commit?token=signed")
			response.Request = r
			return response, nil
		}
		if r.URL.Host != "codeload.github.com" || r.Header.Get("Authorization") != "" {
			t.Fatal("archive token forwarded or unexpected host")
		}
		return geminiReply(200, string(archive)), nil
	})
	result, err := downloadQualitySource(context.Background(), "owner/repo", strings.Repeat("a", 40), "github-secret", t.TempDir())
	if err != nil || calls != 2 || len(result.Files) != 1 {
		t.Fatalf("archive failed: %v", err)
	}
	mockGitHub(t, func(r *http.Request) (*http.Response, error) {
		response := geminiReply(302, "")
		response.Header = make(http.Header)
		response.Header.Set("Location", "https://evil.example/archive")
		response.Request = r
		return response, nil
	})
	if _, err := downloadQualitySource(context.Background(), "o/r", strings.Repeat("a", 40), "github-secret", t.TempDir()); err == nil {
		t.Fatal("unsafe redirect accepted")
	}
}

func TestQualityRejectsRequestsAndMissingScannerBeforeRetrieval(t *testing.T) {
	t.Setenv("QLTY_BIN", "/missing/qlty")
	mockGitHub(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached network")
		return nil, nil
	})
	for _, tc := range []struct {
		method, body, origin string
		status               int
	}{
		{"GET", "", "", 405}, {"POST", `{`, "", 400}, {"POST", `{"repository":"https://github.com/o/r"}`, "", 400}, {"POST", strings.Repeat(" ", 4097), "", 413}, {"POST", `{"repository":"o/r"}`, "https://evil.example", 403}, {"POST", `{"repository":"o/r"}`, "", 503},
	} {
		_, mux := testAuth(t)
		request := httptest.NewRequest(tc.method, "/api/analysis", strings.NewReader(tc.body))
		request.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, request)
		if w.Code != tc.status {
			t.Fatalf("got %d want %d: %s", w.Code, tc.status, w.Body)
		}
	}
}

func TestQualityRealCLIAndAIIndependence(t *testing.T) {
	bin := os.Getenv("QLTY_TEST_BIN")
	if bin == "" {
		t.Skip("set QLTY_TEST_BIN to run the pinned CLI integration check")
	}
	t.Setenv("QLTY_BIN", bin)
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	archive := qualityArchive(t, []archiveFile{{"snapshot/src/main.go", "package main\nfunc main() {}\n", 0}, {"snapshot/src/parameters.ts", "export function many(a: number,b: number,c: number,d: number,e: number,f: number): number { return a+b+c+d+e+f; }\n", 0}, {"snapshot/.qlty/qlty.toml", "not valid config", 0}})
	mockGitHub(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.github.com" {
			t.Fatal("unexpected external or AI request")
		}
		if r.Header.Get("Authorization") != "Bearer github-secret" {
			t.Fatal("private GitHub token missing")
		}
		switch r.URL.Path {
		case "/repos/owner/repo":
			return geminiReply(200, `{"full_name":"owner/repo","default_branch":"main","private":true}`), nil
		case "/repos/owner/repo/commits/main":
			return geminiReply(200, `{"sha":"`+strings.Repeat("a", 40)+`"}`), nil
		case "/repos/owner/repo/tarball/" + strings.Repeat("a", 40):
			return geminiReply(200, string(archive)), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})
	a, mux := testAuth(t)
	a.sessions["valid"] = userSession{token: "github-secret", expires: time.Now().Add(time.Hour)}
	r := httptest.NewRequest("POST", "/api/analysis", strings.NewReader(`{"repository":"owner/repo"}`))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "valid"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("Qlty integration failed: %d %s", w.Code, w.Body)
	}
	var report QualityReport
	if json.Unmarshal(w.Body.Bytes(), &report) != nil || len(report.Files) != 2 || report.EngineVersion != qltyVersion || report.TotalFindings == 0 || report.CodeLines == 0 {
		t.Fatalf("incorrect real scan: %s", w.Body)
	}
	if strings.Contains(w.Body.String(), "github-secret") || report.InputFiles != 2 {
		t.Fatal("credential/config isolation failed")
	}
}

func TestQualitySubprocessDoesNotInheritSecretsAndIsBounded(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "secret-model-key")
	t.Setenv("SONAR_TOKEN", "secret-sonar-key")
	body, err := qualityCommand(context.Background(), t.TempDir(), "env")
	if err != nil || strings.Contains(string(body), "secret-") || !strings.Contains(string(body), "QLTY_TELEMETRY=off") {
		t.Fatalf("unsafe scanner environment: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := qualityCommand(ctx, t.TempDir(), "sleep", "10"); err != context.Canceled {
		t.Fatalf("cancellation not propagated: %v", err)
	}
	buffer := &limitedBuffer{limit: 3}
	if _, err := buffer.Write([]byte("four")); err == nil || buffer.Len() != 0 {
		t.Fatal("output limit ineffective")
	}
}

func TestQualityLiveRepositories(t *testing.T) {
	names := os.Getenv("QLTY_LIVE_REPOSITORIES")
	if names == "" {
		t.Skip("set QLTY_LIVE_REPOSITORIES, QLTY_LIVE_GITHUB_TOKEN and QLTY_TEST_BIN for a live scan")
	}
	token, bin := os.Getenv("QLTY_LIVE_GITHUB_TOKEN"), os.Getenv("QLTY_TEST_BIN")
	if token == "" || bin == "" {
		t.Fatal("live scan requires explicit GitHub credentials and the pinned Qlty binary")
	}
	t.Setenv("QLTY_BIN", bin)
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	for _, name := range strings.Split(names, ",") {
		t.Run(name, func(t *testing.T) {
			a, mux := testAuth(t)
			a.sessions["live-test"] = userSession{token: token, expires: time.Now().Add(time.Hour)}
			body, _ := json.Marshal(map[string]string{"repository": name})
			request := httptest.NewRequest("POST", "/api/analysis", bytes.NewReader(body))
			request.AddCookie(&http.Cookie{Name: sessionCookie, Value: "live-test"})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != 200 {
				t.Fatalf("live scan returned %d: %s", response.Code, response.Body)
			}
			var report QualityReport
			if json.Unmarshal(response.Body.Bytes(), &report) != nil || len(report.Files) == 0 || report.EngineVersion != qltyVersion {
				t.Fatal("invalid live report")
			}
			if strings.Contains(response.Body.String(), token) {
				t.Fatal("GitHub token leaked")
			}
			if dir := os.Getenv("QLTY_TEST_REPORT_DIR"); dir != "" {
				if err := os.WriteFile(filepath.Join(dir, strings.ReplaceAll(name, "/", "-")+".json"), response.Body.Bytes(), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Logf("%s @ %s: %d files with metrics, %d code lines, %d functions, complexity %d, %d findings (%.1fs)", report.Repository, report.Commit[:7], len(report.Files), report.CodeLines, report.Functions, report.Complexity, report.TotalFindings, report.DurationSeconds)
		})
	}
}
