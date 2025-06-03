package infrastructure

import (
	"fmt"

	"github.com/franouer/proto-sync/internal/domain"
	"gopkg.in/yaml.v3"
)

type BufRepositoryImpl struct {
	logger   domain.Logger
	fileRepo domain.FileRepository
}

// BufConfig represents the structure of buf.yaml
type BufConfig struct {
	Version string `yaml:"version"`
	Modules []struct {
		Path string `yaml:"path"`
		Name string `yaml:"name,omitempty"`
	} `yaml:"modules"`
}

// NewBufRepository creates a new buf repository
func NewBufRepository(logger domain.Logger, fileRepo domain.FileRepository) domain.BufRepository {
	return &BufRepositoryImpl{
		logger:   logger,
		fileRepo: fileRepo,
	}
}

func (b *BufRepositoryImpl) ParseBufYaml(bufYamlPath string) (*domain.ModuleInfo, error) {
	if !b.fileRepo.FileExists(bufYamlPath) {
		return nil, fmt.Errorf("buf.yaml file not found at: %s", bufYamlPath)
	}

	data, err := b.fileRepo.ReadFile(bufYamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read buf.yaml file: %w", err)
	}

	var config BufConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse buf.yaml: %w", err)
	}

	// Extract the first module path from buf.yaml
	if len(config.Modules) == 0 {
		return nil, fmt.Errorf("no modules found in %s", bufYamlPath)
	}

	module := config.Modules[0]
	if module.Path == "" {
		return nil, fmt.Errorf("module path is empty in %s", bufYamlPath)
	}

	return &domain.ModuleInfo{
		Name: module.Name,
		Path: module.Path,
	}, nil
}
