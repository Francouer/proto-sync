package infrastructure

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Francouer/proto-sync/internal/domain"
)

type FileRepositoryImpl struct {
	logger domain.Logger
}

// NewFileRepository creates a new file repository
func NewFileRepository(logger domain.Logger) domain.FileRepository {
	return &FileRepositoryImpl{
		logger: logger,
	}
}

func (f *FileRepositoryImpl) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *FileRepositoryImpl) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func (f *FileRepositoryImpl) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	if err := f.CreateDir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}

	return destFile.Sync()
}

func (f *FileRepositoryImpl) CreateDir(path string) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, 0o755)
}

func (f *FileRepositoryImpl) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *FileRepositoryImpl) ListFiles(dirPath string, pattern string) ([]domain.ProtoFile, error) {
	var files []domain.ProtoFile

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file matches pattern
		if pattern != "" {
			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}

		// For proto files, we typically want *.proto pattern
		if pattern == "" && !strings.HasSuffix(info.Name(), ".proto") {
			return nil
		}

		file := domain.ProtoFile{
			Name:         info.Name(),
			Path:         path,
			Size:         info.Size(),
			ModifiedTime: info.ModTime(),
		}

		files = append(files, file)
		return nil
	})

	return files, err
}

func (f *FileRepositoryImpl) MakeWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Add write permission to user
	mode := info.Mode()
	mode |= 0o200 // Add write permission for owner

	return os.Chmod(path, mode)
}
