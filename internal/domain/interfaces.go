package domain

import "context"

// Logger defines the logging interface
type Logger interface {
	Info(msg string, args ...interface{})
	Success(msg string, args ...interface{})
	Warning(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// FileRepository handles file system operations
type FileRepository interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	CopyFile(src, dst string) error
	CreateDir(path string) error
	FileExists(path string) bool
	ListFiles(path string, pattern string) ([]ProtoFile, error)
	MakeWritable(path string) error
}

// GoModRepository handles go.mod operations
type GoModRepository interface {
	ParseProtobufLibraries(goModPath string) (*GoModInfo, error)
	GetLatestVersion(repo string) (string, error)
	ListVersions(repo string) ([]string, error)
	DownloadModule(ctx context.Context, repo, version string) error
	GetModulePath(repo, version string) (string, error)
}

// BufRepository handles buf.yaml operations
type BufRepository interface {
	ParseBufYaml(bufYamlPath string) (*ModuleInfo, error)
}

// ProtoSyncService defines the main service interface
type ProtoSyncService interface {
	Sync(ctx context.Context, config *SyncConfig) ([]SyncResult, error)
	ListVersions(ctx context.Context, repositories []Repository) (map[string][]string, error)
	ValidateConfig(config *SyncConfig) error
}
