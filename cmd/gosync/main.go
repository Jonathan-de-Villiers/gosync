package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gosync/internal/config"
	"gosync/internal/sync"
	"gosync/internal/updater"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gosync",
		Short: "Enhanced dotfiles management tool",
		Long: `GoSync is a powerful, distributable dotfiles management tool that provides
seamless synchronization of configuration files across multiple machines.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Global flags
	var configFile string
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.config/gosync/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolP("dry-run", "n", false, "show what would be done without making changes")

	// Pull command (sync from repo)
	var pullCmd = &cobra.Command{
		Use:   "pull [package|all]",
		Short: "Pull from repository to system (Repo -> $HOME)",
		Long: `Pull configuration files from the dotfiles repository to your home directory.
This syncs FROM the repository TO your system, updating files in $HOME.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			verbose, _ := cmd.Flags().GetBool("verbose")

			syncer := sync.New(cfg, dryRun, verbose)
			return syncer.Pull(args[0])
		},
	}

	// Sync command (sync to repo)
	var syncCmd = &cobra.Command{
		Use:   "sync [package|all]",
		Short: "Sync to repository ($HOME -> Repo)",
		Long: `Sync configuration files from your home directory to the dotfiles repository.
This syncs FROM your system TO the repository, updating files in the repo.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			verbose, _ := cmd.Flags().GetBool("verbose")

			syncer := sync.New(cfg, dryRun, verbose)
			return syncer.SyncToRepo(args[0])
		},
	}

	// Status command
	var statusCmd = &cobra.Command{
		Use:   "status [package|all]",
		Short: "Show sync status for packages",
		Long: `Display the current sync status, showing which files differ between
your home directory and the dotfiles repository.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			verbose, _ := cmd.Flags().GetBool("verbose")
			syncer := sync.New(cfg, false, verbose)
			return syncer.Status(args[0])
		},
	}

	// Packages command
	var packagesCmd = &cobra.Command{
		Use:   "packages",
		Short: "List available packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			packages, err := cfg.GetPackages()
			if err != nil {
				return fmt.Errorf("failed to get packages: %w", err)
			}

			fmt.Println("Available packages:")
			for _, pkg := range packages {
				fmt.Printf("  - %s\n", pkg)
			}
			return nil
		},
	}

	// Init command
	var initCmd = &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new dotfiles repository",
		Long: `Create a new dotfiles repository in the specified directory.
If no directory is specified, uses $HOME/dotfiles.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join(os.Getenv("HOME"), "dotfiles")
			if len(args) > 0 {
				dir = args[0]
			}

			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			return cfg.InitRepository(dir)
		},
	}

	// Config command
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	var showConfigCmd = &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Configuration file: %s\n", cfg.ConfigPath())
			fmt.Printf("Dotfiles directory: %s\n", cfg.GetDotfilesDir())
			fmt.Printf("Backup directory: %s\n", cfg.GetBackupDir())
			return nil
		},
	}

	configCmd.AddCommand(showConfigCmd)

	// Discover command
	var discoverCmd = &cobra.Command{
		Use:   "discover",
		Short: "Scan for and add untracked configs from ~/.config",
		Long: `Scan your ~/.config directory for configuration directories that are not
yet tracked in your dotfiles repository. This helps you discover and add
new packages to sync.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			syncer := sync.New(cfg, false, false)
			return syncer.Discover()
		},
	}

	// Update command
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Check for and install updates",
		Long: `Check for newer versions of gosync and optionally update to the latest release.
This command connects to GitHub to check for updates and can self-update the binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			release, hasUpdate, err := updater.CheckUpdate(version)
			if err != nil {
				return fmt.Errorf("update check failed: %w", err)
			}

			if !hasUpdate {
				fmt.Printf("✓ gosync is up to date (version %s)\n", version)
				return nil
			}

			fmt.Printf("Update available: %s → %s\n", version, release.TagName)

			// Auto-update flag
			autoUpdate, _ := cmd.Flags().GetBool("yes")
			if !autoUpdate {
				fmt.Print("Install update? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Update cancelled")
					return nil
				}
			}

			return updater.Update(release)
		},
	}
	updateCmd.Flags().BoolP("yes", "y", false, "Automatically install update without confirmation")

	// Check update flag on root command
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		checkFlag, _ := cmd.Flags().GetBool("check-update")
		if checkFlag {
			release, hasUpdate, err := updater.CheckUpdate(version)
			if err == nil && hasUpdate {
				fmt.Printf("⚠ Update available: %s → %s (run 'gosync update' to install)\n", version, release.TagName)
			}
		}
	}
	rootCmd.PersistentFlags().BoolP("check-update", "u", false, "Check for updates before running command")

	// Add commands to root
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(packagesCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(discoverCmd)
	rootCmd.AddCommand(updateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
