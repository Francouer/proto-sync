package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRepository(t *testing.T) {
	repo := Repository{
		Name:    "github.com/example/test-repo",
		Version: "v1.0.0",
		URL:     "https://github.com/example/test-repo",
	}

	assert.Equal(t, "github.com/example/test-repo", repo.Name)
	assert.Equal(t, "v1.0.0", repo.Version)
	assert.Equal(t, "https://github.com/example/test-repo", repo.URL)
}

func TestProtoFile(t *testing.T) {
	now := time.Now()
	file := ProtoFile{
		Name:         "test.proto",
		Path:         "/path/to/test.proto",
		Size:         1024,
		ModifiedTime: now,
	}

	assert.Equal(t, "test.proto", file.Name)
	assert.Equal(t, "/path/to/test.proto", file.Path)
	assert.Equal(t, int64(1024), file.Size)
	assert.Equal(t, now, file.ModifiedTime)
}

func TestSyncConfig(t *testing.T) {
	config := SyncConfig{
		SourcePath:       "schemas/api/v1",
		TargetPath:       "proto",
		BufYamlPath:      "buf.yaml",
		GoModPath:        "../go.mod",
		SpecificFile:     "test.proto",
		DryRun:           true,
		SingleRepo:       false,
		ListVersions:     false,
		SpecifiedVersion: "v1.0.0",
	}

	assert.Equal(t, "schemas/api/v1", config.SourcePath)
	assert.Equal(t, "proto", config.TargetPath)
	assert.True(t, config.DryRun)
	assert.False(t, config.SingleRepo)
	assert.Equal(t, "v1.0.0", config.SpecifiedVersion)
}

func TestSyncResult(t *testing.T) {
	repo := Repository{Name: "test-repo", Version: "v1.0.0"}
	file := ProtoFile{Name: "test.proto", Path: "/test.proto"}

	result := SyncResult{
		Repository:   repo,
		FilesUpdated: []ProtoFile{file},
		Success:      true,
		Error:        nil,
	}

	assert.Equal(t, repo, result.Repository)
	assert.Len(t, result.FilesUpdated, 1)
	assert.Equal(t, file, result.FilesUpdated[0])
	assert.True(t, result.Success)
	assert.NoError(t, result.Error)
}
