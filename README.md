# Proto Sync fully generated repo

A flexible CLI tool to download and update proto files from remote repositories. Automatically detects protobuf libraries from go.mod or allows manual specification.

## Features

- Auto-detect protobuf libraries from go.mod
- Download specific versions or latest
- Copy specific proto files or all
- Dry-run mode for previewing actions
- Colorful logging output
- List available versions
- Clean architecture for easy extension

## Installation

```bash
go install github.com/Francouer/proto-sync@latest
```

## Usage

```bash
# Auto-detect and download from go.mod
proto-sync

# Download specific version of first repo
proto-sync --version v0.12.0 --single-repo

# Use specific repository
proto-sync --repo github.com/my-org/my-api

# Download only specific proto file
proto-sync --proto-file product_availability.proto

# Preview what would be done
proto-sync --dry-run

# List available versions for all repos
proto-sync --list-versions
```

## Architecture

This project follows Clean Architecture principles with clear separation of concerns:

- `cmd/` - CLI entry point
- `internal/app/` - Application use cases
- `internal/domain/` - Business entities and interfaces  
- `internal/infrastructure/` - External services (filesystem, HTTP, Go commands)
- `internal/interface/` - CLI handlers and adapters
