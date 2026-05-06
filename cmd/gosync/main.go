package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gosync/internal/config"
	"gosync/internal/sync"

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

	// Pull command
	var pullCmd = &cobra.Command{
		Use:   "pull [package|all]",
		Short: "Sync from repository to system (Repo -> $HOME)",
		Long: `Pull configuration files from the dotfiles repository to your home directory.
This will overwrite files in your home directory with versions from the repository.`,
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

	// Push command
	var pushCmd = &cobra.Command{
		Use:   "push [package|all]",
		Short: "Sync from system to repository ($HOME -> Repo)",
		Long: `Push configuration files from your home directory to the dotfiles repository.
This will update files in the repository with versions from your home directory.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			dryRun, _ := cmd.Flags().GetBool("dry-run")
			verbose, _ := cmd.Flags().GetBool("verbose")

			syncer := sync.New(cfg, dryRun, verbose)
			return syncer.Push(args[0])
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

	// Add commands to root
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(packagesCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
