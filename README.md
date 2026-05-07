# GoSync

A powerful, distributable dotfiles management tool written in Go that provides seamless synchronization of configuration files across multiple machines.

## Features

- **Cross-platform**: Works on Linux, macOS, and Windows
- **Package-based organization**: Organize configs by packages (nvim, alacritty, git, etc.)
- **Automatic backups**: Timestamped backups before any changes
- **Diff visualization**: See changes before applying them
- **Dry-run mode**: Preview changes without making them
- **OS filtering**: Automatically filter packages incompatible with your OS
- **Untracked detection**: Scan for new configs to add to your dotfiles
- **Self-updater**: Built-in update checker and installer
- **Colored output**: Easy-to-read status with color coding
- **Smart exclusions**: Auto-ignores .DS_Store, lock files, backups
- **Compact status**: Only shows files needing attention
- **Git integration**: Ready for remote repository sync (planned)
- **Template system**: Environment-specific configurations (planned)
- **Plugin architecture**: Extensible with custom sync logic (planned)

## Installation

### From Release (Recommended)

**Linux/macOS:**
```bash
curl -L https://github.com/Jonathan-de-Villiers/gosync/releases/latest/download/gosync-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz | tar xz
sudo mv gosync /usr/local/bin/
```

**Or use the install script:**
```bash
curl -sSL https://raw.githubusercontent.com/Jonathan-de-Villiers/gosync/main/scripts/install.sh | bash
```

### From Source

```bash
git clone https://github.com/Jonathan-de-Villiers/gosync.git
cd gosync
make install
```

### Build from Source

```bash
git clone https://github.com/Jonathan-de-Villiers/gosync.git
cd gosync
make build
```

The binary will be available in the `build/` directory.

## Quick Start

1. **Initialize a dotfiles repository** (if you don't have one):
   ```bash
   gosync init
   ```

2. **List available packages**:
   ```bash
   gosync packages
   ```

3. **Check status**:
   ```bash
   gosync status all
   ```

4. **Pull configurations** (repo → system):
   ```bash
   gosync pull all
   ```

5. **Sync changes** (system → repo):
   ```bash
   gosync sync all
   ```

## Usage

### Basic Commands

```bash
# Pull all configurations to your system
gosync pull all

# Pull only specific package
gosync pull nvim

# Sync your changes to the repository
gosync sync all

# Show status of all packages
gosync status all

# Show status of specific package
gosync status nvim

# List available packages
gosync packages

# Show configuration
gosync config show
```

### Discover Command

```bash
# Scan ~/.config for untracked configs and add them
gosync discover
```

This will scan your `~/.config` directory and offer to create packages for any
untracked configuration directories. It automatically:
- Skips system directories (Chrome, browsers, etc.)
- Detects already-tracked configs
- Creates the package structure in your dotfiles repo
- Copies the current config to the repo

### Update Commands

```bash
# Check for updates
gosync update

# Update without confirmation
gosync update -y

# Check for updates before running any command
gosync -u status all
```

### Advanced Options

```bash
# Dry run - see what would be done without making changes
gosync pull all --dry-run

# Verbose output
gosync pull all --verbose

# Show all files (including in-sync) in status
gosync status all --verbose

# Use custom config file
gosync pull all --config /path/to/config.yaml

# Disable confirmation prompts
gosync pull all --no-confirm
```

## Configuration

GoSync uses a YAML configuration file located at `~/.config/gosync/config.yaml`. 

### Default Configuration

```yaml
dotfiles_dir: "~/dotfiles"
backup_dir: "~/dotfiles-backups"

sync:
  exclude_patterns:
    - "*.tmp"
    - "*.log"
    - ".DS_Store"
  backup_enabled: true
  confirm_changes: true
  diff_command: "diff -u"

platform:
  os: "auto"  # auto, linux, darwin, windows
  os_filters:
    darwin: ["hyprland", "hyprlauncher", "waybar", "wofi"]
    linux: []

git:
  enabled: false
  remote: "origin"
  branch: "main"
  auto_commit: false
  auto_push: false

templates:
  enabled: false
  engine: "envsubst"
  variables: {}
  secret_files: []

plugins:
  enabled: false
  plugins: []
```

## Package Structure

Each package in your dotfiles repository maintains the same directory structure as in your home directory:

```
dotfiles/
├── nvim/
│   └── .config/
│       └── nvim/
│           ├── init.lua
│           └── lua/
├── alacritty/
│   └── .config/
│       └── alacritty/
│           └── alacritty.yml
└── git/
    ├── .gitconfig
    └── .gitignore
```

## Migration from Bash Script

If you're migrating from the original `sync-dotfiles.sh` script:

1. Your existing dotfiles structure is compatible
2. Run `gosync config show` to verify paths
3. Use `gosync status all` to check current state
4. Start with `--dry-run` to preview changes

## Development

### Building

```bash
# Development build
make dev

# Production build
make build

# Build for all platforms
make build-all
```

### Testing

```bash
make test
```

### Linting

```bash
make lint
```

### Formatting

```bash
make fmt
```

## Roadmap

- [x] Basic sync functionality (pull/push/status)
- [x] Configuration management
- [x] Backup system
- [x] Diff visualization
- [ ] Git integration
- [ ] Template system
- [ ] Plugin architecture
- [ ] Web UI dashboard
- [ ] REST API
- [ ] Encryption support
- [ ] Remote backup storage

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Acknowledgments

Inspired by the original `sync-dotfiles.sh` script and enhanced with modern Go architecture for better performance, reliability, and extensibility.
