package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const maxQualityArchive = 32 << 20
const maxQualityExpanded = 128 << 20
const maxQualityFile = 512 << 10
const maxQualitySource = 24 << 20
const maxQualityFiles = 2000

type qualitySource struct {
	Files   map[string]int
	Skipped int
	Limited bool
}

func downloadQualitySource(ctx context.Context, repository, commit, token, dir string) (qualitySource, error) {
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+repository+"/tarball/"+commit, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		return qualitySource{}, fmt.Errorf("Could not download the GitHub snapshot. Retry the scan.")
	}
	if response.StatusCode == http.StatusFound || response.StatusCode == http.StatusTemporaryRedirect {
		location, locationErr := response.Location()
		response.Body.Close()
		if locationErr != nil || location.Scheme != "https" || location.Host != "codeload.github.com" || location.User != nil {
			return qualitySource{}, fmt.Errorf("GitHub returned an unexpected archive download location.")
		}
		// Private archive redirects are signed; never forward the OAuth header.
		req, _ = http.NewRequestWithContext(ctx, http.MethodGet, location.String(), nil)
		response, err = client.Do(req)
		if err != nil {
			return qualitySource{}, fmt.Errorf("Could not download the GitHub snapshot. Retry the scan.")
		}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return qualitySource{}, githubStatusError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxQualityArchive+1))
	if err != nil || len(body) > maxQualityArchive {
		return qualitySource{}, fmt.Errorf("The snapshot could not be downloaded within the 32 MiB archive limit. Try a smaller repository.")
	}
	return extractQualitySource(bytes.NewReader(body), dir)
}

func qualityPath(name string) bool {
	if name == "" || len(name) > 512 || path.Clean(name) != name || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\\x00\r\n") {
		return false
	}
	lower := strings.ToLower(name)
	for _, part := range strings.Split(lower, "/") {
		if part == ".." || strings.HasPrefix(part, ".env") || slices.Contains([]string{".git", ".qlty", "node_modules", "vendor", "dist", "build", "target", "coverage", ".next", "venv", ".venv", "__pycache__", "generated"}, part) {
			return false
		}
	}
	base := path.Base(lower)
	for _, prefix := range []string{"credentials.", "secrets.", "service-account.", "id_rsa", "id_ed25519"} {
		if strings.HasPrefix(base, prefix) {
			return false
		}
	}
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") {
		return false
	}
	return slices.Contains([]string{".go", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".py", ".rb", ".java", ".kt", ".kts", ".c", ".h", ".cpp", ".cc", ".hpp", ".cs", ".rs", ".php", ".swift", ".scala", ".ex", ".exs", ".erl", ".hrl", ".lua", ".vue", ".svelte"}, path.Ext(lower))
}

func extractQualitySource(reader io.Reader, dir string) (qualitySource, error) {
	result := qualitySource{Files: map[string]int{}}
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return result, fmt.Errorf("GitHub returned an invalid source archive.")
	}
	defer gz.Close()
	expanded := &io.LimitedReader{R: gz, N: maxQualityExpanded + 1}
	archive := tar.NewReader(expanded)
	root, total, entries := "", 0, 0
	for {
		header, err := archive.Next()
		if err == io.EOF {
			if expanded.N <= 0 {
				return result, fmt.Errorf("The expanded snapshot exceeds the 128 MiB limit.")
			}
			break
		}
		if err != nil {
			return result, fmt.Errorf("The source archive is incomplete or exceeds the expanded-size limit.")
		}
		entries++
		if entries > 20000 || header.Size > maxQualityExpanded {
			return result, fmt.Errorf("The snapshot is too large for this trial. Try a smaller repository.")
		}
		// GitHub tarballs include a global PAX header containing the commit ID.
		if header.Typeflag == tar.TypeDir || header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		prefix, name, ok := strings.Cut(header.Name, "/")
		if !ok || prefix == "" || prefix == ".." || prefix == "." || strings.HasPrefix(prefix, "/") || (root != "" && root != prefix) || path.Clean(name) != name || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\\x00") {
			return result, fmt.Errorf("The source archive contains an invalid path.")
		}
		root = prefix
		if header.Typeflag != tar.TypeReg || !qualityPath(name) {
			result.Skipped++
			continue
		}
		if header.Size > maxQualityFile {
			result.Skipped++
			continue
		}
		if len(result.Files) >= maxQualityFiles || int64(total)+header.Size > maxQualitySource {
			result.Skipped++
			result.Limited = true
			continue
		}
		body, err := io.ReadAll(io.LimitReader(archive, maxQualityFile+1))
		if err != nil || int64(len(body)) != header.Size {
			return result, fmt.Errorf("The source archive contains an incomplete file.")
		}
		if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
			result.Skipped++
			continue
		}
		if len(body) == 0 {
			result.Skipped++
			continue
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return result, fmt.Errorf("Could not prepare source files.")
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return result, fmt.Errorf("Could not prepare source files (duplicate archive paths are not allowed).")
		}
		_, writeErr := file.Write(body)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			return result, fmt.Errorf("Could not write the scan snapshot.")
		}
		lines := bytes.Count(body, []byte{'\n'})
		if body[len(body)-1] != '\n' {
			lines++
		}
		result.Files[name] = lines
		total += len(body)
	}
	return result, nil
}
