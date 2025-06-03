package domain

import "time"

// Repository represents a protobuf repository
type Repository struct {
	Name    string
	Version string
	URL     string
}

// ProtoFile represents a protobuf file
type ProtoFile struct {
	Name         string
	Path         string
	Size         int64
	ModifiedTime time.Time
}

// SyncConfig represents the configuration for syncing proto files
type SyncConfig struct {
	Repositories     []Repository
	SourcePath       string
	TargetPath       string
	BufYamlPath      string
	GoModPath        string
	SpecificFile     string
	DryRun           bool
	SingleRepo       bool
	ListVersions     bool
	SpecifiedVersion string
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	Repository   Repository
	FilesUpdated []ProtoFile
	Success      bool
	Error        error
}

// ModuleInfo represents information from buf.yaml
type ModuleInfo struct {
	Name string
	Path string
}

// GoModInfo represents protobuf libraries from go.mod
type GoModInfo struct {
	Repositories []Repository
	ModuleName   string
}
