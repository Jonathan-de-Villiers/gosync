# Changelog

All notable changes to GoSync will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.5] - 2024-05-07

### Fixed
- `gosync -u` now works standalone to check for updates

## [1.0.4] - 2024-05-07

### Added
- `discover` command to scan `~/.config` and add untracked configs as packages

### Changed
- Renamed `push` command to `sync` for clarity
- Compact status output showing only changed/new/missing files
- Improved status output with direction indicators and timestamps

## [1.0.3] - 2024-05-07

### Added
- Self-updater with `gosync update` command
- Smart exclusion patterns for `.DS_Store`, lock files, and backup directories

## [1.0.2] - 2024-05-07

### Added
- Colored output for status using `fatih/color` package
- Color-coded status indicators (yellow=sync, blue=pull, magenta=modified, green=in-sync)
- Intuitive direction arrows showing sync direction

### Changed
- Updated all repository references to `Jonathan-de-Villiers/gosync`
- Fixed install commands for cross-platform compatibility

## [1.0.1] - 2024-05-07

### Fixed
- CI workflow: Removed coverage profile causing covdata errors
- CI workflow: Fixed linter compatibility by using Go 1.21
- CI workflow: Restricted binary execution test to linux-amd64
- Release workflow: Updated artifact actions to v4
- Release workflow: Added `contents: write` permission

## [1.0.0] - 2024-05-07

### Added
- Initial release of GoSync
- Core sync functionality (pull/push/status)
- Package-based configuration management
- Automatic backups with timestamps
- Diff visualization
- Dry-run mode
- OS filtering for platform-specific packages
- Git integration
- Configuration management
- Multi-platform support (Linux, macOS, Windows)
- Installation scripts for bash and PowerShell
- Homebrew, Scoop, and Chocolatey support
- Comprehensive README documentation

[Unreleased]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.5...HEAD
[1.0.5]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/Jonathan-de-Villiers/gosync/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/Jonathan-de-Villiers/gosync/releases/tag/v1.0.0
