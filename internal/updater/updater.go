package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubAPIURL = "https://api.github.com/repos/Jonathan-de-Villiers/gosync/releases/latest"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckUpdate checks if a newer version is available
func CheckUpdate(currentVersion string) (*Release, bool, error) {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return nil, false, fmt.Errorf("cannot check updates for development builds")
	}

	release, err := getLatestRelease()
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	isUpdateAvailable := latestVersion != current
	return release, isUpdateAvailable, nil
}

// Update performs the self-update
func Update(release *Release) error {
	// Find the appropriate asset for this platform
	asset := findAssetForPlatform(release.Assets)
	if asset == nil {
		return fmt.Errorf("no suitable binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("Downloading %s...\n", asset.Name)

	// Download the binary
	tempFile, err := downloadBinary(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Make backup of current binary
	backupPath := execPath + ".backup"
	if err := copyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace current binary with new one
	if err := os.Rename(tempFile, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Make new binary executable
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	fmt.Println("✓ Update successful!")
	return nil
}

func getLatestRelease() (*Release, error) {
	resp, err := http.Get(githubAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func findAssetForPlatform(assets []Asset) *Asset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	for _, asset := range assets {
		if strings.Contains(asset.Name, goos) && strings.Contains(asset.Name, goarch) {
			return &asset
		}
	}

	return nil
}

func downloadBinary(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile := filepath.Join(os.TempDir(), "gosync-update")
	out, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return tempFile, nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
