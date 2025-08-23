# Build and Release Script

This directory contains the build and release script for the AI Proxy application.

## build-release.sh

A comprehensive build script that handles:

1. Building the Go application binary
2. Building the Docker image with proper tagging
3. Supporting versioning (using git tags or manual version input)
4. Cleaning up temporary files after build
5. Including error handling and progress feedback

### Usage

```bash
# Auto-detect version from git
./scripts/build-release.sh

# Build with specific version
./scripts/build-release.sh -v 1.2.3

# Show help
./scripts/build-release.sh --help
```

### Features

- **Version Detection**: Automatically detects version from git tags or uses commit hash
- **Go Build**: Compiles the Go application with embedded version information
- **Docker Build**: Creates Docker images with proper tagging (if Docker is available)
- **Cleanup**: Automatically cleans up temporary files
- **Error Handling**: Comprehensive error handling with clear messages
- **Progress Feedback**: Colored output showing build progress
- **Cross-platform**: Works on Linux and macOS

### Versioning

The script supports multiple ways to determine the version:

1. **Manual Input**: Use `-v` or `--version` flag to specify a version
2. **Git Tags**: Automatically detects the latest git tag
3. **Commit Hash**: Uses commit hash when no tags are available

Version format examples:
- `1.2.3` (exact tag)
- `1.2.3-dev+abc1234` (commits ahead of tag)
- `0.0.0-dev+abc1234` (no tags found)