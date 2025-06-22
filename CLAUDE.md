# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running

```bash
# Build the binary with version information
make build

# Run the built binary
./bin/asset-watcher
```

### Testing and Quality Checks

```bash
# Format the code
make format

# Run linters (includes gofumpt, go vet, staticcheck, golangci-lint)
make lint

# Run all tests
make test

# Security scanning
make vuln

# Coverage analysis
make cov-unit      # Unit test coverage
make cov-integration  # Integration test coverage
```

### Version Management

```bash
# Bump version (creates git tag)
make bump_patch   # 1.0.0 -> 1.0.1
make bump_minor   # 1.0.0 -> 1.1.0
make bump_major   # 1.0.0 -> 2.0.0
```

## Architecture Overview

This is a Go CLI tool that fetches IP address assets from Google Cloud organizations. The architecture follows clean separation of concerns:

### Core Flow

1. **Configuration** (`config.go`) - Loads settings from environment variables
2. **Fetcher** (`fetcher.go`) - Wraps Google Asset API client, implements asset iteration
3. **Processor** (`processor.go`) - Filters assets based on project inclusion/exclusion and status
4. **Output** (`output.go`) - Formats results as table or JSON
5. **Logger** (`logger.go`) - Provides structured logging with Cloud Logging compatibility

### Key Design Patterns

- **Dependency Injection**: Logger and config are passed explicitly to all components
- **Interface-based Design**: `AssetFetcher` interface allows for easy testing and mocking
- **Iterator Pattern**: Assets are processed one-by-one using the Asset API's pagination
- **No Global State**: All dependencies are explicitly passed, making testing straightforward

### Testing Strategy

- All core components have corresponding test files (`*_test.go`)
- Table-driven tests for comprehensive coverage
- Mock implementations for external dependencies
- Special subprocess pattern for testing fatal errors (see `TestRunFatalCases` in `main_test.go`)

### Configuration

The tool is configured entirely through environment variables (see `config.go`):

- `ASSET_WATCHER_ORGANIZATION_ID` - Required GCP organization ID
- `ASSET_WATCHER_INCLUDED_PROJECTS` - Comma-separated list of projects to include
- `ASSET_WATCHER_EXCLUDED_PROJECTS` - Comma-separated list of projects to exclude
- `ASSET_WATCHER_EXCLUDED_STATUSES` - Comma-separated list of address statuses to exclude
- `ASSET_WATCHER_OUTPUT_FORMAT` - Output format (table or json)
- `ASSET_WATCHER_DEBUG` - Enable debug logging

### CI/CD Pipeline

GitHub Actions workflows handle:

- Building and testing on every push/PR
- Security scanning (gosec, govulncheck, gitleaks)
- Automated dependency updates via Dependabot
- Release automation with GoReleaser

When making changes, ensure you run `make format`, `make lint`, and `make test` before committing to catch any issues early.
