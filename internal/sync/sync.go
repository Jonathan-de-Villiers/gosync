package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gosync/internal/config"
	"gosync/internal/diff"
)

type Syncer struct {
	config     *config.Config
	dryRun     bool
	verbose    bool
	diffEngine *diff.DiffEngine
}

type SyncMode string

const (
	PullMode SyncMode = "pull"
	PushMode SyncMode = "push"
)

type FileInfo struct {
	Path    string
	RelPath string
	IsDir   bool
	ModTime time.Time
	Size    int64
	Exists  bool
}

type SyncResult struct {
	Path      string
	Action    string
	Success   bool
	Error     error
	Backup    string
	DiffShown bool
}

func New(cfg *config.Config, dryRun, verbose bool) *Syncer {
	return &Syncer{
		config:     cfg,
		dryRun:     dryRun,
		verbose:    verbose,
		diffEngine: diff.New(cfg.Sync.DiffCommand),
	}
}

func (s *Syncer) Pull(target string) error {
	return s.sync(PullMode, target)
}

func (s *Syncer) Push(target string) error {
	return s.sync(PushMode, target)
}

func (s *Syncer) Status(target string) error {
	return s.status(target)
}

func (s *Syncer) sync(mode SyncMode, target string) error {
	if target == "all" {
		packages, err := s.config.GetPackages()
		if err != nil {
			return fmt.Errorf("failed to get packages: %w", err)
		}

		for _, pkg := range packages {
			if !s.config.IsPackageAllowed(pkg) {
				if s.verbose {
					fmt.Printf("Skipping '%s' on %s\n", pkg, s.config.Platform.OS)
				}
				continue
			}

			if err := s.syncPackage(mode, pkg); err != nil {
				return fmt.Errorf("failed to sync package '%s': %w", pkg, err)
			}
		}

		// Scan for untracked configs during push all
		if mode == PushMode {
			if err := s.scanUntracked(); err != nil {
				fmt.Printf("Warning: failed to scan for untracked configs: %v\n", err)
			}
		}
	} else {
		if !s.config.IsPackageAllowed(target) {
			return fmt.Errorf("package '%s' is not allowed on %s", target, s.config.Platform.OS)
		}

		if err := s.syncPackage(mode, target); err != nil {
			return fmt.Errorf("failed to sync package '%s': %w", target, err)
		}
	}

	return nil
}

func (s *Syncer) syncPackage(mode SyncMode, pkg string) error {
	pkgDir := filepath.Join(s.config.DotfilesDir, pkg)

	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		return fmt.Errorf("package '%s' does not exist", pkg)
	}

	files, err := s.getPackageFiles(pkg)
	if err != nil {
		return fmt.Errorf("failed to get package files: %w", err)
	}

	if len(files) == 0 {
		if s.verbose {
			fmt.Printf("No files to sync in package '%s'\n", pkg)
		}
		return nil
	}

	fmt.Printf("Syncing package '%s' (%s mode)\n", pkg, mode)

	var results []SyncResult
	for _, file := range files {
		result, err := s.syncFile(mode, pkg, file)
		if err != nil {
			return fmt.Errorf("failed to sync file '%s': %w", file, err)
		}
		results = append(results, result)
	}

	// Print summary
	s.printResults(results)
	return nil
}

func (s *Syncer) syncFile(mode SyncMode, pkg, relPath string) (SyncResult, error) {
	result := SyncResult{
		Path:    relPath,
		Success: true,
	}

	var src, dest string
	if mode == PullMode {
		src = filepath.Join(s.config.DotfilesDir, pkg, relPath)
		dest = filepath.Join(os.Getenv("HOME"), relPath)
	} else {
		src = filepath.Join(os.Getenv("HOME"), relPath)
		dest = filepath.Join(s.config.DotfilesDir, pkg, relPath)
	}

	// Check if source exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			result.Action = "skipped (source missing)"
			return result, nil
		}
		result.Success = false
		result.Error = err
		return result, err
	}

	// Get destination info
	_, err = os.Stat(dest)
	destExists := err == nil

	// Check if files are different
	if destExists && !s.hasChanges(src, dest) {
		result.Action = "unchanged"
		return result, nil
	}

	// Show diff if destination exists
	if destExists {
		if err := s.showDiff(src, dest, relPath); err != nil && s.verbose {
			fmt.Printf("Warning: failed to show diff: %v\n", err)
		}
		result.DiffShown = true
	} else {
		fmt.Printf("New file: %s\n", relPath)
	}

	// Create backup if enabled and destination exists
	if s.config.Sync.BackupEnabled && destExists {
		backupPath, err := s.createBackup(dest)
		if err != nil {
			result.Success = false
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result, err
		}
		result.Backup = backupPath
	}

	// Confirm changes if enabled
	if s.config.Sync.ConfirmChanges && !s.dryRun {
		if !s.confirmSync(fmt.Sprintf("Apply changes to %s?", relPath)) {
			result.Action = "skipped (user declined)"
			return result, nil
		}
	}

	// Perform the sync
	if s.dryRun {
		result.Action = "dry run"
		if s.verbose {
			fmt.Printf("DRY RUN: Would sync %s -> %s\n", src, dest)
		}
	} else {
		if err := s.copyFile(src, dest, srcInfo.IsDir()); err != nil {
			result.Success = false
			result.Error = err
			return result, err
		}
		result.Action = "synced"
	}

	return result, nil
}

func (s *Syncer) getPackageFiles(pkg string) ([]string, error) {
	pkgDir := filepath.Join(s.config.DotfilesDir, pkg)
	var files []string

	err := filepath.WalkDir(pkgDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the package directory itself
		if path == pkgDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(pkgDir, path)
		if err != nil {
			return err
		}

		// Skip excluded patterns
		for _, pattern := range s.config.Sync.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, relPath); matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Only include files that match include patterns (if any specified)
		if len(s.config.Sync.IncludePatterns) > 0 {
			included := false
			for _, pattern := range s.config.Sync.IncludePatterns {
				if matched, _ := filepath.Match(pattern, relPath); matched {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

func (s *Syncer) hasChanges(src, dest string) bool {
	srcInfo, err1 := os.Stat(src)
	destInfo, err2 := os.Stat(dest)

	if err1 != nil || err2 != nil {
		return true
	}

	// Check modification time and size
	if !srcInfo.ModTime().Equal(destInfo.ModTime()) || srcInfo.Size() != destInfo.Size() {
		return true
	}

	// For files, do a content comparison
	if !srcInfo.IsDir() {
		return s.diffEngine.HasChanges(src, dest)
	}

	return false
}

func (s *Syncer) showDiff(src, dest, relPath string) error {
	fmt.Printf("\nChanges in %s:\n", relPath)
	return s.diffEngine.ShowDiff(src, dest)
}

func (s *Syncer) createBackup(file string) (string, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupDir := filepath.Join(s.config.BackupDir, timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	relPath, err := filepath.Rel(os.Getenv("HOME"), file)
	if err != nil {
		return "", err
	}

	backupPath := filepath.Join(backupDir, relPath)
	backupDirPath := filepath.Dir(backupPath)

	if err := os.MkdirAll(backupDirPath, 0755); err != nil {
		return "", err
	}

	if err := s.copyFile(file, backupPath, false); err != nil {
		return "", err
	}

	return backupPath, nil
}

func (s *Syncer) copyFile(src, dest string, isDir bool) error {
	if isDir {
		return os.MkdirAll(dest, 0755)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Copy file content
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(srcFile)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dest, srcInfo.Mode())
}

func (s *Syncer) confirmSync(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func (s *Syncer) printResults(results []SyncResult) {
	var synced, skipped, failed int

	for _, result := range results {
		switch result.Action {
		case "synced":
			synced++
			fmt.Printf("✓ %s\n", result.Path)
		case "unchanged":
			skipped++
			if s.verbose {
				fmt.Printf("- %s (unchanged)\n", result.Path)
			}
		case "skipped", "dry run":
			skipped++
			if s.verbose {
				fmt.Printf("- %s (%s)\n", result.Path, result.Action)
			}
		default:
			failed++
			fmt.Printf("✗ %s: %v\n", result.Path, result.Error)
		}

		if result.Backup != "" && s.verbose {
			fmt.Printf("  Backup: %s\n", result.Backup)
		}
	}

	fmt.Printf("\nSummary: %d synced, %d skipped", synced, skipped)
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}

func (s *Syncer) status(target string) error {
	if target == "all" {
		packages, err := s.config.GetPackages()
		if err != nil {
			return fmt.Errorf("failed to get packages: %w", err)
		}

		for _, pkg := range packages {
			if !s.config.IsPackageAllowed(pkg) {
				continue
			}

			if err := s.statusPackage(pkg); err != nil {
				return fmt.Errorf("failed to get status for package '%s': %w", pkg, err)
			}
		}
	} else {
		if !s.config.IsPackageAllowed(target) {
			return fmt.Errorf("package '%s' is not allowed on %s", target, s.config.Platform.OS)
		}

		if err := s.statusPackage(target); err != nil {
			return fmt.Errorf("failed to get status for package '%s': %w", target, err)
		}
	}

	return nil
}

func (s *Syncer) statusPackage(pkg string) error {
	fmt.Printf("\nStatus for package '%s':\n", pkg)

	files, err := s.getPackageFiles(pkg)
	if err != nil {
		return err
	}

	var changes, newFiles, missing int
	for _, file := range files {
		repoPath := filepath.Join(s.config.DotfilesDir, pkg, file)
		homePath := filepath.Join(os.Getenv("HOME"), file)

		repoExists, err := s.fileExists(repoPath)
		if err != nil {
			return err
		}

		homeExists, err := s.fileExists(homePath)
		if err != nil {
			return err
		}

		if !repoExists && homeExists {
			newFiles++
			fmt.Printf("  + %s (new in home)\n", file)
		} else if repoExists && !homeExists {
			missing++
			fmt.Printf("  - %s (missing in home)\n", file)
		} else if repoExists && homeExists && s.hasChanges(repoPath, homePath) {
			changes++
			fmt.Printf("  Δ %s (modified)\n", file)
		} else if s.verbose {
			fmt.Printf("  = %s (in sync)\n", file)
		}
	}

	fmt.Printf("Summary: %d changed, %d new, %d missing\n", changes, newFiles, missing)
	return nil
}

func (s *Syncer) fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *Syncer) scanUntracked() error {
	fmt.Println("\nScanning for untracked configurations...")

	configDir := filepath.Join(os.Getenv("HOME"), ".config")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("failed to read .config directory: %w", err)
	}

	packages, err := s.config.GetPackages()
	if err != nil {
		return fmt.Errorf("failed to get packages: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip large browser and system directories
		skipDirs := []string{"google-chrome", "Brave-Browser", "BraveSoftware", "mozilla", "systemd"}
		shouldSkip := false
		for _, skip := range skipDirs {
			if name == skip {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Check if already tracked
		tracked := false
		for _, pkg := range packages {
			pkgConfigPath := filepath.Join(s.config.DotfilesDir, pkg, ".config", name)
			if _, err := os.Stat(pkgConfigPath); err == nil {
				tracked = true
				break
			}
		}

		if !tracked {
			fmt.Printf("? Untracked config: ~/.config/%s\n", name)
			if s.confirmSync(fmt.Sprintf("Add '%s' to dotfiles?", name)) {
				pkgName := name
				if s.confirmSync(fmt.Sprintf("Use '%s' as package name? (enter for different name)", name)) {
					fmt.Scanln(&pkgName)
					if pkgName == "" {
						pkgName = name
					}
				}

				newPkgDir := filepath.Join(s.config.DotfilesDir, pkgName, ".config")
				if err := os.MkdirAll(newPkgDir, 0755); err != nil {
					fmt.Printf("Failed to create package directory: %v\n", err)
					continue
				}

				srcPath := filepath.Join(configDir, name)
				destPath := filepath.Join(newPkgDir, name)

				if err := s.copyFile(srcPath, destPath, true); err != nil {
					fmt.Printf("Failed to copy config: %v\n", err)
					continue
				}

				fmt.Printf("✓ Added %s to package %s\n", name, pkgName)
			}
		}
	}

	return nil
}
