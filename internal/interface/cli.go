package interfaces

import (
	"context"
	"fmt"
	"os"

	"github.com/Francouer/proto-sync/internal/domain"
	"github.com/spf13/cobra"
)

type CLIHandler struct {
	service domain.ProtoSyncService
	logger  domain.Logger
}

// NewCLIHandler creates a new CLI handler
func NewCLIHandler(service domain.ProtoSyncService, logger domain.Logger) *CLIHandler {
	return &CLIHandler{
		service: service,
		logger:  logger,
	}
}

// CreateRootCommand creates the root cobra command
func (c *CLIHandler) CreateRootCommand() *cobra.Command {
	var config domain.SyncConfig

	rootCmd := &cobra.Command{
		Use:   "proto-sync",
		Short: "A flexible CLI tool to download and update proto files from remote repositories",
		Long: `Proto Sync automatically detects protobuf libraries from go.mod or allows manual specification.
It downloads specific versions and copies proto files to local directories with colorful logging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleSync(cmd.Context(), &config)
		},
	}

	// Add flags
	c.addFlags(rootCmd, &config)

	// Add subcommands
	rootCmd.AddCommand(c.createListVersionsCommand(&config))

	return rootCmd
}

func (c *CLIHandler) addFlags(cmd *cobra.Command, config *domain.SyncConfig) {
	// Set default values from environment variables or defaults
	defaultRepo := os.Getenv("REPO_NAME")
	defaultSourcePath := getEnvOrDefault("SOURCE_PATH_IN_REPO", "schemas/api/v1")
	defaultBufYaml := getEnvOrDefault("BUF_YAML_PATH", "buf.yaml")
	defaultGoMod := getEnvOrDefault("GO_MOD_PATH", "../go.mod")
	defaultProtoFile := os.Getenv("PROTO_FILE_NAME")

	cmd.Flags().StringVarP(&config.SpecifiedVersion, "version", "v", "", "Specify version to download (default: auto-detect from go.mod)")
	cmd.Flags().StringVarP(&defaultRepo, "repo", "r", defaultRepo, "Repository name (default: auto-detect from go.mod)")
	cmd.Flags().StringVarP(&config.SourcePath, "source", "s", defaultSourcePath, "Source path in repository")
	cmd.Flags().StringVarP(&config.BufYamlPath, "buf-yaml", "b", defaultBufYaml, "Path to buf.yaml file")
	cmd.Flags().StringVarP(&config.GoModPath, "go-mod", "g", defaultGoMod, "Path to go.mod file")
	cmd.Flags().StringVarP(&config.SpecificFile, "proto-file", "f", defaultProtoFile, "Download only specific proto file")
	cmd.Flags().BoolVarP(&config.DryRun, "dry-run", "d", false, "Show what would be done without executing")
	cmd.Flags().BoolVar(&config.SingleRepo, "single-repo", false, "Process only the first repository found")

	// Handle repository parsing after flags are parsed
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if defaultRepo != "" {
			repo := domain.Repository{
				Name: defaultRepo,
				URL:  fmt.Sprintf("https://%s", defaultRepo),
			}
			config.Repositories = []domain.Repository{repo}
		}
		return nil
	}
}

func (c *CLIHandler) createListVersionsCommand(config *domain.SyncConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "list-versions",
		Short: "List available versions for all repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.handleListVersions(cmd.Context(), config)
		},
	}
}

func (c *CLIHandler) handleSync(ctx context.Context, config *domain.SyncConfig) error {
	// Validate that required tools are available
	if err := c.validateRequiredTools(); err != nil {
		return err
	}

	results, err := c.service.Sync(ctx, config)
	if err != nil {
		c.logger.Error("Sync failed: %v", err)
		return err
	}

	if config.DryRun {
		return nil
	}

	// Print summary
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	c.logger.Info("Sync completed: %d/%d repositories processed successfully", successCount, len(results))
	return nil
}

func (c *CLIHandler) handleListVersions(ctx context.Context, config *domain.SyncConfig) error {
	if err := c.validateRequiredTools(); err != nil {
		return err
	}

	// Get repositories from go.mod if not specified
	repositories := config.Repositories
	if len(repositories) == 0 {
		if err := c.service.ValidateConfig(config); err != nil {
			return fmt.Errorf("invalid configuration: %w", err)
		}

		// We need to parse go.mod to get repositories
		// For now, we'll return an error asking user to specify repositories
		return fmt.Errorf("no repositories specified. Use --repo flag or ensure go.mod has '// Protobuf libraries' section")
	}

	versions, err := c.service.ListVersions(ctx, repositories)
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	// Print versions
	for repo, versionList := range versions {
		fmt.Printf("--- Versions for %s ---\n", repo)
		for _, version := range versionList {
			fmt.Println(version)
		}
		fmt.Println()
	}

	return nil
}

func (c *CLIHandler) validateRequiredTools() error {
	// Check if go is available
	if !c.isCommandAvailable("go") {
		return fmt.Errorf("go is required but not installed")
	}
	return nil
}

func (c *CLIHandler) isCommandAvailable(command string) bool {
	// This is a simple check - in a real implementation you might want to use exec.LookPath
	return true // Assume tools are available for now
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ShowUsage prints detailed usage information
func (c *CLIHandler) ShowUsage() {
	usage := `Usage: proto-sync [OPTIONS]

A flexible script to download and update proto files from remote repositories.
Automatically detects protobuf libraries from go.mod using // Protobuf libraries comment or allows manual specification.

Options:
    -h, --help              Show this help message
    -v, --version VERSION   Specify version to download (default: auto-detect from go.mod)
    -r, --repo REPO         Repository name (default: auto-detect from go.mod)
    -s, --source PATH       Source path in repository (default: schemas/api/v1)
    -b, --buf-yaml PATH     Path to buf.yaml file (default: buf.yaml)
    -g, --go-mod PATH       Path to go.mod file (default: ../go.mod)
    -f, --proto-file FILE   Download only specific proto file (e.g., product_availability.proto)
    -d, --dry-run          Show what would be done without executing
    --list-versions        List available versions for all repos and exit
    --single-repo          Process only the first repository found

Environment Variables:
    REPO_NAME              Repository name (overrides auto-detection)
    SOURCE_PATH_IN_REPO    Source path in repository
    BUF_YAML_PATH          Path to buf.yaml file
    GO_MOD_PATH            Path to go.mod file
    PROTO_FILE_NAME        Specific proto file to download

Examples:
    proto-sync                                          # Auto-detect and download from go.mod
    proto-sync --version v0.12.0 --single-repo        # Download specific version of first repo
    proto-sync --repo github.com/my-org/my-api         # Use specific repository
    proto-sync --proto-file product_availability.proto # Download only product_availability.proto
    proto-sync --dry-run                               # Preview what would be done
    proto-sync list-versions                           # List available versions for all repos`

	fmt.Println(usage)
}
