# Docker Configuration with Mounted Files

This guide explains how to use the Docker configuration with mounted files for the AI Proxy service. By mounting configuration files as volumes, you can easily modify settings without rebuilding the container.

## Table of Contents
1. [Creating Configuration Files](#creating-configuration-files)
2. [Running the Container with Volume Mounts](#running-the-container-with-volume-mounts)
3. [Modifying Configuration Files](#modifying-configuration-files)
4. [Verifying Changes](#verifying-changes)
5. [Best Practices](#best-practices)

## Creating Configuration Files

Before running the container, you need to create two configuration files: `.env` for environment variables and `provider-config.yaml` for provider settings.

### 1. Create the .env File

Create a `.env` file with your authentication token and port settings:

```bash
# Create a directory for your config files
mkdir -p ./ai-proxy-config

# Create the .env file
cat > ./ai-proxy-config/.env << EOF
# REQUIRED: Proxy access token for API access
AUTH_TOKEN="your_secure_token_here"

# Optional: Port to listen on (default: 8080)
PORT="8080"

# Optional: Retry configuration
RETRY_MAX_RETRIES=3
RETRY_BASE_DELAY=1s
RETRY_MAX_DELAY=30s
RETRY_JITTER=0.1
EOF
```

Replace `your_secure_token_here` with a secure token that clients will use to authenticate with your proxy.

### 2. Create the provider-config.yaml File

Create a `provider-config.yaml` file with your provider configurations:

```bash
# Create the provider-config.yaml file
cat > ./ai-proxy-config/provider-config.yaml << EOF
models:
  # Example Groq model
  - name: groq/llama-3.2-90b-vision-preview
    provider: groq
    priority: 1
    requests_per_minute: 10
    requests_per_hour: 100
    requests_per_day: 3500
    url: "https://api.groq.com/openai/v1/chat/completions"
    token: "your_groq_api_token"
    max_request_length: 128000
    model_size: BIG
    http_client_config:
      timeout_seconds: 30
      idle_conn_timeout_seconds: 90

  # Example OpenRouter model
  - name: openrouter/deepseek/deepseek-chat:free
    provider: openrouter
    priority: 1
    requests_per_minute: 20
    requests_per_hour: 100
    requests_per_day: 200
    url: "https://openrouter.ai/api/v1/chat/completions"
    token: "your_openrouter_api_token"
    max_request_length: 131072
    model_size: BIG
    http_client_config:
      timeout_seconds: 30
      idle_conn_timeout_seconds: 90
EOF
```

Replace the API tokens with your actual provider tokens.

## Running the Container with Volume Mounts

Once you've created your configuration files, you can run the container with volume mounts:

### Using Docker Run

```bash
# Build the Docker image (if not already built)
docker build -t ai-proxy -f scripts/Dockerfile .

# Run the container with volume mounts
docker run -d \
  --name ai-proxy \
  -p 8080:8080 \
  -v "$(pwd)/ai-proxy-config:/app/config" \
  ai-proxy
```

### Using Docker Compose

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
```

Then run with:

```bash
docker-compose up -d
```

## Modifying Configuration Files

To modify the configuration:

### 1. Edit the Configuration Files

You can edit the configuration files directly on your host system:

```bash
# Edit the .env file
nano ./ai-proxy-config/.env

# Edit the provider-config.yaml file
nano ./ai-proxy-config/provider-config.yaml
```

### 2. Restart the Container

After making changes, restart the container to apply them:

```bash
# With docker run
docker restart ai-proxy

# With docker-compose
docker-compose restart
```

Note: Some changes (like AUTH_TOKEN) may require a full container restart, while others might be hot-reloaded depending on implementation.

## Verifying Changes

To verify that your changes have taken effect:

### 1. Check Container Logs

```bash
# Check container logs
docker logs ai-proxy

# Follow logs in real-time
docker logs -f ai-proxy
```

Look for messages indicating successful loading of configuration files.

### 2. Test API Endpoints

Test that your proxy is working correctly:

```bash
# Get list of available models
curl -H "Authorization: Bearer your_secure_token_here" \
  http://localhost:8080/models

# Test a simple request (replace with your actual model)
curl -X POST http://localhost:8080/chat/completions \
  -H "Authorization: Bearer your_secure_token_here" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "groq/llama-3.2-90b-vision-preview",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'
```

### 3. Verify Configuration Values

Check that the configuration values are being used:

```bash
# Check if the port is correctly set
docker port ai-proxy

# Verify environment variables inside the container
docker exec ai-proxy env

# Check mounted files inside the container
docker exec ai-proxy ls -la /app/config
```

## Best Practices

1. **Secure Your Tokens**: Never commit your `.env` or `provider-config.yaml` files to version control. Add them to your `.gitignore` file.

2. **Use Strong Tokens**: Generate strong, random tokens for your `AUTH_TOKEN`.

3. **Backup Configurations**: Keep backups of your working configurations.

4. **Environment-Specific Configs**: Use different configuration files for development, staging, and production environments.

5. **Monitor Logs**: Regularly check container logs for errors or warnings.

6. **Update Regularly**: Keep your container updated with the latest image.

7. **Resource Limits**: Consider setting resource limits in your Docker Compose file:

```yaml
services:
  ai-proxy:
    # ... other settings ...
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
```

By following this guide, you can easily configure and manage your AI Proxy service using Docker with mounted configuration files.

## VPS Deployment Guide

For deploying the AI Proxy service to a VPS, you can use Podman with the provided Dockerfile and related scripts. This approach leverages the same containerized deployment model as Docker but uses Podman for running containers on your VPS.

### 1. Prerequisites

Before deploying to your VPS, ensure you have:

1. A VPS running a Linux distribution (Ubuntu/Debian/CentOS recommended)
2. Podman installed on your VPS:
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install -y podman
   
   # CentOS/RHEL
   sudo dnf install -y podman
   ```
3. The following files from the repository:
   - `scripts/Dockerfile` - Container image definition
   - `scripts/build-release.sh` - Build script (optional)

### 2. Preparing Configuration Files

Create your configuration files on the VPS:

```bash
# Create a directory for configuration files
mkdir -p ~/ai-proxy-config

# Create the .env file
cat > ~/ai-proxy-config/.env << EOF
# REQUIRED: Proxy access token for API access
AUTH_TOKEN="your_secure_token_here"

# Optional: Port to listen on (default: 8080)
PORT="8080"

# Optional: Retry configuration
RETRY_MAX_RETRIES=3
RETRY_BASE_DELAY=1s
RETRY_MAX_DELAY=30s
RETRY_JITTER=0.1
EOF

# Create the provider-config.yaml file
cat > ~/ai-proxy-config/provider-config.yaml << EOF
models:
  # Example Groq model
  - name: groq/llama-3.2-90b-vision-preview
    provider: groq
    priority: 1
    requests_per_minute: 10
    requests_per_hour: 100
    requests_per_day: 3500
    url: "https://api.groq.com/openai/v1/chat/completions"
    token: "your_groq_api_token"
    max_request_length: 128000
    model_size: BIG
    http_client_config:
      timeout_seconds: 30
      idle_conn_timeout_seconds: 90

  # Example OpenRouter model
  - name: openrouter/deepseek/deepseek-chat:free
    provider: openrouter
    priority: 1
    requests_per_minute: 20
    requests_per_hour: 100
    requests_per_day: 200
    url: "https://openrouter.ai/api/v1/chat/completions"
    token: "your_openrouter_api_token"
    max_request_length: 131072
    model_size: BIG
    http_client_config:
      timeout_seconds: 30
      idle_conn_timeout_seconds: 90
EOF
```

Replace the placeholder values with your actual configuration.

### 3. Building and Running with Podman

You can build and run the AI Proxy service using Podman in several ways:

#### Option 1: Using the build script

```bash
# Clone the repository (if not already done)
git clone <repository-url>
cd ai-proxy

# Build the binary and Docker image
./scripts/build-release.sh

# Run the container with Podman
podman run -d \
  --name ai-proxy \
  -p 8080:8080 \
  -v "$(pwd)/ai-proxy-config:/app/config:Z" \
  ai-proxy:latest
```

#### Option 2: Building manually

```bash
# Clone the repository (if not already done)
git clone <repository-url>
cd ai-proxy

# Build the Docker image
podman build -t ai-proxy -f scripts/Dockerfile .

# Run the container
podman run -d \
  --name ai-proxy \
  -p 8080:8080 \
  -v "$(pwd)/ai-proxy-config:/app/config:Z" \
  ai-proxy:latest
```

### 4. Managing the Service

You can manage the AI Proxy service using Podman commands:

```bash
# Check if the container is running
podman ps

# View container logs
podman logs ai-proxy

# Follow container logs in real-time
podman logs -f ai-proxy

# Stop the container
podman stop ai-proxy

# Start the container
podman start ai-proxy

# Restart the container
podman restart ai-proxy

# Remove the container
podman rm ai-proxy
```

### 5. Setting up Systemd Service (Optional)

For automatic startup on boot, you can create a systemd service that runs Podman:

```bash
# Create a systemd service file
sudo tee /etc/systemd/system/ai-proxy.service > /dev/null << EOF
[Unit]
Description=AI Proxy Service (Podman)
After=network.target

[Service]
Type=forking
User=user
Group=user
Environment=PODMAN_SYSTEMD_UNIT=1
Restart=on-failure
TimeoutStopSec=70
ExecStart=/usr/bin/podman run -d --name ai-proxy -p 8080:8080 -v /home/user/ai-proxy-config:/app/config:Z ai-proxy:latest
ExecStop=/usr/bin/podman stop -t 10 ai-proxy
ExecReload=/usr/bin/podman restart ai-proxy
KillMode=none

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd configuration
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable ai-proxy

# Start the service
sudo systemctl start ai-proxy

# Check service status
sudo systemctl status ai-proxy
```

### 6. Verifying Deployment

Test that your proxy is working correctly:

```bash
# Get list of available models
curl -H "Authorization: Bearer your_secure_token_here" \
  http://localhost:8080/models

# Test a simple request (replace with your actual model)
curl -X POST http://localhost:8080/chat/completions \
  -H "Authorization: Bearer your_secure_token_here" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "groq/llama-3.2-90b-vision-preview",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'
```

Your AI Proxy service should now be running on your VPS using Podman and accessible via the configured port.