# Architecture Documentation

## Overview

This project follows **Clean Architecture** principles to create a maintainable, testable, and extensible CLI application for syncing protobuf files. The architecture promotes separation of concerns and dependency inversion.

## Project Structure

```
proto-sync/
├── cmd/
│   └── proto-sync/           # CLI entry point
│       └── main.go          # Main application with dependency injection
├── internal/
│   ├── domain/              # Business entities and interfaces (inner layer)
│   │   ├── entities.go      # Core business objects
│   │   ├── interfaces.go    # Contracts for external dependencies
│   │   └── entities_test.go # Unit tests
│   ├── app/                 # Application use cases (application layer)
│   │   └── proto_sync_service.go # Main business logic orchestration
│   ├── infrastructure/      # External concerns (outer layer)
│   │   ├── logger.go        # Colorful logging implementation
│   │   ├── file_repository.go    # File system operations
│   │   ├── gomod_repository.go   # Go module operations
│   │   └── buf_repository.go     # Buf.yaml parsing
│   └── interface/           # Adapters (interface layer)
│       └── cli.go          # CLI command handling
├── examples/               # Example configuration files
│   ├── buf.yaml           # Sample buf configuration
│   └── go.mod             # Sample go.mod with protobuf libraries
├── build/                 # Build artifacts
├── go.mod                 # Module dependencies
├── go.sum                 # Dependency checksums
├── Makefile              # Build automation
├── README.md             # Project documentation
└── ARCHITECTURE.md       # This file
```

## Clean Architecture Layers

### 1. Domain Layer (`internal/domain/`)
**Innermost layer - No dependencies on external packages**

- **Entities**: Core business objects (`Repository`, `ProtoFile`, `SyncConfig`, etc.)
- **Interfaces**: Contracts that outer layers must implement
- **Rules**: Business logic that doesn't depend on external systems

**Key principles:**
- No dependencies on frameworks or external libraries
- Pure business logic
- Defines interfaces that outer layers implement

### 2. Application Layer (`internal/app/`)
**Use cases and application-specific business rules**

- **ProtoSyncService**: Orchestrates the main sync workflow
- **Validation**: Configuration validation logic
- **Coordination**: Coordinates between different repositories

**Key principles:**
- Depends only on Domain layer
- Implements business use cases
- Framework-independent business logic

### 3. Interface Adapters (`internal/interface/`)
**Converts data between use cases and external systems**

- **CLI Handler**: Converts CLI input to domain objects
- **Command Processing**: Handles CLI flags and commands
- **Output Formatting**: Formats results for display

**Key principles:**
- Adapts external interfaces to internal use cases
- Handles input/output conversion
- Framework-specific but isolated

### 4. Infrastructure Layer (`internal/infrastructure/`)
**Frameworks, drivers, and external systems**

- **File Repository**: File system operations
- **Go Module Repository**: Go module download and management
- **Buf Repository**: buf.yaml parsing
- **Logger**: Colorful console logging

**Key principles:**
- Implements interfaces defined in Domain layer
- Contains all framework and system dependencies
- Easily replaceable implementations

## Dependency Flow

```
CLI → Interface → Application → Domain ← Infrastructure
```

- **Dependency Rule**: Dependencies point inward only
- **Interface Segregation**: Small, focused interfaces
- **Dependency Inversion**: High-level modules don't depend on low-level modules

## Key Design Patterns

### 1. Repository Pattern
Each external system (files, go modules, buf.yaml) has its own repository interface:
```go
type FileRepository interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
    // ...
}
```

### 2. Dependency Injection
All dependencies are injected in `main.go`:
```go
// Initialize dependencies
logger := infrastructure.NewColorLogger()
fileRepo := infrastructure.NewFileRepository(logger)
service := app.NewProtoSyncService(logger, fileRepo, ...)
```

### 3. Command Pattern
CLI commands are handled through the Command pattern using Cobra.

## Benefits of This Architecture

### 1. **Testability**
- Business logic can be unit tested without external dependencies
- Infrastructure can be mocked easily
- Clear separation of concerns

### 2. **Maintainability**
- Changes to external systems don't affect business logic
- Each layer has a single responsibility
- Easy to understand and modify

### 3. **Extensibility**
- New features can be added without changing existing code
- New external systems can be integrated easily
- Different implementations can be swapped

### 4. **Framework Independence**
- Business logic doesn't depend on CLI framework
- Can easily add web API, gRPC, or other interfaces
- External libraries can be replaced

## Extension Points

### Adding New Commands
1. Add new method to `CLIHandler`
2. Create new command in `CreateRootCommand()`
3. Implement business logic in application layer

### Adding New Data Sources
1. Define interface in `domain/interfaces.go`
2. Implement in `infrastructure/`
3. Inject in `main.go`

### Adding New Output Formats
1. Extend `Logger` interface if needed
2. Create new implementation in infrastructure
3. Inject appropriate logger

### Adding New Protocols
1. Create new repository interface
2. Implement in infrastructure layer
3. Update application service to use new repository

## Testing Strategy

### Unit Tests
- Domain entities (pure business objects)
- Application services (mocked dependencies)
- Infrastructure components (isolated)

### Integration Tests
- End-to-end CLI functionality
- Real file system operations
- Actual go module downloads

### Example Test Structure
```go
func TestProtoSyncService_Sync(t *testing.T) {
    // Arrange
    mockFileRepo := &MockFileRepository{}
    mockGoModRepo := &MockGoModRepository{}
    service := NewProtoSyncService(logger, mockFileRepo, mockGoModRepo, ...)
    
    // Act
    result, err := service.Sync(ctx, config)
    
    // Assert
    assert.NoError(t, err)
    assert.True(t, result.Success)
}
```

This architecture ensures the codebase remains maintainable, testable, and easy to extend as requirements evolve. 