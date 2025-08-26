#!/bin/bash

# AI Proxy Production Build and Release Script
# Creates ai-proxy-release.tar.gz artifact with precise structure for VPS deployment
# This script builds the Go application binary and packages it for deployment

set -euo pipefail  # Exit on error, undefined variables, pipe failures
IFS=$'\n\t'       # Secure Internal Field Separator

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Build configuration
readonly BUILD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly PKG_DIR="$BUILD_DIR/pkg"
readonly SCRIPTS_DIR="$BUILD_DIR/scripts"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

exit_with_error() {
    local exit_code="${2:-1}"
    log_error "$1"
    exit "$exit_code"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get version from git tag or commit hash
get_version() {
    local version_input="$1"
    
    # If version is provided as argument, use it
    if [[ -n "$version_input" ]]; then
        echo "$version_input"
        return
    fi
    
    # Try to get version from git tag
    if command_exists git && git rev-parse --git-dir > /dev/null 2>&1; then
        # Get the latest tag
        local git_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
        
        if [[ -n "$git_tag" ]]; then
            # Get commit count since the tag
            local commit_count=$(git rev-list --count "$git_tag"..HEAD 2>/dev/null || echo "0")
            
            if [[ "$commit_count" == "0" ]]; then
                # We're exactly at a tag
                echo "$git_tag"
                return
            else
                # We're some commits ahead of the tag
                local commit_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
                echo "${git_tag}-dev+${commit_hash}"
                return
            fi
        else
            # No tags found, use commit hash
            local commit_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
            echo "0.0-dev+${commit_hash}"
            return
        fi
    fi
    
    # Fallback version
    echo "0.0.0-unknown"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    if [[ "$exit_code" -ne 0 ]]; then
        log_error "Build failed. Cleaning up..."
    else
        log_info "Cleaning up temporary files..."
    fi
    
    # Remove temporary binary if it exists
    if [[ -f "$PKG_DIR/ai-proxy" ]]; then
        rm -f "$PKG_DIR/ai-proxy"
    fi
    
    if [[ "$exit_code" -ne 0 ]]; then
        log_error "Build cleanup completed"
    else
        log_success "Build cleanup completed"
    fi
}

# Build Go binary for Linux (production target)
build_binary() {
    local version="$1"
    local commit_hash=""
    local build_date=""
    
    log_info "Building Go binary for Linux production deployment..."
    
    # Get commit hash and build date
    if command_exists git && git rev-parse --git-dir > /dev/null 2>&1; then
        commit_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    else
        commit_hash="unknown"
    fi
    
    build_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    # Build the binary with version information and Linux target
    local ldflags="-X main.version=${version} -X main.commit=${commit_hash} -X main.date=${build_date} -s -w"
    
    # Create pkg directory if it doesn't exist
    mkdir -p "$PKG_DIR"
    
    # Build for Linux (production VPS target)
    if ! CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="$ldflags" \
        -o "$PKG_DIR/ai-proxy" \
        "$BUILD_DIR"; then
        exit_with_error "Failed to build Go binary for Linux" 5
    fi
    
    # Set executable permissions
    chmod +x "$PKG_DIR/ai-proxy"
    
    # Validate binary
    if [[ ! -f "$PKG_DIR/ai-proxy" ]]; then
        exit_with_error "Binary not created: $PKG_DIR/ai-proxy" 6
    fi
    
    local binary_size=$(du -h "$PKG_DIR/ai-proxy" | cut -f1)
    log_success "Go binary built successfully: $PKG_DIR/ai-proxy (${binary_size})"
}

# Function to create release artifact with precise structure
create_release_artifact() {
    local version="$1"
    local pkg_name="ai-proxy-${version}"
    local archive_path="$PKG_DIR/ai-proxy-release.tar.gz"
    
    log_info "Creating release artifact with precise structure..."
    
    # Create package directory
    local pkg_dir="$PKG_DIR/${pkg_name}"
    mkdir -p "${pkg_dir}"
    
    # Copy binary to package directory
    if [[ -f "$PKG_DIR/ai-proxy" ]]; then
        cp "$PKG_DIR/ai-proxy" "${pkg_dir}/"
        log_info "Copied binary to package directory"
    else
        exit_with_error "Binary not found. Please build the binary first." 3
    fi
    
    # Copy deployment files with production references
    cp -f "$SCRIPTS_DIR/.env.example" "${pkg_dir}/.env.example"
    cp -f "$SCRIPTS_DIR/provider-config-example.yaml" "${pkg_dir}/provider-config.yaml"
    cp -f "$SCRIPTS_DIR/Dockerfile" "${pkg_dir}/Dockerfile"
    cp -f "$BUILD_DIR/README.md" "${pkg_dir}/README.md"
    cp -f "$BUILD_DIR/LICENSE" "${pkg_dir}/LICENSE"
    
    log_info "Copied deployment files to package directory"
    
    # Create installation instructions
    cat > "${pkg_dir}/INSTALL.md" << EOF
# AI Proxy Installation Instructions

## Version: ${version}

## Quick Start

1. Extract the package:
   \`\`\`
   tar -xzf  ai-proxy-release.tar.gz
   cd ${pkg_name}
   \`\`\`

2. Configure the application:
   - Copy \`.env.example\` to \`.env\` and set your AUTH_TOKEN
   - Modify \`provider-config.yaml\` with your provider settings

3. Run with Docker:
   \`\`\`
   docker build -t ai-proxy:${version} .
   docker run -d -p 8080:8080 \\
     -v /srv/ai-proxy:/app/config \\
     ai-proxy:${version}
   \`\`\`

4. Or run directly:
   \`\`\`
   # Set environment variables
   export AUTH_TOKEN="your_token_here"
   export PORT="8080"
   
   # Run the application
   ./ai-proxy --config=provider-config.yaml --env-file=.env
   \`\`\`

## Configuration

- \`.env\`: Environment variables (AUTH_TOKEN, PORT)
- \`provider-config.yaml\`: Provider configuration

## Directories

- \`/app/config\`: Mount this directory to provide configuration files when using Docker

## Ports

- Default port: 8080 (can be changed with PORT environment variable)

## Requirements

- Docker (for containerized deployment)
- Or a compatible OS with the provided binary
EOF
    
    log_info "Created installation instructions"
    
    # Create tar.gz archive with precise name
    (cd "$PKG_DIR" && tar -czf "ai-proxy-release.tar.gz" "${pkg_name}")
    
    log_success "Created release artifact: $archive_path"
    
    # Show package info
    local package_size=$(du -h "$archive_path" | cut -f1)
    log_success "Archive size: ${package_size}"
    
    # Validate archive contents
    log_info "Validating archive contents..."
    if tar -tzf "$archive_path" | head -10; then
        log_success "Archive validation completed"
    else
        exit_with_error "Archive validation failed" 4
    fi
}

# Function to create installer script artifact
create_installer_script() {
    local installer_path="$PKG_DIR/installer.sh"
    
    log_info "Creating installer script artifact..."
    
    # Copy the production installer script
    if [[ -f "$SCRIPTS_DIR/installer.sh" ]]; then
        cp "$SCRIPTS_DIR/installer.sh" "$installer_path"
        chmod +x "$installer_path"
        log_success "Installer script created: $installer_path"
        
        # Validate installer script syntax
        if bash -n "$installer_path"; then
            log_success "Installer script syntax validated"
        else
            exit_with_error "Installer script syntax validation failed" 7
        fi
    else
        exit_with_error "Installer script not found at $SCRIPTS_DIR/installer.sh" 8
    fi
}

# Function to validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check if Go is installed
    if ! command_exists go; then
        exit_with_error "Go is not installed or not in PATH" 2
    fi
    
    # Check Go version (should be 1.24 or higher)
    local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | cut -d ' ' -f 1 | sed 's/go//')
    local major_version=$(echo "$go_version" | cut -d '.' -f 1)
    local minor_version=$(echo "$go_version" | cut -d '.' -f 2)
    
    if [[ "$major_version" -lt 1 ]] || [[ "$major_version" -eq 1 && "$minor_version" -lt 24 ]]; then
        log_warning "Go version 1.24 or higher is recommended. Current version: $go_version"
    else
        log_success "Go version: $go_version"
    fi
    
    # Check if we're in the correct directory
    if [[ ! -f "$BUILD_DIR/main.go" ]]; then
        exit_with_error "main.go not found. Please run this script from the project root directory." 3
    fi
    
    # Check if required production files exist
    local required_files=(
        "$SCRIPTS_DIR/Dockerfile"
        "$SCRIPTS_DIR/.env.example"
        "$SCRIPTS_DIR/provider-config-example.yaml"
        "$SCRIPTS_DIR/installer.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            exit_with_error "Required production file not found: $file" 4
        fi
    done
    
    log_success "Prerequisites validated"
}

# Display final deployment instructions
display_deployment_instructions() {
    local version="$1"
    local archive="$PKG_DIR/ai-proxy-release.tar.gz"
    local installer="$PKG_DIR/installer.sh"
    
    echo
    log_success "AI Proxy production build completed successfully!"
    echo
    log_info "Build Summary:"
    log_info "  - Version: $version"
    log_info "  - Binary: Linux/AMD64 production-ready"
    log_info "  - Archive: $(du -h "$archive" | cut -f1)"
    log_info " - Installer: $(du -h "$installer" | cut -f1)"
    echo
    log_info "VPS Deployment:"
    log_info "  1. Upload artifacts to target VPS:"
    log_info "     scp $archive $installer user@vps-host:/tmp/"
    log_info "  2. Run installer on VPS:"
    log_info "     cd /tmp && chmod +x installer.sh && sudo ./installer.sh"
    echo
    log_info "Required Artifacts:"
    log_info "  - $archive"
    log_info "  - $installer"
    echo
    log_info "The installer will handle:"
    log_info " - Interactive port configuration (1-65535)"
    log_info " - Extraction to /tmp/ai-proxy (isolated context)"
    log_info "  - Configuration setup in /srv/ai-proxy"
    log_info " - Atomic PORT updates in .env"
    log_info "  - Dynamic EXPOSE and HEALTHCHECK port modification"
    log_info "  - Production container deployment"
    log_info "  - Comprehensive health checks"
    log_info " - Error handling with rollback capability"
}

# Main function
main() {
    local version=""
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--version)
                version="$2"
                shift 2
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo "Build AI Proxy for VPS production deployment"
                echo ""
                echo "Options:"
                echo " -v, --version VERSION  Specify version (default: auto-detect from git)"
                echo " -h, --help             Show this help message"
                echo ""
                echo "Examples:"
                echo " $0                    # Auto-detect version from git"
                echo " $0 -v 1.2.3          # Build with specific version"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use -h or --help for usage information."
                exit 1
                ;;
        esac
    done
    
    # Print banner
    echo "================================================"
    echo "  AI Proxy Production Build Script"
    echo "  Target: Linux VPS Deployment"
    echo "================================================"
    echo
    
    # Set up error handling
    trap cleanup EXIT
    
    # Validate prerequisites
    validate_prerequisites
    
    # Get version
    version=$(get_version "$version")
    log_info "Build version: $version"
    
    # Create pkg directory
    mkdir -p "$PKG_DIR"
    
    # Build Go binary for Linux production
    build_binary "$version"
    
    # Create release artifact with precise structure
    create_release_artifact "$version"
    
    # Create installer script
    create_installer_script
    
    # Display deployment instructions
    display_deployment_instructions "$version"
}

# Run main function with all arguments
main "$@"