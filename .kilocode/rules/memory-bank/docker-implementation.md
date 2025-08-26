# Docker Implementation

**Last Updated:** 2025-08-26T15:33:47.424Z

## Overview

This document describes the Docker implementation for the AI Proxy Server, including the single-stage Dockerfile and the automated installer script that handles production deployment with Docker/Podman support.

## Dockerfile Features

The optimized Dockerfile includes the following features:

1. **Single-stage Build**:
   - Optimized for pre-compiled binary deployment from `build-release.sh`
   - No Go compilation needed within the Dockerfile

2. **Minimal Base Image**: Uses Alpine Linux (latest) for a smaller footprint

3. **Security Best Practices**:
   - Runs as a non-root user (ai-proxy, UID 1001)
   - Sets proper file ownership and permissions
   - Minimal packages installed (only ca-certificates and essential tools)

4. **Configuration File Management**:
   - Defines a volume at `/app/config` for mounting host configuration files
   - Allows users to update configuration files without rebuilding

5. **Dynamic Port Configuration**:
   - Reads port configuration from the mounted `.env` file
   - Falls back to port 8080 if no port is specified
   - Documents that the port is determined at runtime from configuration

6. **Proper File Permissions**:
   - Sets ownership to non-root user
   - Makes binary executable

7. **Health Checks**:
   - The `installer.sh` script manages health checks externally to ensure compatibility with both Docker and Podman.
   - The Dockerfile itself does not include a `HEALTHCHECK` instruction, as this is handled at runtime.

## Automated Installation Script

The project includes an automated installer script (`scripts/installer.sh`) that handles production deployment with the following features:

1. **Container Runtime Detection**:
   - Automatically detects and uses either Docker or Podman
   - Provides clear error messages if neither is available

2. **Security Best Practices**:
   - Uses `--security-opt no-new-privileges` for container security
   - Sets secure permissions (640) on configuration files
   - Runs containers with restricted privileges

3. **Error Handling and Rollback**:
   - Comprehensive error handling with informative messages
   - Automatic rollback mechanism on installation failures
   - Health check validation after deployment

4. **Configuration Management**:
   - Automatically creates configuration directory (`/srv/ai-proxy`)
   - Copies template configuration files if they don't exist
   - Preserves existing configuration files
   - Sets proper ownership and permissions

5. **Deployment Management**:
   - Cleans up existing containers and images before deployment
   - Builds the latest image from pre-compiled artifacts
   - Runs container with proper restart policies
   - Performs health checks to verify successful deployment

## Usage Instructions

### Automated Installation (Recommended)

1. Run the installer script:
   ```bash
   sudo ./scripts/installer.sh
   ```

2. The script will:
   - Detect Docker or Podman
   - Set up configuration files in `/srv/ai-proxy`
   - Build and deploy the container
   - Perform health checks
   - Provide next steps for configuration

### Manual Docker Deployment

1. Build the image:
   ```
   docker build -t ai-proxy:latest -f scripts/Dockerfile .
   ```

2. Create configuration directory and files:
   ```bash
   mkdir -p ./ai-proxy-config
   cp scripts/.env.example ./ai-proxy-config/.env
   cp scripts/provider-config-example.yaml ./ai-proxy-config/provider-config.yaml
   ```

3. Edit configuration files:
   - Set your AUTH_TOKEN in `./ai-proxy-config/.env`
   - Configure your providers in `./ai-proxy-config/provider-config.yaml`

4. Run the container with mounted configuration files:
   ```
   docker run -d \
     --name ai-proxy \
     --restart unless-stopped \
     --security-opt no-new-privileges \
     -p 8080:8080 \
     -v "$(pwd)/ai-proxy-config:/app/config" \
     ai-proxy:latest
   ```

### Docker Compose Deployment

Create a `docker-compose.yml` file:
```yaml
version: '3.8'

services:
  ai-proxy:
    build:
      context: .
      dockerfile: scripts/Dockerfile
    container_name: ai-proxy
    ports:
      - "8080:8080"
    volumes:
      - ./ai-proxy-config:/app/config
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
```

Then run with:
```bash
docker-compose up -d
```

## Configuration File Management

### Initial Setup
The installer script automatically creates configuration files in `/srv/ai-proxy` with secure permissions.

### Manual Updates
To update configuration files without rebuilding:

1. Modify the configuration files on the host system:
   - Edit `/srv/ai-proxy/.env` or your custom config directory
   - Edit `/srv/ai-proxy/provider-config.yaml` or your custom config directory

2. Restart the container to apply changes:
   ```bash
   # With docker
   docker restart ai-proxy
   
   # With podman
   podman restart ai-proxy
   
   # With docker-compose
   docker-compose restart
   ```

## Port Configuration

The Dockerfile is designed to read the port configuration from the `.env` file:

1. During container startup, the application reads the `PORT` variable from the mounted `.env` file
2. If no port is specified in the `.env` file, the application defaults to port 8080
3. The container exposes the port specified in the configuration

This approach allows for flexible port configuration without requiring a rebuild of the Docker image.

## Health Checks

The `installer.sh` script performs health checks during deployment to ensure the service starts correctly. The Dockerfile does not contain a `HEALTHCHECK` instruction, as this is managed externally for better compatibility with container runtimes like Podman, which has different support for this feature.

## Security Features

1. **Container Security**:
   - Runs as non-root user (ai-proxy)
   - Uses `--security-opt no-new-privileges`
   - Minimal base image with only required packages

2. **Configuration Security**:
   - Secure file permissions (640) on sensitive configuration files
   - Separation of configuration from container image
   - Environment variable support for sensitive data

3. **Runtime Security**:
   - Proper error handling without information disclosure
   - Input validation and sanitization
   - Resource limits through container constraints

## Troubleshooting

### Common Issues

1. **Port Conflicts**:
   - Change the PORT variable in your `.env` file
   - Update the port mapping in your docker run command or docker-compose.yml

2. **Permission Issues**:
   - Ensure configuration files have proper permissions (640)
   - Verify the ai-proxy user can read the mounted files

3. **Health Check Failures**:
   - Check container logs: `docker logs ai-proxy`
   - Verify configuration files are correctly formatted
   - Ensure AUTH_TOKEN is properly set

### Log Management

View container logs:
```bash
# Docker
docker logs ai-proxy

# Follow logs in real-time
docker logs -f ai-proxy

# Podman
podman logs ai-proxy

# Follow logs in real-time
podman logs -f ai-proxy