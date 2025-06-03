package infrastructure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/franouer/proto-sync/internal/domain"
)

type GoModRepositoryImpl struct {
	logger domain.Logger
}

// NewGoModRepository creates a new Go module repository
func NewGoModRepository(logger domain.Logger) domain.GoModRepository {
	return &GoModRepositoryImpl{
		logger: logger,
	}
}

func (g *GoModRepositoryImpl) ParseProtobufLibraries(goModPath string) (*domain.GoModInfo, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod file at %s: %w", goModPath, err)
	}
	defer file.Close()

	g.logger.Info("Parsing protobuf libraries from %s...", goModPath)

	var repositories []domain.Repository
	foundComment := false
	scanner := bufio.NewScanner(file)

	// Regex to match replace directive
	replaceRegex := regexp.MustCompile(`^\s*replace\s+([^\s]+)\s+([^\s]+)\s*=>\s*([^\s]+)\s+([^\s]+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check if we found the protobuf libraries comment
		if strings.Contains(strings.ToLower(line), "// protobuf libraries") {
			foundComment = true
			continue
		}

		// If we found the comment, look for replace directives
		if foundComment {
			// Stop at empty lines or other comments
			if line == "" || (strings.HasPrefix(line, "//") && !strings.Contains(strings.ToLower(line), "protobuf")) {
				break
			}

			// Parse replace directive
			matches := replaceRegex.FindStringSubmatch(line)
			if len(matches) == 5 {
				remoteRepo := matches[3]
				remoteVersion := matches[4]

				// Extract just the repository path (remove github.com/ prefix if present)
				if strings.HasPrefix(remoteRepo, "github.com/") {
					remoteRepo = strings.TrimPrefix(remoteRepo, "github.com/")
				}

				repo := domain.Repository{
					Name:    fmt.Sprintf("github.com/%s", remoteRepo),
					Version: remoteVersion,
					URL:     fmt.Sprintf("https://github.com/%s", remoteRepo),
				}

				repositories = append(repositories, repo)
				g.logger.Info("Found protobuf library: %s@%s", repo.Name, repo.Version)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading go.mod file: %w", err)
	}

	if !foundComment {
		return nil, fmt.Errorf("could not find '// Protobuf libraries' comment in %s", goModPath)
	}

	if len(repositories) == 0 {
		g.logger.Warning("No protobuf libraries found after '// Protobuf libraries' comment")
	}

	return &domain.GoModInfo{
		Repositories: repositories,
	}, nil
}

func (g *GoModRepositoryImpl) GetLatestVersion(repo string) (string, error) {
	g.logger.Info("Checking latest version for %s...", repo)

	// Try using go list first
	cmd := exec.Command("go", "list", "-m", "-versions", repo)
	output, err := cmd.Output()
	if err == nil {
		versions := strings.Fields(string(output))
		if len(versions) > 1 {
			return versions[len(versions)-1], nil
		}
	}

	// Fallback: try to get latest from go proxy
	encodedRepo := strings.ReplaceAll(repo, "/", "%2F")
	proxyURL := fmt.Sprintf("https://proxy.golang.org/%s/@latest", encodedRepo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(proxyURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version for %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest version for %s: HTTP %d", repo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var versionInfo struct {
		Version string `json:"Version"`
	}

	if err := json.Unmarshal(body, &versionInfo); err != nil {
		return "", fmt.Errorf("failed to parse version response: %w", err)
	}

	if versionInfo.Version == "" {
		return "", fmt.Errorf("empty version returned for %s", repo)
	}

	return versionInfo.Version, nil
}

func (g *GoModRepositoryImpl) ListVersions(repo string) ([]string, error) {
	g.logger.Info("Listing available versions for %s...", repo)

	// Try using go list first
	cmd := exec.Command("go", "list", "-m", "-versions", repo)
	output, err := cmd.Output()
	if err == nil {
		versions := strings.Fields(string(output))
		if len(versions) > 1 {
			return versions[1:], nil // Skip the first element which is the module name
		}
	}

	// Fallback: try to get from go proxy
	encodedRepo := strings.ReplaceAll(repo, "/", "%2F")
	proxyURL := fmt.Sprintf("https://proxy.golang.org/%s/@v/list", encodedRepo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions for %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch versions for %s: HTTP %d", repo, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	versions := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(versions) == 1 && versions[0] == "" {
		return []string{}, nil
	}

	return versions, nil
}

func (g *GoModRepositoryImpl) DownloadModule(ctx context.Context, repo, version string) error {
	moduleWithVersion := fmt.Sprintf("%s@%s", repo, version)
	g.logger.Info("Downloading %s...", moduleWithVersion)

	cmd := exec.CommandContext(ctx, "go", "mod", "download", moduleWithVersion)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to download %s: %w\nOutput: %s", moduleWithVersion, err, string(output))
	}

	return nil
}

func (g *GoModRepositoryImpl) GetModulePath(repo, version string) (string, error) {
	// Get GOMODCACHE
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOMODCACHE: %w", err)
	}

	gomodcache := strings.TrimSpace(string(output))
	if gomodcache == "" {
		return "", fmt.Errorf("GOMODCACHE is empty")
	}

	moduleWithVersion := fmt.Sprintf("%s@%s", repo, version)
	modulePath := filepath.Join(gomodcache, moduleWithVersion)

	return modulePath, nil
}
