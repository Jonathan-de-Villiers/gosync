package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DotfilesDir string `yaml:"dotfiles_dir" json:"dotfilesDir"`
	BackupDir   string `yaml:"backup_dir" json:"backupDir"`

	// Sync settings
	Sync SyncConfig `yaml:"sync" json:"sync"`

	// Platform settings
	Platform PlatformConfig `yaml:"platform" json:"platform"`

	// Git settings
	Git GitConfig `yaml:"git" json:"git"`

	// Templates
	Templates TemplateConfig `yaml:"templates" json:"templates"`

	// Plugins
	Plugins PluginConfig `yaml:"plugins" json:"plugins"`

	configPath string
}

type SyncConfig struct {
	ExcludePatterns []string `yaml:"exclude_patterns" json:"excludePatterns"`
	IncludePatterns []string `yaml:"include_patterns" json:"includePatterns"`
	BackupEnabled   bool     `yaml:"backup_enabled" json:"backupEnabled"`
	ConfirmChanges  bool     `yaml:"confirm_changes" json:"confirmChanges"`
	DiffCommand     string   `yaml:"diff_command" json:"diffCommand"`
}

type PlatformConfig struct {
	OS        string              `yaml:"os" json:"os"`
	OSFilters map[string][]string `yaml:"os_filters" json:"osFilters"`
	EnvVars   map[string]string   `yaml:"env_vars" json:"envVars"`
}

type GitConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Remote     string `yaml:"remote" json:"remote"`
	Branch     string `yaml:"branch" json:"branch"`
	AutoCommit bool   `yaml:"auto_commit" json:"autoCommit"`
	AutoPush   bool   `yaml:"auto_push" json:"autoPush"`
}

type TemplateConfig struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Engine      string            `yaml:"engine" json:"engine"`
	Variables   map[string]string `yaml:"variables" json:"variables"`
	SecretFiles []string          `yaml:"secret_files" json:"secretFiles"`
}

type PluginConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Plugins []string `yaml:"plugins" json:"plugins"`
}

// Load configuration from file or create default
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".config", "gosync", "config.yaml")
	}

	cfg.configPath = configPath

	// Load existing config if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Set defaults and validate
	if err := cfg.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	return cfg, nil
}

func (c *Config) setDefaults() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	if c.DotfilesDir == "" {
		c.DotfilesDir = filepath.Join(homeDir, "dotfiles")
	}

	if c.BackupDir == "" {
		c.BackupDir = filepath.Join(homeDir, "dotfiles-backups")
	}

	// Sync defaults
	if c.Sync.DiffCommand == "" {
		c.Sync.DiffCommand = "diff -u"
	}
	if c.Sync.BackupEnabled {
		c.Sync.BackupEnabled = true
	}
	if c.Sync.ConfirmChanges {
		c.Sync.ConfirmChanges = true
	}

	// Platform defaults
	if c.Platform.OS == "" {
		c.Platform.OS = runtime.GOOS
	}
	if c.Platform.OSFilters == nil {
		c.Platform.OSFilters = map[string][]string{
			"darwin": {"hyprland", "hyprlauncher", "waybar", "wofi"},
			"linux":  {},
		}
	}

	// Git defaults
	if c.Git.Branch == "" {
		c.Git.Branch = "main"
	}

	// Template defaults
	if c.Templates.Engine == "" {
		c.Templates.Engine = "envsubst"
	}

	return nil
}

// Save configuration to file
func (c *Config) Save() error {
	// Ensure config directory exists
	configDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get packages from dotfiles directory
func (c *Config) GetPackages() ([]string, error) {
	entries, err := os.ReadDir(c.DotfilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read dotfiles directory: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip special directories
		if name == ".git" || name == ".github" || name == "scripts" || name == "backups" || name == "dotfiles-backups" {
			continue
		}

		packages = append(packages, name)
	}

	return packages, nil
}

// Check if package is allowed for current OS
func (c *Config) IsPackageAllowed(pkg string) bool {
	excluded, exists := c.Platform.OSFilters[c.Platform.OS]
	if !exists {
		return true
	}

	for _, excludedPkg := range excluded {
		if excludedPkg == pkg {
			return false
		}
	}

	return true
}

// Initialize a new dotfiles repository
func (c *Config) InitRepository(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create dotfiles directory: %w", err)
	}

	// Create .gitignore
	gitignorePath := filepath.Join(dir, ".gitignore")
	gitignoreContent := `# Backup directories
backups/
dotfiles-backups/

# OS-specific files
.DS_Store
Thumbs.db

# Editor files
.vscode/
.idea/
*.swp
*.swo

# Temporary files
*.tmp
*.temp
`

	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README.md
	readmePath := filepath.Join(dir, "README.md")
	readmeContent := "# Dotfiles Repository\n\n" +
		"This is a dotfiles repository managed by GoSync.\n\n" +
		"## Structure\n\n" +
		"Each directory in this repository represents a package of configuration files:\n\n" +
		"- `nvim/` - Neovim configuration\n" +
		"- `alacritty/` - Alacritty terminal configuration\n" +
		"- `git/` - Git configuration\n" +
		"- ... and more\n\n" +
		"## Usage\n\n" +
		"```bash\n" +
		"# Pull all configurations to your system\n" +
		"gosync pull all\n\n" +
		"# Push your changes to the repository\n" +
		"gosync push all\n\n" +
		"# Pull only specific package\n" +
		"gosync pull nvim\n\n" +
		"# Show status\n" +
		"gosync status all\n" +
		"```\n\n" +
		"## Adding New Packages\n\n" +
		"1. Create a new directory for your package\n" +
		"2. Add configuration files maintaining the same directory structure as in $HOME\n" +
		"3. Use `gosync push <package>` to sync files to the repository\n\n" +
		"For more information, see the GoSync documentation.\n"

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Update config to use this directory
	c.DotfilesDir = dir
	return c.Save()
}

// Getters
func (c *Config) ConfigPath() string {
	return c.configPath
}

func (c *Config) GetDotfilesDir() string {
	return c.DotfilesDir
}

func (c *Config) GetBackupDir() string {
	return c.BackupDir
}
