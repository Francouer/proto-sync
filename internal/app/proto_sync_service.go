package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Francouer/proto-sync/internal/domain"
)

type ProtoSyncServiceImpl struct {
	logger    domain.Logger
	fileRepo  domain.FileRepository
	goModRepo domain.GoModRepository
	bufRepo   domain.BufRepository
}

// NewProtoSyncService creates a new proto sync service
func NewProtoSyncService(
	logger domain.Logger,
	fileRepo domain.FileRepository,
	goModRepo domain.GoModRepository,
	bufRepo domain.BufRepository,
) domain.ProtoSyncService {
	return &ProtoSyncServiceImpl{
		logger:    logger,
		fileRepo:  fileRepo,
		goModRepo: goModRepo,
		bufRepo:   bufRepo,
	}
}

func (p *ProtoSyncServiceImpl) ValidateConfig(config *domain.SyncConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.BufYamlPath == "" {
		return fmt.Errorf("buf.yaml path is required")
	}

	if config.GoModPath == "" {
		return fmt.Errorf("go.mod path is required")
	}

	if config.SourcePath == "" {
		return fmt.Errorf("source path is required")
	}

	// Check if required files exist
	if !p.fileRepo.FileExists(config.BufYamlPath) {
		return fmt.Errorf("buf.yaml file not found at: %s", config.BufYamlPath)
	}

	if len(config.Repositories) == 0 && !p.fileRepo.FileExists(config.GoModPath) {
		return fmt.Errorf("go.mod file not found at: %s", config.GoModPath)
	}

	return nil
}

func (p *ProtoSyncServiceImpl) Sync(ctx context.Context, config *domain.SyncConfig) ([]domain.SyncResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Get target path from buf.yaml
	moduleInfo, err := p.bufRepo.ParseBufYaml(config.BufYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse buf.yaml: %w", err)
	}

	config.TargetPath = moduleInfo.Path
	p.logger.Info("Target path from %s: %s", config.BufYamlPath, config.TargetPath)

	// Determine repositories to process
	repositories := config.Repositories
	if len(repositories) == 0 {
		p.logger.Info("Auto-detecting protobuf libraries from %s...", config.GoModPath)
		goModInfo, err := p.goModRepo.ParseProtobufLibraries(config.GoModPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse go.mod: %w", err)
		}
		repositories = goModInfo.Repositories
	}

	// Override version if specified
	if config.SpecifiedVersion != "" {
		for i := range repositories {
			repositories[i].Version = config.SpecifiedVersion
		}
	}

	// Process single repo if requested
	if config.SingleRepo && len(repositories) > 1 {
		p.logger.Info("Single repo mode: processing only the first repository")
		repositories = repositories[:1]
	}

	p.logger.Info("Processing %d repository(ies)...", len(repositories))

	var results []domain.SyncResult
	for _, repo := range repositories {
		result := p.processRepository(ctx, repo, config)
		results = append(results, result)

		if !config.DryRun && result.Error != nil {
			p.logger.Error("Failed to process repository %s: %v", repo.Name, result.Error)
		}
	}

	if !config.DryRun {
		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}

		if successCount == len(results) {
			p.logger.Success("All proto files updated successfully!")
			p.logger.Info("You may want to run 'buf generate' to regenerate code from the updated protos")
		} else {
			p.logger.Warning("%d out of %d repositories processed successfully", successCount, len(results))
		}
	}

	return results, nil
}

func (p *ProtoSyncServiceImpl) processRepository(ctx context.Context, repo domain.Repository, config *domain.SyncConfig) domain.SyncResult {
	result := domain.SyncResult{
		Repository: repo,
		Success:    false,
	}

	p.logger.Info("Processing repository: %s", repo.Name)

	if config.DryRun {
		return p.dryRunRepository(repo, config)
	}

	// Download the module
	if err := p.goModRepo.DownloadModule(ctx, repo.Name, repo.Version); err != nil {
		result.Error = fmt.Errorf("failed to download module: %w", err)
		return result
	}

	// Get module path
	modulePath, err := p.goModRepo.GetModulePath(repo.Name, repo.Version)
	if err != nil {
		result.Error = fmt.Errorf("failed to get module path: %w", err)
		return result
	}

	sourcePath := filepath.Join(modulePath, config.SourcePath)
	if !p.fileRepo.FileExists(sourcePath) {
		result.Error = fmt.Errorf("source directory not found: %s", sourcePath)
		return result
	}

	// Create target directory if it doesn't exist
	if !p.fileRepo.FileExists(config.TargetPath) {
		p.logger.Info("Creating target directory: %s", config.TargetPath)
		if err := p.fileRepo.CreateDir(config.TargetPath); err != nil {
			result.Error = fmt.Errorf("failed to create target directory: %w", err)
			return result
		}
	}

	// Copy proto files
	if config.SpecificFile != "" {
		file, err := p.copySpecificFile(sourcePath, config.TargetPath, config.SpecificFile)
		if err != nil {
			result.Error = err
			return result
		}
		result.FilesUpdated = []domain.ProtoFile{file}
	} else {
		files, err := p.copyAllProtoFiles(sourcePath, config.TargetPath)
		if err != nil {
			result.Error = err
			return result
		}
		result.FilesUpdated = files
	}

	result.Success = true
	return result
}

func (p *ProtoSyncServiceImpl) dryRunRepository(repo domain.Repository, config *domain.SyncConfig) domain.SyncResult {
	result := domain.SyncResult{
		Repository: repo,
		Success:    true,
	}

	p.logger.Info("DRY RUN MODE - Actions that would be performed:")
	fmt.Printf("  1. Download: go mod download %s@%s\n", repo.Name, repo.Version)

	modulePath, err := p.goModRepo.GetModulePath(repo.Name, repo.Version)
	if err != nil {
		fmt.Printf("  2. Error getting module path: %v\n", err)
		return result
	}

	sourcePath := filepath.Join(modulePath, config.SourcePath)
	fmt.Printf("  2. Source directory: %s\n", sourcePath)
	fmt.Printf("  3. Target directory: %s\n", config.TargetPath)

	if p.fileRepo.FileExists(sourcePath) {
		if config.SpecificFile != "" {
			fmt.Printf("  4. Specific proto file that would be copied:\n")
			if p.fileRepo.FileExists(filepath.Join(sourcePath, config.SpecificFile)) {
				fmt.Printf("     - %s\n", config.SpecificFile)
			} else {
				fmt.Printf("     - %s (NOT FOUND - would fail)\n", config.SpecificFile)
			}
		} else {
			fmt.Printf("  4. Proto files that would be copied:\n")
			files, err := p.fileRepo.ListFiles(sourcePath, "*.proto")
			if err != nil {
				fmt.Printf("     Error listing files: %v\n", err)
			} else {
				for _, file := range files {
					fmt.Printf("     - %s\n", file.Name)
				}
			}
		}
	} else {
		fmt.Printf("  4. Source directory does not exist yet (would be created by download)\n")
	}

	return result
}

func (p *ProtoSyncServiceImpl) copySpecificFile(sourcePath, targetPath, fileName string) (domain.ProtoFile, error) {
	sourceFile := filepath.Join(sourcePath, fileName)
	targetFile := filepath.Join(targetPath, fileName)

	if !p.fileRepo.FileExists(sourceFile) {
		// List available files for user reference
		availableFiles, _ := p.fileRepo.ListFiles(sourcePath, "*.proto")
		fileNames := make([]string, len(availableFiles))
		for i, file := range availableFiles {
			fileNames[i] = file.Name
		}

		return domain.ProtoFile{}, fmt.Errorf("specific proto file not found: %s\nAvailable proto files: %s",
			sourceFile, strings.Join(fileNames, ", "))
	}

	// Make target file writable if it exists
	if p.fileRepo.FileExists(targetFile) {
		if err := p.fileRepo.MakeWritable(targetFile); err != nil {
			return domain.ProtoFile{}, fmt.Errorf("failed to make target file writable: %w", err)
		}
	}

	p.logger.Info("Copying specific proto file: %s", fileName)
	if err := p.fileRepo.CopyFile(sourceFile, targetFile); err != nil {
		return domain.ProtoFile{}, fmt.Errorf("failed to copy file: %w", err)
	}

	p.logger.Success("Successfully copied proto file: %s", fileName)

	return domain.ProtoFile{
		Name: fileName,
		Path: targetFile,
	}, nil
}

func (p *ProtoSyncServiceImpl) copyAllProtoFiles(sourcePath, targetPath string) ([]domain.ProtoFile, error) {
	sourceFiles, err := p.fileRepo.ListFiles(sourcePath, "*.proto")
	if err != nil {
		return nil, fmt.Errorf("failed to list proto files: %w", err)
	}

	if len(sourceFiles) == 0 {
		p.logger.Warning("No .proto files found in %s", sourcePath)
		return []domain.ProtoFile{}, nil
	}

	p.logger.Info("Copying %d proto file(s) from %s to %s...", len(sourceFiles), sourcePath, targetPath)

	// Make all existing proto files writable before copying
	existingFiles, _ := p.fileRepo.ListFiles(targetPath, "*.proto")
	for _, file := range existingFiles {
		if err := p.fileRepo.MakeWritable(file.Path); err != nil {
			p.logger.Warning("Failed to make file writable: %s", file.Path)
		}
	}

	var copiedFiles []domain.ProtoFile
	for _, sourceFile := range sourceFiles {
		targetFile := filepath.Join(targetPath, sourceFile.Name)
		if err := p.fileRepo.CopyFile(sourceFile.Path, targetFile); err != nil {
			return copiedFiles, fmt.Errorf("failed to copy %s: %w", sourceFile.Name, err)
		}

		copiedFiles = append(copiedFiles, domain.ProtoFile{
			Name: sourceFile.Name,
			Path: targetFile,
		})
	}

	p.logger.Success("Successfully copied proto files:")
	for _, file := range copiedFiles {
		fmt.Printf("  - %s\n", file.Name)
	}

	return copiedFiles, nil
}

func (p *ProtoSyncServiceImpl) ListVersions(ctx context.Context, repositories []domain.Repository) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, repo := range repositories {
		versions, err := p.goModRepo.ListVersions(repo.Name)
		if err != nil {
			p.logger.Error("Failed to list versions for %s: %v", repo.Name, err)
			continue
		}
		result[repo.Name] = versions
	}

	return result, nil
}
