# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-12-28

### Added
- Initial release of proto-sync CLI tool
- Auto-detect protobuf libraries from go.mod files
- Download specific versions or latest versions of proto files
- Copy specific proto files or all proto files from repositories
- Dry-run mode for previewing actions without making changes
- Colorful logging output for better user experience
- List available versions for repositories
- Clean architecture implementation following Clean Architecture principles
- Support for manual repository specification
- Command-line interface built with Cobra
- Comprehensive documentation and usage examples

### Features
- `proto-sync` - Auto-detect and download from go.mod
- `proto-sync --version <version> --single-repo` - Download specific version
- `proto-sync --repo <repository>` - Use specific repository
- `proto-sync --proto-file <file>` - Download only specific proto file
- `proto-sync --dry-run` - Preview actions without executing
- `proto-sync --list-versions` - List available versions

[1.0.0]: https://github.com/Francouer/proto-sync/releases/tag/v1.0.0 